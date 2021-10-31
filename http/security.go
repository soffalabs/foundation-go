package http

import (
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"net/http"
	"strings"
)

type Credentials struct {
	Username string
	Password string
}

type Authentication struct {
	Username  string
	Audience  string
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

/*
func (f *JwtBearerFilter) Exclude(exclusions ...string) *JwtBearerFilter {
	f.Exclusion = exclusions
	return f
}

*/

func (f *Authentication) Claim(name string) interface{} {
	if f.Claims == nil {
		return nil
	}
	value, exists := f.Claims[name]
	if exists {
		return value
	}
	return nil
}

func (f *JwtBearerFilter) Handle(c *Context) {

	auth := c.Header("Authorization")

	if auth != "" && strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := auth[len("bearer "):]
		decoded, err := h.DecodeJwt(f.Secret, token)
		if err != nil {
			// c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_CREDENTIALS", "error": err.Error()})
			log.Default.Error(err)
			return
		}
		if !h.IsEmpty(f.Audience) && decoded.Audience != f.Audience {
			c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_AUDIENCE"})
			return
		}
		c.gin.Set(AuthenticationKey, Authentication{
			Username:  decoded.Subject,
			Principal: decoded,
			Audience:  decoded.Audience,
			Claims:    decoded.Ext,
		})
	}

	c.gin.Next()
}
