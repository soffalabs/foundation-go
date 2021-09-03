package http

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/soffa-io/soffa-core-go/context"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/soffa-io/soffa-core-go/sentry"
	"net/http"
	"regexp"
)

type Context struct {
	gin     *gin.Context
	Context *context.Ctx
}

func newContext( gin *gin.Context) *Context {
	return &Context{gin: gin, Context: context.New()}
}

func (c *Context) Request() *http.Request {
	return c.gin.Request
}

func (c *Context) SetTenant(value string) *Context {
	c.Context.Set("tenant", value)
	return c
}

func (c *Context) TenantId() string {
	value, exists := c.gin.Get("tenant")
	if exists {
		return value.(string)
	}
	v := c.Header("X-Tenant-Id")
	if h.IsEmpty(v) {
		v = c.Header("X-TenantID")
	}
	return v
}

func (c *Context) Auth() Authentication {
	value, exists := c.gin.Get(AuthenticationKey)
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

func (c *Context) SetHeaders(headers map[string]string) {
	if headers != nil {
		for key, value := range headers {
			c.gin.Request.Header.Set(key, value)
		}
	}
}

func (c *Context) DelHeaders(headers ...string) {
	if headers != nil {
		for _, key := range headers {
			c.gin.Request.Header.Del(key)
		}
	}
}

func (c *Context) Header(name string) string {
	return c.gin.GetHeader(name)
}

func (c *Context) BindJson(dest interface{}) bool {
	if err := c.gin.ShouldBind(dest); err != nil {
		c.gin.JSON(http.StatusBadRequest, gin.H{
			"code":  "validation.error",
			"error": err.Error(),
		})
		return false
	}
	return true
}

func (c *Context) BindUri(dest interface{}) bool {
	if err := c.gin.ShouldBindUri(dest); err != nil {
		_ = log.Default.Capture(fmt.Sprintf("http.request.check:%s", c.gin.Request.RequestURI), err)
		c.gin.JSON(http.StatusBadRequest, gin.H{
			"code":  "validation.error",
			"error": err.Error(),
		})
		return false
	}
	return true
}

func (c *Context) CheckInputWithRegex(value string, pattern string, errorCode string) bool {
	found, err := regexp.MatchString(pattern, value)
	if !found || err != nil {
		message := ""
		if err != nil {
			message = err.Error()
		}
		_ = log.Default.Capture(fmt.Sprintf("http.request.check:%s", c.gin.Request.RequestURI), errors.Errorf(message))
		c.gin.JSON(http.StatusBadRequest, gin.H{
			"code":    errorCode,
			"message": message,
		})
		return false
	}
	return true
}

func (c *Context) IsAborted() bool {
	return c.gin.IsAborted()
}

func (c *Context) RequireBasicAuth() *Credentials {
	user, password, hasAuth := c.gin.Request.BasicAuth()
	if !hasAuth || h.IsStrEmpty(user) {
		_ = log.Default.Capture("http.request.unauthorized", errors.Errorf(c.gin.Request.RequestURI))
		c.gin.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "Missing credentials",
		})
		return nil
	}
	return &Credentials{Username: user, Password: password}
}

func (c *Context) Param(name string) string {
	return c.gin.Param(name)
}

func (c *Context) PostForm(name string) string {
	return c.gin.PostForm(name)
}

func (c *Context) RequireParam(name string) string {
	value := c.gin.Param(name)
	if h.IsStrEmpty(value) {

		message := fmt.Sprintf("Parameter '%s' is missing", name)
		_ = log.Default.Capture(fmt.Sprintf("http.request.check:%s", c.gin.Request.RequestURI), errors.Errorf(message))

		c.gin.AbortWithStatusJSON(http.StatusBadRequest, h.Map{
			"message": message,
		})
	}
	return value
}

// *********************************************************************************************************************
// Response
// *********************************************************************************************************************

func (c *Context) TODO() {
	c.gin.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{
		"message": "Work in progress",
	})
}

func (c *Context) Writer() http.ResponseWriter {
	return c.gin.Writer
}
func (c *Context) OK(body interface{}) {
	c.gin.JSON(http.StatusOK, body)
}

func (c *Context) Created(body interface{}) {
	c.JSON(http.StatusCreated, body)
}

func (c *Context) JSON(status int, body interface{}) {
	if !c.IsAborted() {
		c.gin.JSON(status, body)
	}
}
func (c *Context) String(status int, format string, args... interface{}) {
	if !c.IsAborted() {
		c.gin.String(status, format, args)
	}
}

func (c *Context) NotFound(message string) {
	c.JSON(404, h.Map{"message": message})
}

func (c *Context) BadRequest(body interface{}) {
	c.JSON(http.StatusBadRequest, body)
}

func (c *Context) Forbidden(message string) {
	c.gin.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"message": message,
	})
}

func (c *Context) SendError(orig error) {
	if c.IsAborted() {
		return
	}
	if orig == nil {
		return
	}

	err := errors.Unwrap(orig)

	switch err.(type) {
	default:
		sentry.CaptureException(orig)
		c.gin.JSON(http.StatusInternalServerError, gin.H{
			"message": orig.Error(),
		})
	case errors.ErrTechnical:
		sentry.CaptureException(orig)
		c.gin.JSON(http.StatusBadRequest, gin.H{
			"code":    (err.(errors.ErrTechnical)).Code,
			"message": orig.Error(),
		})
	case errors.ErrUnauthorized:
		c.gin.JSON(http.StatusUnauthorized, gin.H{
			"message": orig.Error(),
		})
	case errors.ErrFunctional:
		code := (err.(errors.ErrFunctional)).Code
		status := http.StatusBadRequest
		if code == errors.ErrNotFoundCode {
			status = http.StatusNotFound
		}
		msg := h.Map {
			"code":    code,
			"message": orig.Error(),
		}
		c.gin.JSON(status, msg)
	}

}

func (c *Context) Send(res interface{}, err error) {
	if c.IsAborted() {
		return
	}
	if err != nil {
		c.SendError(err)
	} else {
		c.OK(res)
	}
}
