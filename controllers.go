package sf

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/soffa-io/soffa-core-go/log"
	"net/http"
	"regexp"
	"strings"
)

type HTTPError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

const (
	AuthenticationKey = "authentication"
)

// *********************************************************************************************************************
// Router
// *********************************************************************************************************************

func (router *AppRouter) SetAuthenticator(validator Authenticator) *AppRouter {
	router.authenticate = validator
	return router
}

func (router *AppRouter) SetJwtSettings(secret string, audience string) *AppRouter {
	router.jwtSecret = secret
	router.audience = audience
	return router
}

func (router *AppRouter) Any(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "*",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
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

func (router *AppRouter) PUT(path string, handler HandlerFunc) *Route {
	route := &Route{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	}
	router.Add(route)
	return route
}

func (router *AppRouter) Use(handler HandlerFunc) *AppRouter {
	router.engine.Use(func(gc *gin.Context) {
		context := RequestContext{Application: router.app}
		handler(Request{gin: gc, Raw: gc.Request, Context: context}, Response{gin: gc})
	})
	return router
}

func (router *AppRouter) Add(r *Route) *AppRouter {
	var paths []string
	if !IsStrEmpty(r.Path) {
		paths = append(paths, r.Path)
	}
	if len(r.Paths) > 0 {
		paths = append(paths, r.Paths[:]...)
	}

	handler := func(gc *gin.Context) {
		if r.basicAuthRequired {
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
			gc.Set(AuthenticationKey, principal)
		} else if r.jwtAuthRequired {
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
				gc.AbortWithStatusJSON(http.StatusForbidden, H{"message": "INVALID_CREDENTIALS"})
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
		context := RequestContext{Application: router.app}
		consumer := getKongConsumer(gc)
		if consumer != nil {
			context.HasTenant = true
			context.Username = consumer.Username
		} else {
			context.HasTenant = false
		}
		r.Handler(Request{gin: gc, Context: context}, Response{gin: gc})
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

func (r *Request) SetHeaders(headers map[string]string)  {
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
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), fmt.Errorf(message))
		r.gin.JSON(http.StatusBadRequest, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return false
	}
	return true
}

func (r Request) RequireBasicAuth() *Credentials {
	user, password, hasAuth := r.gin.Request.BasicAuth()
	if !hasAuth || IsStrEmpty(user) {
		_ = Capture("http.request.unauthorized", fmt.Errorf(r.gin.Request.RequestURI))
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
	if IsStrEmpty(value) {

		message := fmt.Sprintf("Parameter '%s' is missing", name)
		_ = Capture(fmt.Sprintf("http.request.check:%s", r.gin.Request.RequestURI), fmt.Errorf(message))

		r.gin.AbortWithStatusJSON(http.StatusBadRequest, H{
			"message": message,
		})
	}
	return value
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
