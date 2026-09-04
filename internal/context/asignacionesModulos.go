package context

import (
	"net/http"
	"rud-api/internal/repositories"

	"github.com/gin-gonic/gin"
)

type AsignarModuloBody struct {
	ModuloID string `json:"modulo_id" binding:"required"`
}

type AsignarModuloReporteBody struct {
	ModuloReporteID string `json:"modulo_reporte_id" binding:"required"`
}

func AsignarModuloHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("usuario_id")
		if targetUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "usuario_id es requerido"})
			return
		}
		if targetUserID == c.GetString("user_id") {
			c.JSON(http.StatusForbidden, gin.H{"error": "No puedes asignar modulos a tu propio usuario"})
			return
		}

		var body AsignarModuloBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := repo.AsignarModulo(targetUserID, body.ModuloID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Modulo asignado correctamente"})
	}
}

func EliminarModuloHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("usuario_id")
		moduloID := c.Param("modulo_id")
		if targetUserID == "" || moduloID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "usuario_id y modulo_id son requeridos"})
			return
		}

		if err := repo.EliminarModulo(targetUserID, moduloID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Modulo eliminado correctamente"})
	}
}

func AsignarModuloReporteHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("usuario_id")
		if targetUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "usuario_id es requerido"})
			return
		}
		if targetUserID == c.GetString("user_id") {
			c.JSON(http.StatusForbidden, gin.H{"error": "No puedes asignar modulos de reporte a tu propio usuario"})
			return
		}

		var body AsignarModuloReporteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := repo.AsignarModuloReporte(targetUserID, body.ModuloReporteID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Modulo de reporte asignado correctamente"})
	}
}

func EliminarModuloReporteHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("usuario_id")
		moduloReporteID := c.Param("modulo_reporte_id")
		if targetUserID == "" || moduloReporteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "usuario_id y modulo_reporte_id son requeridos"})
			return
		}

		if err := repo.EliminarModuloReporte(targetUserID, moduloReporteID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Modulo de reporte eliminado correctamente"})
	}
}
