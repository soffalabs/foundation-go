package soffa

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/gin-gonic/gin"
	"github.com/go-gormigrate/gormigrate/v2"
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

type App struct {
	Name         string
	Env          string
	ConfigSource string
	Config       interface{}
	datasources  []EntityManager
	router       AppRouter
	Context      interface{}
}

type AppRouter struct {
	engine  *gin.Engine
	Context interface{}
}

type Request struct {
	gin     *gin.Context
	Context interface{}
}

type Validations struct {
	gin *gin.Context
}

type Response struct {
	gin     *gin.Context
	Context interface{}
}

type RouterConfig struct {
	Secure bool
}

type HandlerFunc func(req Request, res Response)

func NewApp(env string, configSource string, config interface{}) *App {
	app := &App{
		Env:          env,
		ConfigSource: configSource,
		Config:       config,
		Context:      H{},
	}
	filenames := []string{fmt.Sprintf(".env.%s", env), ".env"}
	for _, f := range filenames {
		if err := godotenv.Load(f); err != nil {
			log.Debug(err)
		}
	}
	app.Init()
	return app
}

func (app App) IsDevMode() bool {
	return app.Env != "prod"
}

func (app *App) Init() {
	app.Env = strings.ToLower(app.Env)
	if !app.IsDevMode() {
		gin.SetMode(gin.ReleaseMode)
	}
	log.Infof("ENV = %s", app.Env)
	initLogging()
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
}

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

func (app *App) CreateDatasource(name string, url string, migrations []*gormigrate.Migration) EntityManager {
	m := CreateEntityManager(name, url, migrations)
	app.datasources = append(app.datasources, m)
	return m
}

func (app *App) ApplyMigrations() {
	for _, db := range app.datasources {
		if err := db.ApplyMigrations(); err != nil {
			os.Exit(1)
		}
	}
}

func (app *App) NewRouter() AppRouter {
	r := gin.Default()
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))

	app.router = AppRouter{
		engine:  r,
		Context: app.Context,
	}

	app.router.GET("/status", nil, healthechkHandler)
	app.router.GET("/healthz", nil, healthechkHandler)

	return app.router
}

func (app App) Start(port int) {
	_ = app.router.engine.Run(fmt.Sprintf(":%d", port))
}

func (app App) NewTestServer() *httptest.Server {
	return httptest.NewServer(app.router.engine)
}

func (router AppRouter) POST(path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	return router.route("POST", path, config, handler)
}

func (router AppRouter) DELETE(path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	return router.route("DELETE", path, config, handler)
}

func (router AppRouter) PUT(path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	return router.route("PUT", path, config, handler)
}

func (router AppRouter) PATCH(path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	return router.route("PATCH", path, config, handler)
}

func (router AppRouter) GET(path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	return router.route("GET", path, config, handler)
}

func (router AppRouter) route(httpMethod string, path string, config *RouterConfig, handler HandlerFunc) AppRouter {
	group := router.engine.Group(path)
	if config != nil && config.Secure {
		group.Use(securityFilter)
	}
	group.Handle(httpMethod, "", func(gc *gin.Context) {
		handler(Request{gin: gc, Context: router.Context}, Response{gin: gc, Context: router.Context})
	})
	return router
}

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
	} else {
		r.OK(res)
	}
}

func (r Request) Header(name string) string {
	return r.gin.GetHeader(name)
}

func (r Request) GetKongConsumer() *KongConsumerInfo {
	id := r.gin.GetHeader("X-Consumer-ID")
	if IsStrEmpty(id) {
		return nil
	}
	return &KongConsumerInfo {
		Id:       id,
		CustomId: r.gin.GetHeader("X-Consumer-Custom-ID"),
		Username: r.gin.GetHeader("X-Consumer-Username"),
		CredentialIdentifier: r.gin.GetHeader("X-Credential-Identifier"),
		Anonymous: r.gin.GetHeader("X-Anonymous-Consumer") == "true",
	}
}

func (r Request) BindJson(dest interface{}) bool {
	if err := r.gin.ShouldBindJSON(dest); err != nil {
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

func securityFilter(gc *gin.Context) {
	h := gc.GetHeader("X-Anonymous-Consumer")
	if "true" == strings.ToLower(h) {
		gc.AbortWithStatus(403)
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
