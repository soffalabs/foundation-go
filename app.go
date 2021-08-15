package soffa

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func Start(router *gin.Engine) {
	port := GetEnv("PORT", "8080")
	_ = router.Run(fmt.Sprintf(":%s", port))
}
