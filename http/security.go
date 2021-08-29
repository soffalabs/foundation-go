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
	Guest     bool
	Principal interface{}
}

type Filter interface {
	Handle(c *Context)
}

type JwtBearerFilter struct {
	Secret   string
	Audience string
}

func (f JwtBearerFilter) Handle(c *Context) {
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
			log.Error(err)
		}
		return
	}
	if !h.IsEmpty(f.Audience) && decoded.Audience != f.Audience {
		c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_AUDIENCE"})
		return
	}

	c.gin.Set(AuthenticationKey, Authentication{
		Username:  decoded.Subject,
		Principal: decoded.Ext,
	})

	c.gin.Next()
}

/*

func (r *Router) checkSecurityConstraints(route *Route, gc *gin.Context) {
	if route.Open {
		return
	}
	if route.basicAuthRequired {
		user, password, hasAuth := c.gin.Request.BasicAuth()
		if !hasAuth {
			c.gin.AbortWithStatusJSON(http.StatusUnauthorized, h.Map{"message": "AUTH_REQUIRED required"})
			return
		}
		principal, err := r.authenticate(user, password)
		if err != nil || principal == nil {
			c.gin.AbortWithStatusJSON(http.StatusForbidden, h.Map{"message": "INVALID_CREDENTIALS"})
			if err != nil {
				log.Error(err)
			}
			return
		}
		c.gin.Set(AuthenticationKey, Authentication{
			Username:  user,
			Guest:     false,
			Principal: principal,
		})
		return
	}

	if route.jwtAuthRequired || r.jwtAuthRequired {

		if h.IsStrEmpty(r.jwtSecret) {
			c.gin.AbortWithStatusJSON(http.StatusInternalServerError, h.Map{"message": "MISSING_JWT_SECRET"})
			return
		}



	}
}

*/
