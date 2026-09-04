package context

import (
	"net/http"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"rud-api/internal/types"

	"github.com/gin-gonic/gin"
)

type CrearModuloReporteBody struct {
	Nombre       string `json:"nombre" binding:"required"`
	Descripcion  string `json:"descripcion"`
	ParamsSize   int    `json:"params_size" binding:"required"`
	QueryPlane   string `json:"query_plane" binding:"required"`
	QueryPrepare string `json:"query_prepare" binding:"required"`
}

func CrearModuloReporteHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body CrearModuloReporteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newModuloReporte := types.ModuloReporte{
			ID:           libs.GenerateUlid(),
			Nombre:       body.Nombre,
			Descripcion:  body.Descripcion,
			ParamsSize:   body.ParamsSize,
			QueryPlane:   body.QueryPlane,
			QueryPrepare: body.QueryPrepare,
		}

		if err := repo.CreateModuloReporte(newModuloReporte); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando modulo reporte"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":          newModuloReporte.ID,
			"nombre":      newModuloReporte.Nombre,
			"descripcion": newModuloReporte.Descripcion,
			"params_size": newModuloReporte.ParamsSize,
		})
	}
}

func ListarModuloReportesHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportes, err := repo.GetAllModuloReportes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo reportes"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"reportes": reportes,
		})
	}
}
