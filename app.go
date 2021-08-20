package sf

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
)

type Application struct {
	Name        string
	Description string
	Version     string
	Routes      []Route
	DbManager   *DbManager
	// @private
	env          string
	configSource string
	router       *AppRouter
	initialized  bool
	factory      func(app *Application) error
	globals      map[string]interface{}
}

func CreateApp(name string, version, desc string, factory func(app *Application) error) *Application {
	app := &Application{Name: name, Version: version, Description: desc, env: os.Getenv("ENV")}
	app.factory = factory
	return app
}

func (app Application) IsProd() bool {
	return app.env == "prod"
}

func (app Application) IsTestEnv() bool {
	return app.env == "test"
}

type AppRouter struct {
	engine *gin.Engine
	app    *Application
}

type Route struct {
	Method         string
	Path           string
	Paths          []string
	Secure         bool
	TenantRequired bool
	Handler        HandlerFunc
}

type RequestContext struct {
	Application *Application
	TenantId    string
	HasTenant   bool
}

type Request struct {
	gin     *gin.Context
	Context RequestContext
}

type Validations struct {
	gin *gin.Context
}

type Response struct {
	gin *gin.Context
}

type RouterConfig struct {
	Secure bool
}

type HandlerFunc func(req Request, res Response)

func (r Response) OK(body interface{}) {
	r.gin.JSON(http.StatusOK, body)
}

func (r Response) JSON(status int, body interface{}) {
	r.gin.JSON(status, body)
}

func (r Response) BadRequest(body interface{}) {
	r.gin.JSON(http.StatusBadRequest, body)
}

func (r Response) Send(res interface{}, err error) {
	if err != nil {
		switch t := err.(type) {
		default:
			r.gin.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		case TechnicalError:
			r.gin.JSON(http.StatusBadRequest, gin.H{
				"code":    t.Code,
				"message": t.Message,
			})
		case UnauthorizedError:
			r.gin.JSON(http.StatusUnauthorized, gin.H{
				"code":    t.Code,
				"message": t.Message,
			})
		case FunctionalError:
			r.gin.JSON(http.StatusBadRequest, gin.H{
				"code":    t.Code,
				"message": t.Message,
			})
		}
		log.Error(err)
	} else {
		r.OK(res)
	}
}

func (r Request) Header(name string) string {
	return r.gin.GetHeader(name)
}

func getKongConsumer(ctx *gin.Context) *KongConsumerInfo {
	id := ctx.GetHeader("X-Consumer-ID")
	if IsStrEmpty(id) {
		return nil
	}
	return &KongConsumerInfo{
		Id:                   id,
		CustomId:             ctx.GetHeader("X-Consumer-Custom-ID"),
		Username:             ctx.GetHeader("X-Consumer-Username"),
		CredentialIdentifier: ctx.GetHeader("X-Credential-Identifier"),
		Anonymous:            ctx.GetHeader("X-Anonymous-Consumer") == "true",
	}
}

func (r Request) BindJson(dest interface{}) bool {
	if err := r.gin.ShouldBind(dest); err != nil {
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), err)
		r.gin.JSON(http.StatusBadRequest, gin.H{
			"code":  "validation.error",
			"error": err.Error(),
		})
		return false
	}
	return true
}

func (r Request) BindUri(dest interface{}) bool {
	if err := r.gin.ShouldBindUri(dest); err != nil {
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), err)
		r.gin.JSON(http.StatusBadRequest, gin.H{
			"code":  "validation.error",
			"error": err.Error(),
		})
		return false
	}
	return true
}

func (r Request) Validations() Validations {
	return Validations{gin: r.gin}
}

func (r Request) CheckInputWithRegex(value string, pattern string, errorCode string) bool {
	found, err := regexp.MatchString(pattern, value)
	if !found || err != nil {
		message := ""
		if err != nil {
			message = err.Error()
		}
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), fmt.Errorf(message))
		r.gin.JSON(http.StatusBadRequest, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return false
	}
	return true
}

func (r Request) RequireBasicAuth() (Credentials, bool) {
	user, password, hasAuth := r.gin.Request.BasicAuth()
	if !hasAuth || IsStrEmpty(user) {
		_ = Capture("http.request.unauthorized", fmt.Errorf(r.gin.Request.RequestURI))
		r.gin.JSON(http.StatusUnauthorized, gin.H{
			"message": "Missing credentials",
		})
		return Credentials{}, false
	}
	return Credentials{Username: user, Password: password}, true
}

func (r Request) Param(name string) string {
	return r.gin.Param(name)
}

func (r Request) RequireParam(name string) string {
	value := r.gin.Param(name)
	if IsStrEmpty(value) {

		message := fmt.Sprintf("Parameter '%s' is missing", name)
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), fmt.Errorf(message))

		r.gin.AbortWithStatusJSON(http.StatusBadRequest, H{
			"message": message,
		})
	}
	return value
}

func (r Request) GetArg(key string) interface{} {
	return r.Context.Application.GetArg(key)
}

func securityFilter(gc *gin.Context) {
	h := gc.GetHeader("X-Anonymous-Consumer")
	if "true" == strings.ToLower(h) {
		message := "anonymous access to this resource is forbidden"
		_ = Capture(fmt.Sprintf("http.guest.access.forbidden:%s", gc.Request.RequestURI), fmt.Errorf(message))
		gc.AbortWithStatusJSON(403, H{
			"message": message,
		})
	}
}

func healthechkHandler(_ Request, res Response) {
	//TODO: check database, amqp, vault
	res.OK(HealthCheck{Status: "UP"})
}

func initLogging() {
	log.SetOutput(os.Stdout)
	logLevel := Getenv("LOG_LEVEL", "DEBUG", true)
	if logLevel == "TRACE" {
		log.SetLevel(log.TraceLevel)
	} else if logLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else if logLevel == "WARN" {
		log.SetLevel(log.WarnLevel)
	} else if logLevel == "ERROR" {
		log.SetLevel(log.ErrorLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func (app *Application) Init(env string) {
	app.InitWithSource(env, "")
}

func (app *Application) InitWithSource(env string, source string) {
	if app.initialized {
		return
	}

	app.env = strings.ToLower(env)
	app.configSource = source
	filenames := []string{fmt.Sprintf(".env.%s", app.env), ".env"}

	for _, f := range filenames {
		if err := godotenv.Load(f); err != nil {
			log.Debug(err)
		}
	}
	if app.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	app.globals = map[string]interface{}{}

	if err := Capture("app.bootstrap", app.factory(app)); err != nil {
		log.Fatal(err)
	}

	app.initialized = true
}

func (app *Application) SetArg(key string, value interface{}) {
	app.globals[key] = value
}

func (app *Application) GetArg(key string) interface{} {
	return app.globals[key]
}

func (app *Application) LoadConfig(dest interface{}) {

	if err := env.Parse(dest); err != nil {
		log.Warn(err)
	}

	if !IsStrEmpty(app.configSource) {
		if strings.HasPrefix(app.configSource, "vault:") {
			log.Infof("Loading config from vault: %s", app.configSource)
			if err := ReadVaultSecret(strings.TrimPrefix(app.configSource, "vault:"), dest); err != nil {
				log.Fatalf("Error starting service, failed to read secrets from vault.\n%v", err)
			}
		} else {
			log.Warnf("configLocation not supported: %s", app.configSource)
		}
	}
}

func (app *Application) createRouter() {
	if app.router != nil {
		return
	}
	r := gin.Default()

	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))

	app.router = &AppRouter{
		engine: r,
		app:    app,
	}

	app.router.addRoute(Route{
		Method:  "GET",
		Paths:   []string{"/status", "/healthz"},
		Secure:  false,
		Handler: healthechkHandler,
	})

	for _, r := range app.Routes {
		app.router.addRoute(r)
	}
}

func (router AppRouter) addRoute(r Route) AppRouter {
	var paths []string
	if !IsStrEmpty(r.Path) {
		paths = append(paths, r.Path)
	}
	if len(r.Paths) > 0 {
		paths = append(paths, r.Paths[:]...)
	}
	for _, path := range paths {
		router.engine.Handle(r.Method, path, func(gc *gin.Context) {
			if r.Secure {
				securityFilter(gc)
			}
			context := RequestContext{Application: router.app}
			consumer := getKongConsumer(gc)
			if consumer != nil {
				context.TenantId = consumer.Id
				context.HasTenant = true
			} else {
				context.HasTenant = false
			}
			if r.TenantRequired && !context.HasTenant {
				gc.AbortWithStatusJSON(403, H{
					"error":   "missing.tenant",
					"message": "No tenant was provided",
				})
			}
			r.Handler(Request{gin: gc, Context: context}, Response{gin: gc})
		})
	}
	return router
}

func (app *Application) NewTestServer() *httptest.Server {
	app.createRouter()
	return httptest.NewServer(app.router.engine)
}

func (app *Application) ApplyDatabaseMigrations() error {
	if !app.initialized {
		return fmt.Errorf("application is not initialized, call app.Init(env) first")
	}
	if app.DbManager == nil {
		return nil
	}

	return Capture("database.migration", app.DbManager.migrate())
}

func (app Application) Execute() {
	cobra.OnInitialize()
	var rootCmd = &cobra.Command{
		Use:     app.Name,
		Short:   app.Description,
		Version: app.Version,
	}
	rootCmd.AddCommand(app.createServerCmd())
	rootCmd.AddCommand(app.createDbCommand())
	_ = rootCmd.Execute()
}

func (app *Application) createServerCmd() *cobra.Command {
	var port int
	var configSource string
	var dbMigrations bool
	var envName string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the service in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			app.InitWithSource(envName, configSource)
			if dbMigrations {
				if err := app.ApplyDatabaseMigrations(); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Info("database migrations were skipped")
			}
			app.Start(port)
		},
	}
	cmd.Flags().StringVarP(&envName, "env", "e", os.Getenv("ENV"), "active environment profile")
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().IntVarP(&port, "port", "p", Getenvi("PORT", 8080), "server port")
	cmd.Flags().BoolVarP(&dbMigrations, "db-migrations", "m", Getenvb("DB_MIGRATIONS", true), "apply database migrations")

	return cmd
}

func (app *Application) createDbCommand() *cobra.Command {
	var configSource string
	var envName string

	cmd := &cobra.Command{
		Use:   "db:migrate",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			app.InitWithSource(envName, configSource)
			if err := app.ApplyDatabaseMigrations(); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().StringVarP(&envName, "env", "e", Getenv(os.Getenv("ENV"), "dev", true), "active environment profile")

	return cmd
}

func (app *Application) Start(port int) {
	app.createRouter()
	_ = app.router.engine.Run(fmt.Sprintf(":%d", port))
}

func init() {
	initLogging()
}

func (rc RequestContext) PrimaryDbLink() DbLink {
	return rc.Application.DbManager.GetPrimaryLink()
}
func (rc RequestContext) DbLink(name string) DbLink {
	return rc.Application.DbManager.GetLink(name)
}

func (rc RequestContext) WithDbLink(id string, cb DbLinkCallback) error {
	dbm := rc.Application.DbManager
	return dbm.withLink(id, cb)
}

func (rc RequestContext) WithTenantDbLink(cb DbLinkCallback) error {
	return rc.WithDbLink(rc.TenantId, cb)
}
