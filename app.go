package sf

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/soffa-io/soffa-core-go/log"
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
	routes      []Route

	// @private
	//HealthChecks []appCheck
	env              string
	configSource     string
	Router           *AppRouter
	initialized      bool
	healthChecks     []ServiceCheck
	factory          func(app *Application)
	globals          map[string]interface{}
	startupListeners []func()
	messageBroker    *MessageBroker
	dataSources      []*DataSource
}

type ServiceCheck struct {
	Name string
	Kind string
	Ping func() error
}

var (
	httpClient = NewHttpClient(false)
)

func CreateApp(name string, version, desc string, factory func(app *Application)) *Application {
	app := &Application{Name: name, Version: version, Description: desc, env: os.Getenv("ENV")}
	app.factory = factory
	app.healthChecks = []ServiceCheck{}
	app.Router = app.CreateRouter()
	return app
}

func (app Application) IsProd() bool {
	return app.env == "prod"
}

func (app Application) IsTestEnv() bool {
	return app.env == "test"
}

func (app *Application) AddKongHealthcheck(url string) {
	app.AddToHealthcheck(ServiceCheck{
		Name: "Kong",
		Kind: "Url",
		Ping: func() error {
			resp, err := httpClient.Get(url, nil)
			if err != nil || resp.IsError {
				return AnyError(err, fmt.Errorf("%s", resp.Body))
			}
			json := JsonValue{value: string(resp.Body)}
			if json.GetString("message", "") == "no Route matched with those values" {
				return nil
			}
			return fmt.Errorf("expectation failed")
		},
	})
}

func (app *Application) AddStartupListener(callback func()) {
	if app.startupListeners == nil {
		app.startupListeners = []func(){}
	}
	app.startupListeners = append(app.startupListeners, callback)
}


func (app *Application) GetMessageBroker() MessageBroker {
	if app.messageBroker == nil {
		log.Fatal("No message broker found.")
	}
	return *app.messageBroker
}


func (app *Application) GetDataSource() *DataSource {
	if app.dataSources == nil {
		log.Fatal("No datasource found.")
	}
	return app.dataSources[0]
}

func (app *Application) AddBrokerHealthcheck(name string, broker MessageBroker) {
	if app.messageBroker == nil {
		app.messageBroker = &broker
	}
	app.healthChecks = append(app.healthChecks, ServiceCheck{
		Name: name,
		Kind: "Broker",
		Ping: broker.Ping,
	})
}

func (app *Application) AddToHealthcheck(check ServiceCheck) {
	app.healthChecks = append(app.healthChecks, check)
}

type AppRouter struct {
	engine *gin.Engine
	app    *Application
	routes []Route
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
	UserId      string
	Username    string
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

func (r Response) Forbidden(message string) {
	r.gin.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"message": message,
	})
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

func securityFilter(gc *gin.Context) bool {
	h := gc.GetHeader("X-Anonymous-Consumer")
	if "true" == strings.ToLower(h) {
		message := "Access to this resource is forbidden, please check your apiKey."
		_ = Capture(fmt.Sprintf("http.guest.access.forbidden:%s", gc.Request.RequestURI), fmt.Errorf(message))
		gc.AbortWithStatusJSON(403, H{
			"message": message,
		})
		return false
	}
	return true
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
	app.factory(app)
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
			log.Info("Loading config from vault: %s", app.configSource)
			if err := ReadVaultSecret(strings.TrimPrefix(app.configSource, "vault:"), dest); err != nil {
				log.Fatalf("Error starting service, failed to read secrets from vault.\n%v", err)
			}
		} else {
			log.Warnf("configLocation not supported: %s", app.configSource)
		}
	}
}

func (app *Application) CreateRouter() *AppRouter {
	r := gin.Default()
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
	router := &AppRouter{
		engine: r,
		app:    app,
	}
	router.Add(&Route{
		Method:  "GET",
		Paths:   []string{"/status", "/healthz"},
		Secure:  false,
		Handler: app.handleHealthCheck,
	})
	/*
		for _, r := range app.routes {
			app.router.addRoute(r)
		}
	*/
	return router
}

func (app *Application) RegisterDatasource(ds *DataSource) {
	if app.dataSources == nil {
		app.dataSources = []*DataSource{}
	}
	log.FatalErr(ds.Init())
	app.dataSources = append(app.dataSources, ds)
}

func (app *Application) getHealthCheck() (bool, []HealthCheck) {
	var comps []HealthCheck
	allUp := true
	if app.dataSources != nil {
		for _, ds := range app.dataSources {
			err := ds.Ping()
			hc := HealthCheck{
				Name:   ds.Name,
				Status: "UP",
				Kind:   "Database",
			}
			if err != nil {
				message := err.Error()
				hc.Status = "DOWN"
				hc.Message = &message
				allUp = false
			}
			comps = append(comps, hc)
		}
	}

	if len(app.healthChecks) > 0 {
		for _, c := range app.healthChecks {
			err := c.Ping()
			hc := HealthCheck{
				Name:   c.Name,
				Status: "UP",
				Kind:   c.Kind,
			}
			if err != nil {
				message := err.Error()
				hc.Status = "DOWN"
				hc.Message = &message
				allUp = false
			}
			comps = append(comps, hc)
		}
	}

	return allUp, comps
}

func (app *Application) printHealthCheck() {
	fmt.Println("\n++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Printf("%s:%s\n", app.Name, app.Version)
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("\nHealthchecks: ")
	allUp, checks := app.getHealthCheck()
	for _, hc := range checks {
		if hc.Status == "UP" {
			fmt.Printf("> %s.%s: %s\n", hc.Kind, hc.Name, hc.Status)
		} else {
			fmt.Printf("> %s.%s:- %s %v\n", hc.Kind, hc.Name, hc.Status, hc.Message)
		}
	}
	if !allUp {
		_ = Capture(fmt.Sprintf("service.start:%s", app.Name), fmt.Errorf("some components are not healthy"))
	}
	fmt.Printf("\n++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++\n\n")
}

func (app *Application) handleHealthCheck(_ Request, res Response) {
	status := "UP"
	allUp, checks := app.getHealthCheck()
	if !allUp {
		status = "DOWN"
	}
	comps := map[string]HealthCheck{}
	for _, c := range checks {
		comps[fmt.Sprintf("%s:%s", strings.ToLower(c.Kind), strings.ToLower(c.Name))] = c
	}
	res.OK(H{
		"application": app.Name,
		"version":     app.Version,
		"description": app.Description,
		"status":      status,
		"components":  comps,
	})
}

func (route *Route) Secured() *Route {
	route.Secure = true
	return route
}

func (router *AppRouter) GET(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *AppRouter) POST(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *AppRouter) Add(r *Route) *AppRouter {
	var paths []string
	if !IsStrEmpty(r.Path) {
		paths = append(paths, r.Path)
	}
	if len(r.Paths) > 0 {
		paths = append(paths, r.Paths[:]...)
	}
	for _, path := range paths {
		router.engine.Handle(r.Method, path, func(gc *gin.Context) {
			if r.Secure && !securityFilter(gc) {
				return
			}
			context := RequestContext{Application: router.app}
			consumer := getKongConsumer(gc)
			if consumer != nil {
				context.TenantId = consumer.Id
				context.HasTenant = true
				context.UserId = consumer.Id
				context.Username = consumer.Username
				log.Info("authenticated request received: %s/%s", context.UserId, context.Username)
			} else {
				context.HasTenant = false
			}
			if r.TenantRequired && !context.HasTenant {
				gc.AbortWithStatusJSON(403, H{
					"error":   "missing.tenant",
					"message": "No tenant was provided",
				})
				return
			}
			r.Handler(Request{gin: gc, Context: context}, Response{gin: gc})
		})
	}
	return router
}

func (app *Application) NewTestServer() *httptest.Server {
	//app.createRouter()
	app.invokeStartupListeners()
	return httptest.NewServer(app.Router.engine)
}

func (app *Application) ApplyDatabaseMigrations() error {
	if !app.initialized {
		return fmt.Errorf("application is not initialized, call app.Init(env) first")
	}
	if app.dataSources == nil {
		return nil
	}
	for _, ds := range app.dataSources {
		if err := ds.ApplyMigrations(nil); err != nil {
			return err
		}
	}
	return nil
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

func (app *Application) invokeStartupListeners() {
	if app.startupListeners != nil {
		defer func() {
			for _, l := range app.startupListeners {
				l()
			}
			log.Info("all startup listeneres invoked.")
		}()
	}
}

func (app *Application) Start(port int) {
	//app.createRouter()
	app.printHealthCheck()
	app.invokeStartupListeners()
	_ = app.Router.engine.Run(fmt.Sprintf(":%d", port))
}

func (app Application) Subscribe(topic string, broadcast bool, handler func(message Message) error) {
	if app.messageBroker == nil {
		log.Fatal("no broker configured")
	}
	(*app.messageBroker).Subscribe(topic, broadcast, handler)
}

func (rc RequestContext) IsTestEnv() bool {
	return rc.Application.IsTestEnv()
}

