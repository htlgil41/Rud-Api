package context

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudReporteBody struct {
	ReporteID  string   `json:"reporte_id" binding:"required"`
	Parametros []string `json:"parametros" binding:"required"`
}

func parseCursor(cursorStr string) *string {
	if cursorStr == "" {
		return nil
	}
	return &cursorStr
}

func GetMisReportesHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		cursorStr := c.Query("cursor")
		cursor := parseCursor(cursorStr)

		result, err := repo.GetReportesGeneradosByUsuario(userID, cursor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo reportes"})
			return
		}

		response := gin.H{
			"user_id":  userID,
			"reportes": result.Reportes,
			"has_more": result.HasMore,
		}
		if result.NextCursor != "" {
			response["next_cursor"] = result.NextCursor
		}

		c.JSON(http.StatusOK, response)
	}
}

func SolicitarReporteHandler(repo *repositories.ModuloReporteRepositorioPg, publisher *repositories.RabbitPublisherRepositorio) gin.HandlerFunc {
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

		evento := types.ReporteSolicitadoEvent{
			ReporteID:       reporteGenerado.ID,
			UsuarioID:       userID,
			ModuloReporteID: body.ReporteID,
			Parametros:      body.Parametros,
			SolicitadoEn:    time.Now(),
		}
		eventBytes, errMarshal := json.Marshal(evento)
		if errMarshal != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error construyendo evento"})
			return
		}

		if errPublish := publisher.Publish(eventBytes); errPublish != nil {
			log.Printf("Error publicando evento reporte solicitado: %v", errPublish)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo procesar la solicitud del reporte"})
			return
		}

		if err := repo.CreateReporteGenerado(reporteGenerado); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando reporte"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"reporte_id": reporteGenerado.ID,
			"estado":     reporteGenerado.Estado,
			"mensaje":    "Reporte en cola de procesamiento",
		})
	}
}

type CancelarReporteBody struct {
	ReporteID string `json:"reporte_id" binding:"required"`
}

func CancelarReporteHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body CancelarReporteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetString("user_id")

		if err := repo.CancelReporteGenerado(body.ReporteID, userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reporte no encontrado o no pertenece al usuario"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"mensaje": "Reporte cancelado correctamente",
		})
	}
}

func GetAllReportesHandler(repo *repositories.ModuloReporteRepositorioPg) gin.HandlerFunc {
	return func(c *gin.Context) {
		cursorStr := c.Query("cursor")
		cursor := parseCursor(cursorStr)

		result, err := repo.GetAllReportesGenerados(cursor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo reportes"})
			return
		}

		response := gin.H{
			"reportes": result.Reportes,
			"has_more": result.HasMore,
		}
		if result.NextCursor != "" {
			response["next_cursor"] = result.NextCursor
		}

		c.JSON(http.StatusOK, response)
	}
}
