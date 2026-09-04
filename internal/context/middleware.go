package context

import (
	"net/http"
	"strings"

	"rud-api/internal/libs"
	"rud-api/internal/repositories"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(joseToken *libs.JoseManagerToken) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Formato de token invalido"})
			return
		}

		payload, err := joseToken.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token invalido"})
			return
		}

		c.Set("user_id", payload.IdUser)
		c.Set("username", payload.Username)
		c.Next()
	}
}

func RequireModulo(repo *repositories.UsuarioRepositoriePg, modulos ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		for _, modulo := range modulos {
			has, err := repo.HasModulo(userID, modulo)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error validando permisos"})
				return
			}
			if !has {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "No tienes permiso: " + modulo})
				return
			}
		}
		c.Next()
	}
}
