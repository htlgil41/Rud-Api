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
	"github.com/xuri/excelize/v2"
)

func AnalisisGrupoQueueTask(
	message amqp.Delivery,
	query string,
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
	grupo, errGrupo := analisisRepo.GetGrupoCodigos()
	if errGrupo != nil {
		bitacora.WriteString("\nError obteniendo grupos: ")
		bitacora.WriteString(errGrupo.Error())
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

	var d []types.VentasGrupo = []types.VentasGrupo{}
	for _, f := range a {
		analisisgrupo, errDep := analisisRepo.GetVentasGrupo(
			query,
			f,
			f,
			helpers.TransformSliceToInSqlString(departamentos),
			helpers.TransformSliceToInSqlString(grupo),
		)
		if errDep != nil {
			fmt.Fprintf(&bitacora, "\nSe produjo un error en la etapa de construccion [%s]", errDep.Error())
			continue
		}
		d = append(d, analisisgrupo...)
	}
	bitacora.WriteString("\nInformacion recolectada correctamente - Se procede a crear el exel")
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
		}
	}()

	sheet := "Sheet1"
	index, err := f.NewSheet(sheet)
	if err != nil {
		bitacora.WriteString("\nError al crear el Sheets en el excel")
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteFallo, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}

		fmt.Println("No se pudo contruir el reporte excel")
		return
	}

	f.SetCellValue(sheet, "A1", "Fecha")
	f.SetCellValue(sheet, "B1", "Sucursal")
	f.SetCellValue(sheet, "C1", "Grupo")
	f.SetCellValue(sheet, "D1", "Departamento")
	f.SetCellValue(sheet, "E1", "Total")
	f.SetCellValue(sheet, "F1", "Precio")
	f.SetCellValue(sheet, "G1", "Subtotal")
	f.SetCellValue(sheet, "H1", "NCantidad")
	f.SetCellValue(sheet, "I1", "NCosto")
	f.SetCellValue(sheet, "J1", "UtilidadPer")
	f.SetCellValue(sheet, "K1", "Cantidad")
	f.SetCellValue(sheet, "L1", "Utilidad")
	f.SetCellValue(sheet, "M1", "CostoOferta")

	for idata, data := range d {
		row := idata + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), data.Fecha.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Sucursal)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), data.Grupo)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), data.Departamento)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), data.Total)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), data.Precio)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), data.Subtotal)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.NCantidad)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), data.NCosto)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), data.UtilidadPer)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), data.Cantidad)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), data.Utilidad)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), data.CostoOferta)
	}

	f.SetActiveSheet(index)
	if err := f.SaveAs(fmt.Sprintf("%sAnalisis_ventas_grupo %s.xlsx", time.Now().Format("2006-01-02 15:04:05"), evento.UsuarioID)); err != nil {
		fmt.Println(err)
	}

	bitacora.WriteString("\nReporte construido correctamente")
	if errAck := message.Ack(false); errAck != nil {
		log.Printf("Error confirmando el mensaje: %v", errAck)
		if errNack := message.Nack(false, true); errNack != nil {

			bitacora.WriteString("\nFallo confirmar la solicitud el reporte sigue en cola")
			if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteCompletado, bitacora.String()); errEstado != nil {
				log.Printf("Error actualizando estado del reporte: %v", errEstado)
			}
			log.Printf("Error devolviendo el mensaje a la cola: %v", errNack)
		}
		return
	}

	bitacora.WriteString("\nRud ha construido el reporte correctamente")
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteCompletado, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	fmt.Println("Process alredy")
}
