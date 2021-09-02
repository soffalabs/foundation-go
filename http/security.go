package http

import (
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"net/http"
	"regexp"
	"strings"
)

type Credentials struct {
	Username string
	Password string
}

type Authentication struct {
	Username  string
	Guest     bool
	Principal interface{}
	Claims    map[string]interface{}
}

type Filter interface {
	Handle(c *Context)
}

type JwtBearerFilter struct {
	Secret    string
	Audience  string
	Exclusion []string
}

func (f *JwtBearerFilter) Exclude(exclusions ...string) *JwtBearerFilter {
	f.Exclusion = exclusions
	return f
}

func (f *JwtBearerFilter) Handle(c *Context) {

	if len(f.Exclusion)>0 {
		for _, e := range f.Exclusion {
			uri := strings.TrimSuffix(c.Request().RequestURI, "/")
			if uri == e {
				c.gin.Next()
				return
			}
			re := regexp.MustCompile(e)
			if re.MatchString(uri) {
				c.gin.Next()
				return
			}
		}
	}

	auth := c.Header("Authorization")
	if auth == "" {
		c.gin.AbortWithStatusJSON(http.StatusUnauthorized, h.Map{"message": "AUTH_REQUIRED required"})
		return
	}
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		c.gin.AbortWithStatusJSON(http.StatusUnauthorized, h.Map{"message": "AUTH_REQUIRED required"})
		return
	}
	token := auth[len("bearer "):]
	decoded, err := h.DecodeJwt(f.Secret, token)
	if err != nil {
		c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_CREDENTIALS", "error": err.Error()})
		if err != nil {
			log.Default.Error(err)
		}
		return
	}
	if !h.IsEmpty(f.Audience) && decoded.Audience != f.Audience {
		c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_AUDIENCE"})
		return
	}

	c.gin.Set(AuthenticationKey, Authentication{
		Username:  decoded.Subject,
		Principal: decoded,
		Claims:    decoded.Ext,
	})

	c.gin.Next()
}
