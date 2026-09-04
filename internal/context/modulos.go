package context

import (
	"net/http"
	"rud-api/internal/repositories"

	"github.com/gin-gonic/gin"
)

func GetMisModulosHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		modulos, err := repo.GetUsuarioModulos(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo modulos"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"modulos": modulos,
		})
	}
}
