package context

import (
	"net/http"
	"rud-api/internal/consts"

	"github.com/gin-gonic/gin"
)

func HealthCheckServer() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": "rud-api",
			"version": consts.AppVersion,
		})
	})
}
