package sf

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
	"github.com/soffa-io/soffa-core-go/commons"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/soffa-io/soffa-core-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type AppRouter struct {
	engine     *gin.Engine
	App        *Application
	appContext *ApplicationContext
	routes          []Route
	authenticate    Authenticator
	jwtAuthRequired bool
	jwtSecret       string
	audience        string
}

type Route struct {
	Method            string
	Path              string
	Paths             []string
	Handler           HandlerFunc
	basicAuthRequired bool
	jwtAuthRequired   bool
	open              bool
}

type Authenticator = func(string, string) (*Authentication, error)

type AppTest struct {
	App    *Application
	server *httptest.Server
	test   *testing.T
	expect *httpexpect.Expect
}

type Props struct {
	Name        string
	Values      []string
	Required    bool
	Description string
}

type Request struct {
	gin     *gin.Context
	Raw     *http.Request
	Context *ApplicationContext
	//Auth    *Authentication
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

type Application struct {
	Name         string
	Description  string
	Version      string
	Configure    func(app *ApplicationContext)
	DataSources  func(app *ApplicationContext) []*DbLink
	OnReady      func()
	Broker       func(app *ApplicationContext) BrokerInfo
	RpcClient       func(app *ApplicationContext) rpc.Client
	CreateRouter func(router *AppRouter)

	// @private
	conf           ConfManager
	routes         []Route
	router         *AppRouter
	healthChecks   []ServiceCheck
	messageBroker  *MessageBroker
	dataSources    []*DbLink
	dataSourcesMap map[string]*DbLink
	args           map[string]interface{}
}

type ApplicationContext struct {
	app  *Application
	Name string
}

type ServiceCheck struct {
	Name string
	Kind string
	Ping func() error
}

// *********************************************************************************************************************
// *********************************************************************************************************************

/*
func Inject(function interface{}) error {
	return DI.Invoke(function)
}

func RegisterBean(constructor interface{}) {
	log.FatalErr(DI.Provide(constructor))
}*/

func (app *Application) Init(env string) {

	app.healthChecks = []ServiceCheck{}
	app.conf = newConfManager(env)
	app.dataSources = []*DbLink{}
	app.dataSourcesMap = map[string]*DbLink{}
	app.args = map[string]interface{}{}

	context := &ApplicationContext{
		app: app,
		Name: app.Name,
	}

	if app.DataSources != nil {
		ds := app.DataSources(context)
		if ds != nil {
			for _, item := range ds {
				err := item.bootstrap()
				log.FatalErr(err)
				app.dataSources = append(app.dataSources, item)
				app.dataSourcesMap[item.Name] = item
			}
		}
	}

	if app.Broker != nil {
		//brokerUrl := inbound.Conf("amqp.url", "AMQP_URL", true)
		info := app.Broker(context)
		impl, err := newMessageBroker(context, info.Url)
		log.FatalErr(err)
		impl.Listen( info.Queue, info.Exchange, []string{info.Queue}, info.Handler)
		app.messageBroker = &MessageBroker{
			broker:   impl,
			queue:    info.Queue,
			exchange: info.Exchange,
		}
	}

	{
		// router
		r := gin.Default()
		r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
		router := &AppRouter{
			engine:     r,
			App:        app,
			appContext: context,
			jwtSecret:  os.Getenv("JWT_SECRET"),
		}
		router.Add(&Route{
			Method:  "GET",
			Paths:   []string{"/status", "/healthz"},
			Handler: app.handleHealthCheck,
			open:    true,
		})
		app.router = router
	}

	if app.CreateRouter != nil {
		app.CreateRouter(app.router)
	}

	if app.Configure != nil {
		app.Configure(context)
	}
}

func (app *Application) IsTestEnv() bool {
	return app.conf.IsTest()
}

func (app *Application) IsProd() bool {
	return app.conf.IsProd()
}

type BrokerInfo struct {
	Url      string
	Queue    string
	Exchange string
	Handler  MessageHandler
}

func (app *Application) AddHealthCheck(name string, kind string, ping func() error) {
	app.healthChecks = append(app.healthChecks, ServiceCheck{
		Kind: kind,
		Name: name,
		Ping: ping,
	})
}

func (ac ApplicationContext) GetBroker() MessageBroker {
	return ac.app.GetBroker()
}
func (ac ApplicationContext) GetDbLink(name string) DbLink {
	return ac.app.GetDbLink(name)
}

func (ac ApplicationContext) GetDb() DbLink {
	return ac.app.GetDbLink("primary")
}

func (app Application) GetBroker() MessageBroker {
	if app.messageBroker == nil {
		panic("No message broker found")
	}
	return *app.messageBroker
}

func (ac ApplicationContext) Conf(name string, env string, required bool) string {
	return ac.app.Conf(name, env, required)
}

func (ac ApplicationContext) IsTestEnv() bool {
	return ac.app.IsTestEnv()
}

func (ac ApplicationContext) Set(name string, value interface{}) {
	ac.app.args[name] = value
}

func (ac ApplicationContext) Get(name string) interface{} {
	return ac.app.args[name]
}

func (app Application) Conf(name string, env string, required bool) string {
	value := app.conf.Get(name, env)
	if commons.IsStrEmpty(value) && required {
		log.Fatalf("The required parameter %s (%s) was not provided, please check your config.", name, env)
	}
	return value
}

func (app Application) Router() *AppRouter {
	return app.router
}

func (app *Application) GetDbLink(name string) DbLink {
	return *app.dataSourcesMap[strings.ToLower(name)]
}

func (app *Application) Get(arg string) (interface{}, bool) {
	if value, ok := app.args[arg]; ok {
		return  value, true
	}else {
		return nil, false
	}
}

func (app *Application) GetPrimaryEntityManager() *DbLink {
	return app.dataSourcesMap["primary"]
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
		"application":     app.Name,
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
	for _, ds := range app.dataSources {
		err := ds.Migrate(nil)
		log.FatalErr(err)
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

func (app *Application) CreateTestHelper(t *testing.T) AppTest {
	app.Init("test")
	assert.Nil(t, app.ApplyDatabaseMigrations())
	time.Sleep(200 * time.Millisecond)
	app.invokeStartupListeners()
	server := httptest.NewServer(app.router.engine)
	return AppTest{
		App:    app,
		test:   t,
		expect: httpexpect.New(t, server.URL),
		server: server,
	}

}

func (t *AppTest) Close() {
	t.server.Close()
}

func (t *AppTest) Truncate(dsName string, names []string) {
	ds := t.App.GetDbLink(dsName)
	for _, name := range names {
		_ = ds.Exec("DELETE FROM " + ds.TableName(name))
	}
}

func (t *AppTest) GET(path string) TestRequest {
	return TestRequest{
		request: t.expect.GET(path),
	}
}

func (t *AppTest) POST(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.POST(path).WithJSON(data),
	}
}

func (t *AppTest) PUT(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.PUT(path).WithJSON(data),
	}
}

func (t *AppTest) DELETE(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.DELETE(path).WithJSON(data),
	}
}

func (t *AppTest) PATCH(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.PATCH(path).WithJSON(data),
	}
}

func (t TestRequest) Expect() TestResponse {
	return TestResponse{
		response: t.request.Expect(),
	}
}

func (t TestRequest) Bearer(token string) TestRequest {
	t.request.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	return t
}

func (t TestRequest) BasicAuth(user string, password string) TestRequest {
	t.request.WithBasicAuth(user, password)
	return t
}

type TestRequest struct {
	request *httpexpect.Request
}

type TestResponse struct {
	response *httpexpect.Response
}

type TestResult struct {
	value *httpexpect.Value
}

func (t TestResponse) OK() TestResponse {
	t.response.Status(http.StatusOK)
	return t
}

func (t TestResponse) Unauthorized() TestResponse {
	t.response.Status(http.StatusUnauthorized)
	return t
}

func (t TestResponse) BadRequest() TestResponse {
	t.response.Status(http.StatusBadRequest)
	return t
}

func (t TestResponse) Forbidden() TestResponse {
	t.response.Status(http.StatusForbidden)
	return t
}

func (t TestResponse) Status(status int) TestResponse {
	t.response.Status(status)
	return t
}

func (t TestResponse) Json(path string) TestResult {
	return TestResult{
		value: t.response.JSON().Path(path),
	}
}

func (t TestResult) Is(value interface{}) TestResult {
	t.value.Equal(value)
	return t
}

func (t TestResult) Contains(value string) TestResult {
	t.value.String().Contains(value)
	return t
}

func (t TestResult) NotContains(value string) TestResult {
	t.value.String().NotContains(value)
	return t
}

func (t TestResult) IsArray() TestResult {
	t.value.Array()
	return t
}

func (t TestResult) IsEmptyArray() TestResult {
	t.value.Array().Empty()
	return t
}

func (t TestResult) IsNonEmptyArray() TestResult {
	t.value.Array().NotEmpty()
	return t
}

func (t TestResult) String() string {
	return t.value.String().Raw()
}

func (t TestResult) NotEmpty() TestResult {
	t.value.String().NotEmpty()
	return t
}

func (t TestResult) IsTrue() TestResult {
	t.value.Boolean().True()
	return t
}

func (t TestResult) Equal(value interface{}) TestResult {
	t.value.Equal(value)
	return t
}
