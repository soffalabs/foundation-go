package soffa

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
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

	Datasource   *DatasourceManager
	Routes       []Route
	Env          string
	ConfigSource string
	Cli          bool
	Config       interface{}
	Context      interface{}

	router      *AppRouter
	initialized bool
}

func (app Application) IsDevMode() bool {
	return app.Env != "prod"
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

		log.Error(err)

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
		log.Debugf("Validation error -- %v", err.Error())
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

func securityFilter(gc *gin.Context) {
	h := gc.GetHeader("X-Anonymous-Consumer")
	if "true" == strings.ToLower(h) {
		gc.AbortWithStatusJSON(403, H{
			"message": "Anonymous access to this resource is forbidden",
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

func (app *Application) init() {
	if app.initialized {
		return
	}

	app.Env = strings.ToLower(app.Env)
	filenames := []string{fmt.Sprintf(".env.%s", app.Env), ".env"}

	for _, f := range filenames {
		if err := godotenv.Load(f); err != nil {
			log.Debug(err)
		}
	}

	if app.Config != nil {
		if err := env.Parse(app.Config); err != nil {
			log.Warn(err)
		}
	}

	if !IsStrEmpty(app.ConfigSource) {
		if strings.HasPrefix(app.ConfigSource, "vault:") {
			log.Infof("Loading config from vault: %s", app.ConfigSource)
			if err := ReadVaultSecret(strings.TrimPrefix(app.ConfigSource, "vault:"), &app.Config); err != nil {
				log.Fatalf("Error starting service, failed to read secrets from vault.\n%v", err)
			}
		} else {
			log.Warnf("configLocation not supported: %s", app.ConfigSource)
		}
	}

	app.Datasource.init()

	if !app.IsDevMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	app.initialized = true
	return
}

func (app *Application) createRouter() {
	app.init()
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
	return app.Datasource.Migrate()
}

func (app *Application) Execute() {

}
func (app *Application) Start(port int) {
	_ = app.router.engine.Run(fmt.Sprintf(":%d", port))
}

func init() {
	initLogging()
}

func (rc RequestContext) GetEntityManager(tenantId string) EntityManager {
	return *rc.Application.Datasource.GetTenant(tenantId)
}

/*
func (app *App) CreateMessagePublisher(url string) MessagePublisher {
	return CreateMessagePublisher(url, app.IsDevMode())
}

func (app *App) RegisterBroadcastListener(amqpurl string, channel string, handler MessageHandler) {
	CreateBroadcastMessageListener(amqpurl, channel, app.IsDevMode(), func(event Message) error {
		event.Context = app.Context
		return handler(event)
	})
}

func (app *App) CreateMessageListener(amqpurl string, channel string, handler MessageHandler) {
	CreateTopicMessageListener(amqpurl, channel, app.IsDevMode(), func(event Message) error {
		event.Context = app.Context
		return handler(event)
	})
}

*/
