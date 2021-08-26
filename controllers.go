package sf

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"net/http"
	"regexp"
	"strings"
)

type HTTPError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type HandlerFunc func(req Request, res Response)

type CrudHandler interface {
	Create(req Request, res Response)
	Update(req Request, res Response)
	Delete(req Request, res Response)
	List(req Request, res Response)
}

const (
	AuthenticationKey = "authentication"
)

// *********************************************************************************************************************
// Router
// *********************************************************************************************************************

func (router *Router) SetAuthenticator(validator Authenticator) *Router {
	router.authenticate = validator
	return router
}

func (router *Router) SetJwtSettings(secret string, audience string) *Router {
	router.jwtSecret = secret
	router.audience = audience
	return router
}

func (router *Router) Any(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "*",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *Router) GET(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *Router) POST(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *Router) PATCH(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "PATCH",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *Router) PUT(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *Router) JwtAuth() *Router {
	router.jwtAuthRequired = true
	return router
}

func (router *Router) DELETE(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

type RouteOpts struct {
	JwtAuth   bool
	BasicAuth bool
}

func (router *Router) CRUDWithOptions(base string, handler CrudHandler, opts *RouteOpts) {
	var routes []*Route
	routes = append(routes, router.GET(base, handler.List))
	routes = append(routes, router.POST(base, handler.Create))
	routes = append(routes, router.DELETE(base, handler.Delete))
	routes = append(routes, router.PATCH(fmt.Sprintf("%s/:id", base), handler.Update))
	if opts != nil {
		for _, r := range routes {
			r.jwtAuthRequired = opts.JwtAuth
			r.basicAuthRequired = opts.BasicAuth
		}
	}
}

func (router *Router) Use(handler HandlerFunc) *Router {
	router.engine.Use(func(gc *gin.Context) {
		handler(
			Request{gin: gc, Raw: gc.Request, Context: router.appContext},
			Response{gin: gc},
		)
	})
	return router
}

func (router *Router) CRUD(base string, handler CrudHandler) {
	router.CRUDWithOptions(base, handler, nil)
}

func (router *Router) Add(r *Route) *Router {
	var paths []string
	if !h.IsStrEmpty(r.Path) {
		paths = append(paths, r.Path)
	}
	if len(r.Paths) > 0 {
		paths = append(paths, r.Paths[:]...)
	}

	handler := func(gc *gin.Context) {
		r.checkSecurityConstraints(router, gc)
		if !gc.IsAborted() {
			r.Handler(
				Request{gin: gc, Raw: gc.Request, Context: router.appContext},
				Response{gin: gc},
			)
		}
	}

	for _, path := range paths {
		if r.Method == "*" {
			router.engine.Any(path, handler)
		} else {
			router.engine.Handle(r.Method, path, handler)
		}
	}
	return router
}

func (route *Route) checkSecurityConstraints(router *Router, gc *gin.Context) {
	if route.open {
		return
	}
	if route.basicAuthRequired {
		user, password, hasAuth := gc.Request.BasicAuth()
		if !hasAuth {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, H{"message": "AUTH_REQUIRED required"})
			return
		}
		principal, err := router.authenticate(user, password)
		if err != nil || principal == nil {
			gc.AbortWithStatusJSON(http.StatusForbidden, H{"message": "INVALID_CREDENTIALS"})
			if err != nil {
				log.Error(err)
			}
			return
		}
		gc.Set(AuthenticationKey, Authentication{
			Username:  user,
			Guest:     false,
			Principal: principal,
		})
		return
	}

	if route.jwtAuthRequired || router.jwtAuthRequired {

		if h.IsStrEmpty(router.jwtSecret) {
			gc.AbortWithStatusJSON(http.StatusInternalServerError, H{"message": "MISSING_JWT_SECRET"})
			return
		}

		auth := gc.GetHeader("Authorization")
		if auth == "" {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, H{"message": "AUTH_REQUIRED required"})
			return
		}
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, H{"message": "AUTH_REQUIRED required"})
			return
		}
		token := auth[len("bearer "):]
		decoded, err := DecodeJwt(router.jwtSecret, token)
		if err != nil {
			gc.AbortWithStatusJSON(http.StatusForbidden, H{"message": "INVALID_CREDENTIALS", "error": err.Error()})
			if err != nil {
				log.Error(err)
			}
			return
		}
		if decoded.Audience != router.audience {
			gc.AbortWithStatusJSON(http.StatusForbidden, H{"message": "INVALID_AUDIENCE"})
			return
		}
		gc.Set(AuthenticationKey, Authentication{
			Username:  decoded.Subject,
			Principal: decoded.Ext,
		})

	}
}

func (route *Route) BasicAuth() *Route {
	route.basicAuthRequired = true
	return route
}

func (route *Route) JwtAuth() *Route {
	route.jwtAuthRequired = true
	return route
}

// *********************************************************************************************************************
// Response
// *********************************************************************************************************************

func (r Response) TODO() {
	r.gin.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{
		"message": "Work in progress",
	})
}

func (r Response) Writer() http.ResponseWriter {
	return r.gin.Writer
}
func (r Response) OK(body interface{}) {
	r.gin.JSON(http.StatusOK, body)
}

func (r Response) JSON(status int, body interface{}) {
	r.gin.JSON(status, body)
}

func (r Response) NotFound(message string) {
	r.gin.JSON(404, H{"message": message})
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

// *********************************************************************************************************************
// Request
// *********************************************************************************************************************

func (r *Request) Auth() Authentication {
	value, exists := r.gin.Get(AuthenticationKey)
	if exists {
		return value.(Authentication)
	} else {
		return Authentication{
			Guest:     true,
			Principal: nil,
			Username:  "guest",
		}
	}
}

func (r *Request) SetHeaders(headers map[string]string) {
	if headers != nil {
		for key, value := range headers {
			r.Raw.Header.Set(key, value)
		}
	}
}

func (r *Request) DelHeaders(headers ...string) {
	if headers != nil {
		for _, key := range headers {
			r.Raw.Header.Del(key)
		}
	}
}

func (r Request) Header(name string) string {
	return r.gin.GetHeader(name)
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
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), errors.Errorf(message))
		r.gin.JSON(http.StatusBadRequest, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return false
	}
	return true
}

func (r Request) IsAborted() bool {
	return r.gin.IsAborted()
}

func (r Request) RequireBasicAuth() *Credentials {
	user, password, hasAuth := r.gin.Request.BasicAuth()
	if !hasAuth || h.IsStrEmpty(user) {
		_ = Capture("http.request.unauthorized", errors.Errorf(r.gin.Request.RequestURI))
		r.gin.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "Missing credentials",
		})
		return nil
	}
	return &Credentials{Username: user, Password: password}
}

func (r Request) Param(name string) string {
	return r.gin.Param(name)
}

func (r Request) RequireParam(name string) string {
	value := r.gin.Param(name)
	if h.IsStrEmpty(value) {

		message := fmt.Sprintf("Parameter '%s' is missing", name)
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), errors.Errorf(message))

		r.gin.AbortWithStatusJSON(http.StatusBadRequest, H{
			"message": message,
		})
	}
	return value
}
