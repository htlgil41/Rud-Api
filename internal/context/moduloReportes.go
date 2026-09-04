package context

import (
	"fmt"
	"net/http"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"rud-api/internal/types"

	"github.com/gin-gonic/gin"
)

type SolicitudReporteBody struct {
	ReporteID  string   `json:"reporte_id" binding:"required"`
	Parametros []string `json:"parametros" binding:"required"`
}

func GetMisReportesHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		reportes, err := repo.GetUsuarioModuloReportes(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo reportes"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"reportes": reportes,
		})
	}
}

func SolicitarReporteHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body SolicitudReporteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetString("user_id")

		hasAccess, err := repo.HasModuloReporte(userID, body.ReporteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validando permisos"})
			return
		}
		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para este reporte"})
			return
		}

		moduloReporte, err := repo.GetModuloReporteByID(body.ReporteID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reporte no encontrado"})
			return
		}

		if len(body.Parametros) != moduloReporte.ParamsSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf(
					"Se esperaban %d parametros, se recibieron %d",
					moduloReporte.ParamsSize,
					len(body.Parametros),
				),
			})
			return
		}

		reporteGenerado := types.ReporteGenerado{
			ID:              libs.GenerateUlid(),
			ModuloReporteID: body.ReporteID,
			UsuarioID:       userID,
			Estado:          "PENDIENTE",
		}

		if err := repo.CreateReporteGenerado(reporteGenerado); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando reporte"})
			return
		}

		// TODO: Enviar a cola/evento para procesamiento
		// EventDispatcher.Send("reporte.pendiente", map[string]interface{}{
		//     "reporte_id": reporteGenerado.ID,
		//     "query":      moduloReporte.QueryPlane,
		//     "parametros": body.Parametros,
		// })

		c.JSON(http.StatusCreated, gin.H{
			"reporte_id": reporteGenerado.ID,
			"estado":     reporteGenerado.Estado,
			"mensaje":    "Reporte en cola de procesamiento",
		})
	}
}
