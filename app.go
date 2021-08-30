package soffa

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/soffa-io/soffa-core-go/broker"
	"github.com/soffa-io/soffa-core-go/conf"
	"github.com/soffa-io/soffa-core-go/db"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/http"
	"github.com/soffa-io/soffa-core-go/log"
)

type App struct {
	Name             string
	Version          string
	router           *http.Router
	cfg              *conf.Manager
	dbManager        *db.Manager
	broker           broker.Client
	onReadyListeners []func()
	scheduler *Scheduler
	args      map[string]interface{}
}

func NewApp(cfg *conf.Manager, name string, version string) *App {
	cfg.Load()
	a := &App{
		cfg:       cfg,
		Name:      name,
		scheduler: &Scheduler{s: gocron.NewScheduler(time.UTC), empty: true},
		Version:   version,
		args:      map[string]interface{}{},
	}
	return a
}

func (a *App) SetArg(key string, value interface{}) *App{
	a.args[key] = value
	return a
}

func (a *App) UseDB(cb func(m *db.Manager)) *App {
	if a.dbManager == nil {
		a.dbManager = db.NewManager()
	}
	cb(a.dbManager)
	return a
}

func (a *App) UseBroker(cb func(client broker.Client)) *App {
	if a.broker == nil {
		brokerUrl := a.cfg.Require("broker.url", "BROKER_URL", "MESSAGE_BROKER_URL")
		a.broker = broker.NewClient(brokerUrl, a.Name)
	}
	cb(a.broker)
	return a
}

func (a *App) Configure(cb func(router *http.Router, scheduler *Scheduler)) *App {
	if a.router == nil {
		a.router = http.NewRouter()
		a.router.Add(&http.Route{
			Method:  "GET",
			Paths:   []string{"/status", "/healthz"},
			Handler: a.handleHealthCheck,
			Open:    true,
		})
	}
	cb(a.router, a.scheduler)
	return a
}


func (a *App) AddStartupListener(fn func()) *App {
	if a.onReadyListeners == nil {
		a.onReadyListeners = []func(){}
	}
	a.onReadyListeners = append(a.onReadyListeners, fn)
	return a
}

func (a *App) MigrateDB() {
	if a.dbManager != nil {
		a.dbManager.Migrate()
	}
}

func (a *App) bootstrap() {
	a.printHealthCheck()
	if a.broker != nil  {
		a.broker.Start()
	}
	if a.dbManager != nil {
		a.dbManager.Migrate()
	}
	if a.scheduler != nil {
		a.scheduler.Start()
	}
	if a.onReadyListeners != nil {
		defer func() {
			for _, l := range a.onReadyListeners {
				l()
			}
			log.Info("All on-ready listeneres invoked.")
		}()
	}
}

func (a *App) Start(port int) {
	a.bootstrap()
	if a.cfg.IsProdEnv() {
		gin.SetMode(gin.ReleaseMode)
	}
	a.router.Start(port)
}

type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h HealthCheck) get(err error) HealthCheck {
	if err != nil {
		h.Status = "DOWN"
		h.Message = err.Error()
	} else {
		h.Status = "UP"
	}
	return h
}

func (a *App) getHealthCheck() (bool, []HealthCheck) {
	var comps []HealthCheck

	if a.dbManager != nil {
		comps = append(comps, HealthCheck{
			Name: "db",
		}.get(a.dbManager.Ping()))
	}

	if a.broker != nil {
		comps = append(comps, HealthCheck{
			Name: "broker",
		}.get(a.broker.Ping()))
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

func (a *App) handleHealthCheck(c *http.Context) {
	status := "UP"
	allUp, checks := a.getHealthCheck()
	if !allUp {
		status = "DOWN"
	}
	comps := map[string]HealthCheck{}
	for _, c := range checks {
		comps[c.Name] = c
	}
	c.OK(h.Map{
		"application": a.Name,
		"version":     a.Version,
		"status":      status,
		"components":  comps,
	})
}

func (a *App) printHealthCheck() {
	fmt.Println("\n++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Printf("%s:%s\n", a.Name, a.Version)
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	fmt.Println("\nHealthchecks: ")
	allUp, checks := a.getHealthCheck()
	for _, hc := range checks {
		if hc.Status == "UP" {

			fmt.Printf("> %s: %s\n", hc.Name, hc.Status)
		} else {
			fmt.Printf("> %s:- %s %v\n", hc.Name, hc.Status, hc.Message)
		}
	}
	if !allUp {
		_ = log.Capture(fmt.Sprintf("service.start:%s", a.Name), errors.Errorf("some components are not healthy"))
	}
	fmt.Printf("\n++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++\n\n")

}

func (a *App) Arg(key string) interface{}{
	return a.args[key]
}


type Scheduler struct {
	app   *App
	s     *gocron.Scheduler
	empty bool
}

func (s *Scheduler) Start() {
	if !s.empty {
		s.s.StartAsync()
		log.Info("Job secheduler is started.")
	}
}

func (s *Scheduler) Every(interval string, task func()) {
	_, err := s.s.Every(interval).Do(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("critital error from task execution -- %v", r)
			}
		}()
		task()
	})
	log.FatalIf(err)
	s.empty = false

}
