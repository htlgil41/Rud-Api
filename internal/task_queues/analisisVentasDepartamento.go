package taskqueues

import (
	"fmt"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xuri/excelize/v2"

	"rud-api/internal/consts"
	"rud-api/internal/helpers"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
)

type AgrupacionDepartamento struct {
	Monto                   float64
	PorcentajeMonto         float64
	Costo                   float64
	Utilidad                float64
	PorcentajeUtilidadMonto float64
	PorcentajeUtilidad      float64
	CostoOferta             float64
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func AnalisisDepartamentoQueueTask(
	message amqp.Delivery,
	query string,
	repo *repositories.ModuloReporteRepositorioPg,
	analisisRepo *repositories.AnalisisVentasRepositorie,
	evento types.ReporteSolicitadoEvent,
) {
	var bitacora strings.Builder
	var estadoFinal string
	defer func(t *strings.Builder) {
		if estadoFinal == "" {
			estadoFinal = consts.EstadoReporteFallo
		}
		if err := repo.ActualizarEstadoReporte(evento.ReporteID, estadoFinal, t.String()); err != nil {
			log.Printf("Error crítico actualizando estado final del reporte %s: %v", evento.ReporteID, err)
		}
	}(&bitacora)

	bitacora.WriteString("Evento recibido para su procesamiento")
	start, errStart := time.Parse("2006-01-02", evento.Parametros[0])
	end, errEnd := time.Parse("2006-01-02", evento.Parametros[1])
	if errStart != nil || errEnd != nil {
		bitacora.WriteString("\nError al parsear los parámetros a fecha")
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	rangoFechas, errDates := helpers.GeneratesDatesNoMayorToday(start, end)
	if errDates != nil {
		bitacora.WriteString("\nError generando rango de fechas: ")
		bitacora.WriteString(errDates.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}
	fmt.Fprintf(&bitacora, "\nFechas generadas correctamente (%d días)", len(rangoFechas))

	departamentos, errDep := analisisRepo.GetDepartamentosCodigos()
	if errDep != nil {
		bitacora.WriteString("\nError obteniendo departamentos: ")
		bitacora.WriteString(errDep.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}
	bitacora.WriteString("\nDepartamentos obtenidos correctamente")

	deptosInSQL := helpers.TransformSliceToInSqlString(departamentos)
	var ventasData []types.VentasDepartamento

	for _, fecha := range rangoFechas {
		datosDia, err := analisisRepo.GetVentasDepartamento(query, fecha, fecha, deptosInSQL)
		if err != nil {
			fmt.Fprintf(&bitacora, "\nAdvertencia: error obteniendo ventas para %s: %v", fecha, err)
			continue
		}
		ventasData = append(ventasData, datosDia...)
	}

	if len(ventasData) == 0 {
		bitacora.WriteString("\nNo se encontraron datos para el rango de fechas seleccionado")
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}
	bitacora.WriteString("\nInformación recolectada correctamente. Procediendo a crear el Excel.")

	err := generarExcelDepartamentos(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el archivo Excel: ")
		bitacora.WriteString(err.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	bitacora.WriteString("\nRud ha construido el reporte correctamente")
	estadoFinal = consts.EstadoReporteCompletado

	if err := message.Ack(false); err != nil {
		log.Printf("Error confirmando el mensaje (Ack): %v", err)
		if errNack := message.Nack(false, true); errNack != nil {
			log.Printf("Error crítico: no se pudo confirmar ni reencolar el mensaje (Nack): %v", errNack)
		}
		return
	}

	log.Println("Proceso de reporte completado exitosamente")
}

func generarExcelDepartamentos(ventasData []types.VentasDepartamento) error {
	archivo := excelize.NewFile()
	defer archivo.Close()

	sheet := "Sheet1"
	encabezados := []string{
		"Fecha", "Sucursal", "Departamento", "Monto Subtotal", "Monto Total", "(%)",
		"NCosto", "Costo", "Utilidad", "Utilidad Per", "(%)", "Costo Oferta", "Diferencia", "Cantidad",
	}

	for col, header := range encabezados {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		archivo.SetCellValue(sheet, cell, header)
	}

	var totalMonto float64
	for _, v := range ventasData {
		totalMonto += v.Subtotal
	}

	for row, data := range ventasData {
		fila := row + 2
		utilidadCostoTotal := safeDivide(data.Total, totalMonto) * 100

		archivo.SetCellValue(sheet, fmt.Sprintf("A%d", fila), data.Fecha.Format("2006-01-02"))
		archivo.SetCellValue(sheet, fmt.Sprintf("B%d", fila), data.Sucursal)
		archivo.SetCellValue(sheet, fmt.Sprintf("C%d", fila), data.Departamento)
		archivo.SetCellValue(sheet, fmt.Sprintf("D%d", fila), data.Subtotal)
		archivo.SetCellValue(sheet, fmt.Sprintf("E%d", fila), data.Total)
		archivo.SetCellValue(sheet, fmt.Sprintf("F%d", fila), utilidadCostoTotal)
		archivo.SetCellValue(sheet, fmt.Sprintf("G%d", fila), data.NCosto)
		archivo.SetCellValue(sheet, fmt.Sprintf("H%d", fila), data.Costo)
		archivo.SetCellValue(sheet, fmt.Sprintf("I%d", fila), data.Subtotal-data.NCosto)
		archivo.SetCellValue(sheet, fmt.Sprintf("J%d", fila), data.UtilidadPer)
		archivo.SetCellValue(sheet, fmt.Sprintf("K%d", fila), safeDivide(utilidadCostoTotal, totalMonto)*100)
		archivo.SetCellValue(sheet, fmt.Sprintf("L%d", fila), data.CostoOferta)
		archivo.SetCellValue(sheet, fmt.Sprintf("M%d", fila), data.Diferencia)
		archivo.SetCellValue(sheet, fmt.Sprintf("N%d", fila), data.Cantidad)
	}

	agrupacion := make(map[string]AgrupacionDepartamento)
	for _, v := range ventasData {
		ag := agrupacion[v.Departamento]
		ag.Monto += v.Subtotal
		ag.Costo += v.Costo
		ag.CostoOferta += v.Costo
		agrupacion[v.Departamento] = ag
	}

	encabezadosAgrupados := []string{
		"Departamento", "Monto", "Porcentaje Monto", "Costo", "Utilidad",
		"Porcentaje Utilidad Monto", "Porcentaje Utilidad", "Costo Oferta",
	}
	offsetCol := 16
	for col, header := range encabezadosAgrupados {
		cell, _ := excelize.CoordinatesToCellName(offsetCol+col+1, 1)
		archivo.SetCellValue(sheet, cell, header)
	}

	var totalUtilidadPorcentaje float64
	for _, v := range agrupacion {
		totalUtilidadPorcentaje += safeDivide(v.Monto, totalMonto) * 100
	}

	filaAgrupada := 2
	for depto, v := range agrupacion {
		utilidad := v.Monto - v.Costo
		porcentajeUtilidad := safeDivide(utilidad, v.Monto) * 100
		porcentajeSobreTotal := safeDivide(porcentajeUtilidad, totalUtilidadPorcentaje) * 100

		archivo.SetCellValue(sheet, fmt.Sprintf("Q%d", filaAgrupada), depto)
		archivo.SetCellValue(sheet, fmt.Sprintf("R%d", filaAgrupada), v.Monto)
		archivo.SetCellValue(sheet, fmt.Sprintf("S%d", filaAgrupada), safeDivide(v.Monto, totalMonto)*100)
		archivo.SetCellValue(sheet, fmt.Sprintf("T%d", filaAgrupada), v.Costo)
		archivo.SetCellValue(sheet, fmt.Sprintf("U%d", filaAgrupada), utilidad)
		archivo.SetCellValue(sheet, fmt.Sprintf("V%d", filaAgrupada), porcentajeUtilidad)
		archivo.SetCellValue(sheet, fmt.Sprintf("W%d", filaAgrupada), porcentajeSobreTotal)
		archivo.SetCellValue(sheet, fmt.Sprintf("X%d", filaAgrupada), v.CostoOferta)
		filaAgrupada++
	}

	return archivo.SaveAs("reporte.xlsx")
}
