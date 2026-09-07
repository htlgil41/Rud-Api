package taskqueues

import (
	"fmt"
	"log"
	"rud-api/internal/consts"
	"rud-api/internal/helpers"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func AnalisisDepartamentoQueueTask(
	message amqp.Delivery,
	repo *repositories.ModuloReporteRepositorioPg,
	analisisRepo *repositories.AnalisisVentasRepositorie,
	evento types.ReporteSolicitadoEvent,
) {
	var bitacora strings.Builder
	bitacora.WriteString("Evento recibido para su procesamiento")
	start, errStart := time.Parse("2006-01-02", evento.Parametros[0])
	if errStart != nil {
		bitacora.WriteString("\nError al parsear el parametro a fecha")
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}

		message.Ack(false)
		return
	}
	end, errend := time.Parse("2006-01-02", evento.Parametros[1])
	if errend != nil {
		bitacora.WriteString("\nError al parsear el parametro a fecha")
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}

		message.Ack(false)
		return
	}

	a, errDatesGenerates := helpers.GeneratesDatesNoMayorToday(start, end)
	if errDatesGenerates != nil {
		message.Ack(false)
		return
	}
	fmt.Fprintf(&bitacora, "\nFechas generadas correctamente (%d)", len(a))
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	departamentos, errDepartamentos := analisisRepo.GetDepartamentosCodigos()
	if errDepartamentos != nil {
		bitacora.WriteString("\nError obteniendo departamentos: ")
		bitacora.WriteString(errDepartamentos.Error())
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteFallo, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}
		message.Ack(false)
		return
	}
	bitacora.WriteString("\nDepartamentos obtenidos correctamente")
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	for _, f := range a {
		analisisDepartamento, errDep := analisisRepo.GetVentasDepartamento(
			f,
			f,
			helpers.TransformSliceToInSqlString(departamentos),
		)
		if errDep != nil {
			fmt.Println(errDep)
			continue
		}

		fmt.Println(analisisDepartamento)
	}

	bitacora.WriteString("\nReporte construido correctamente")
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteCompletado, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	if errAck := message.Ack(false); errAck != nil {
		log.Printf("Error confirmando el mensaje: %v", errAck)
		if errNack := message.Nack(false, true); errNack != nil {
			log.Printf("Error devolviendo el mensaje a la cola: %v", errNack)
		}
		return
	}

	fmt.Println("Process alredy")
}
