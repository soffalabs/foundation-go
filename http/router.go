package http

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soffa-io/soffa-core-go/h"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"net/http"
)

const (
	AuthenticationKey = "authentication"
)

type Router struct {
	engine  *gin.Engine
	routes  []Route
	filters []Filter
}

type Error struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type HandlerFunc func(ctx *Context)

type CrudHandler interface {
	Create(ctx *Context)
	Update(ctx *Context)
	Delete(ctx *Context)
	List(ctx *Context)
}

type Route struct {
	Method            string
	Path              string
	Paths             []string
	Handler           HandlerFunc
	basicAuthRequired bool
	jwtAuthRequired   bool
	Open              bool
}

func NewRouter() *Router {
	r := gin.Default()
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
	r.Any("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	return &Router{engine: r}
}

func (r *Router) HttpHandler() http.Handler {
	return r.engine
}

func (r *Router) Any(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "*",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

func (r *Router) GET(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

func (r *Router) POST(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

func (r *Router) PATCH(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "PATCH",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

func (r *Router) PUT(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

func (r *Router) DELETE(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	}
	r.Add(route)
	return route
}

type RouteOpts struct {
	JwtAuth   bool
	BasicAuth bool
}

func (r *Router) CRUDWithOptions(base string, handler CrudHandler, opts *RouteOpts) {
	var routes []*Route
	routes = append(routes, r.GET(base, handler.List))
	routes = append(routes, r.POST(base, handler.Create))
	routes = append(routes, r.DELETE(base, handler.Delete))
	routes = append(routes, r.PATCH(fmt.Sprintf("%s/:id", base), handler.Update))
	routes = append(routes, r.POST(fmt.Sprintf("%s/:id", base), handler.Update))
	if opts != nil {
		for _, r := range routes {
			r.jwtAuthRequired = opts.JwtAuth
			r.basicAuthRequired = opts.BasicAuth
		}
	}
}

func (r *Router) Use(handlers ...Filter) *Router {
	var middlewares []gin.HandlerFunc
	for _, f := range handlers {
		middlewares = append(middlewares, func(gc *gin.Context) {
			f.Handle(newContext(gc))
		})
	}
	r.engine.Use(middlewares...)
	return r
}

func (r *Router) CRUD(base string, handler CrudHandler) {
	r.CRUDWithOptions(base, handler, nil)
}

func (r *Router) Add(route *Route) *Router {
	var paths []string
	if !h.IsStrEmpty(route.Path) {
		paths = append(paths, route.Path)
	}
	if len(route.Paths) > 0 {
		paths = append(paths, route.Paths[:]...)
	}

	handler := func(gc *gin.Context) {
		c := newContext(gc)
		defer func() {
			if err := recover(); err != nil {
				c.SendError(err.(error))
			}
		}()
		route.Handler(c)
	}

	for _, path := range paths {
		if route.Method == "*" {
			r.engine.Any(path, handler)
		} else {
			r.engine.Handle(route.Method, path, handler)
		}
	}
	return r
}

func (r *Router) Start(port int) {
	_ = r.engine.Run(fmt.Sprintf(":%d", port))
}
