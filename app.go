package soffa

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func Start(router *gin.Engine) {
	port := Getenv("PORT", "8080", true)
	_ = router.Run(fmt.Sprintf(":%s", port))
}

type Promise struct {
	Error  error
	Result interface{}
}

func (p Promise) Then(next func() Promise) Promise {
	if p.Error != nil {
		return p
	}
	return next()
}