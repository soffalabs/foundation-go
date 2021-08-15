package soffa

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct {
}

func GinHandle(c *gin.Context, operation func() (interface{}, error)) {
	res, err := operation()
	if err != nil {
		switch t := err.(type) {
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		case TechnicalError:
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    t.Code,
				"message": t.Message,
			})
		case FunctionalError:
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    t.Code,
				"message": t.Message,
			})
		}
	} else {
		c.JSON(http.StatusOK, res)
	}
}
