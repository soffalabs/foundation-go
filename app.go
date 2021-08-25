package sf

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

type AppRouter struct {
	engine       *gin.Engine
	app          *Application
	routes       []Route
	authenticate Authenticator
	jwtSecret    string
	audience    string
}

type Route struct {
	Method            string
	Path              string
	Paths             []string
	Handler           HandlerFunc
	basicAuthRequired bool
	jwtAuthRequired   bool
}

type Authenticator = func(string, string) (*Authentication, error)

type Props struct {
	Name        string
	Values      []string
	Required    bool
	Description string
}

type RequestContext struct {
	Application *Application
	TenantId    string
	HasTenant   bool
	Username    string
}

type Request struct {
	gin     *gin.Context
	Raw     *http.Request
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

type Application struct {
	Name        string
	Description string
	Version     string
	Configure   func(app *Application)
	OnReady     func()

	// @private
	conf         ConfManager
	routes       []Route
	router       *AppRouter
	healthChecks []ServiceCheck
	// globals          map[string]interface{}
	//startupListeners []func()
	messageBroker  *MessageBroker
	dataSources    []*DataSource
	dataSourcesMap map[string]DataSource
}

type ServiceCheck struct {
	Name string
	Kind string
	Ping func() error
}

// *********************************************************************************************************************
// *********************************************************************************************************************

func (app *Application) Init(env string) {

	app.healthChecks = []ServiceCheck{}
	app.conf = newConfManager(env)
	app.dataSources = []*DataSource{}

	{
		// router
		r := gin.Default()
		r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
		router := &AppRouter{
			engine: r,
			app:    app,
		}
		router.Add(&Route{
			Method:  "GET",
			Paths:   []string{"/status", "/healthz"},
			Handler: app.handleHealthCheck,
		})
		app.router = router
	}

	app.Configure(app)
}

func (app *Application) IsTestEnv() bool {
	return app.conf.IsTest()
}

func (app *Application) IsProd() bool {
	return app.conf.IsProd()
}

func (app *Application) UseBroker(url string, queueName string, exchange string, handler MessageHandler) MessageBroker {
	impl, err := newMessageBroker(url)
	log.FatalErr(err)
	impl.Listen(queueName, exchange, []string{queueName}, handler)
	app.messageBroker = &MessageBroker{
		broker:   impl,
		queue:    app.Name,
		exchange: exchange,
	}
	return *app.messageBroker
}

func (app *Application) AddHealthCheck(name string, kind string, ping func() error) {
	app.healthChecks = append(app.healthChecks, ServiceCheck{
		Kind: kind,
		Name: name,
		Ping: ping,
	})
}

func (app *Application) AddDataSource(name string, url string, migrations []*gormigrate.Migration) *DataSource {
	return app.AddMultitenanDataSource(name, url, migrations, nil)
}

func (app *Application) AddMultitenanDataSource(name string, url string, migrations []*gormigrate.Migration, loader func() ([]string, error)) *DataSource {
	ds := &DataSource{
		ServiceName:   strings.TrimPrefix(app.Name, "bantu-"),
		Name:          name,
		Url:           url,
		Migrations:    migrations,
		TenantsLoader: loader,
	}
	log.FatalErr(ds.bootstrap())
	app.dataSources = append(app.dataSources, ds)
	return ds
}

func (app Application) GetBroker() MessageBroker {
	if app.messageBroker == nil {
		panic("No message broker found")
	}
	return *app.messageBroker
}

func (app Application) Conf(name string, env string, required bool) string {
	value := app.conf.Get(name, env)
	if IsStrEmpty(value) && required {
		log.Fatalf("The required parameter %s (%s) was not provided, please check your config.", name, env)
	}
	return value
}

func (app *Application) GetDataSource() DataSource {
	if app.dataSources == nil || len(app.dataSources) == 0 {
		panic("No datasource defined")
	}
	if len(app.dataSources) > 1 {
		panic("More than 1 datasoure was registerd, use named datasource instead")
	}
	return *app.dataSources[0]
}

func (app Application) Router() *AppRouter {
	return app.router
}

func (app *Application) GetNamedDataSource(name string) DataSource {
	return app.dataSourcesMap[strings.ToLower(name)]
}

func (app *Application) getHealthCheck() (bool, []HealthCheck) {
	var comps []HealthCheck

	if app.dataSources != nil {
		for _, ds := range app.dataSources {
			comps = append(comps, HealthCheck{
				Name: ds.Name,
				Kind: "Database",
			}.get(ds.Ping()))
		}
	}

	if app.messageBroker != nil {
		comps = append(comps, HealthCheck{
			Kind: "Broker",
			Name: "default",
		}.get(app.messageBroker.broker.Ping()))
	}

	if len(app.healthChecks) > 0 {
		for _, c := range app.healthChecks {
			comps = append(comps, HealthCheck{
				Name: c.Name,
				Kind: c.Kind,
			}.get(c.Ping()))
		}
	}

	allUp := true

	for _, hc := range comps {
		if hc.Status == "DOWN" {
			allUp = false
			break
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

func (app *Application) ApplyDatabaseMigrations() error {
	if app.dataSources == nil {
		return nil
	}
	app.dataSourcesMap = map[string]DataSource{}
	for _, ds := range app.dataSources {
		if err := ds.Migrate(nil); err != nil {
			return err
		}
		app.dataSourcesMap[strings.ToLower(ds.Name)] = *ds
	}
	return nil
}

func (app *Application) invokeStartupListeners() {
	if app.OnReady != nil {
		defer func() {
			app.OnReady()
			log.Info("all startup listeneres invoked.")
		}()
	}
}

func (app *Application) Start(port int) {
	app.printHealthCheck()
	app.invokeStartupListeners()
	if port == 0 {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		Fatal(err)
		l, err := net.ListenTCP("tcp", addr)
		Fatal(err)
		defer func(l *net.TCPListener) {
			_ = l.Close()
		}(l)
		port = l.Addr().(*net.TCPAddr).Port
	}
	_ = app.router.engine.Run(fmt.Sprintf(":%d", port))

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
	var randomPort bool
	var dbMigrations bool
	var envName string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the service in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			app.Init(envName)
			if dbMigrations {
				if err := app.ApplyDatabaseMigrations(); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Info("database migrations were skipped")
			}
			if randomPort {
				app.Start(0)
			} else {
				app.Start(port)
			}
		},
	}
	cmd.Flags().StringVarP(&envName, "env", "e", os.Getenv("ENV"), "active environment profile")
	cmd.Flags().IntVarP(&port, "port", "p", Getenvi("PORT", 8080), "server port")
	cmd.Flags().BoolVarP(&randomPort, "random-port", "r", false, "start the server on a random available port")
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
			app.Init(envName)
			if err := app.ApplyDatabaseMigrations(); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().StringVarP(&envName, "env", "e", Getenv(os.Getenv("ENV"), "dev", true), "active environment profile")

	return cmd
}

func (app *Application) NewTestServer() *httptest.Server {
	//app.createRouter()
	app.invokeStartupListeners()
	return httptest.NewServer(app.router.engine)
}
