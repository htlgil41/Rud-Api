package taskqueues

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xuri/excelize/v2"

	"rud-api/internal/consts"
	"rud-api/internal/helpers"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
)

func AnalisisSubGrupoQueueTask(
	message amqp.Delivery,
	query string,
	repo *repositories.ModuloReporteRepositorioPg,
	analisisRepo *repositories.AnalisisVentasRepositorie,
	evento types.ReporteSolicitadoEvent,
) {
	var bitacora strings.Builder
	var estadoFinal string
	defer func() {
		if estadoFinal == "" {
			estadoFinal = consts.EstadoReporteFallo
		}
		if err := repo.ActualizarEstadoReporte(evento.ReporteID, estadoFinal, bitacora.String()); err != nil {
			log.Printf("Error crítico actualizando estado final del reporte %s: %v", evento.ReporteID, err)
		}
	}()

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
	bitacora.WriteString(fmt.Sprintf("\nFechas generadas correctamente (%d días)", len(rangoFechas)))

	departamentos, errDep := analisisRepo.GetDepartamentosCodigos()
	if errDep != nil {
		bitacora.WriteString("\nError obteniendo departamentos: ")
		bitacora.WriteString(errDep.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	grupos, errGrupo := analisisRepo.GetGrupoCodigos()
	if errGrupo != nil {
		bitacora.WriteString("\nError obteniendo grupos: ")
		bitacora.WriteString(errGrupo.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	subGrupos, errSubGrupo := analisisRepo.GetSubGrupoCodigos()
	if errSubGrupo != nil {
		bitacora.WriteString("\nError obteniendo subgrupos: ")
		bitacora.WriteString(errSubGrupo.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}
	bitacora.WriteString("\nDepartamentos, grupos y subgrupos obtenidos correctamente")

	deptosInSQL := helpers.TransformSliceToInSqlString(departamentos)
	gruposInSQL := helpers.TransformSliceToInSqlString(grupos)
	subGruposInSQL := helpers.TransformSliceToInSqlString(subGrupos)

	var ventasData []types.SubGrupo

	for _, fecha := range rangoFechas {
		datosDia, err := analisisRepo.GetVentasSubGrupo(query, fecha, fecha, deptosInSQL, gruposInSQL, subGruposInSQL)
		if err != nil {
			fmt.Fprintf(&bitacora, "\nAdvertencia: error obteniendo ventas de subgrupo para %s: %v", fecha, err)
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

	err := generarExcelSubGrupo(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el archivo Excel: ")
		bitacora.WriteString(err.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	dashboardHTMLBytes, err := generarHTMLDashboardSubGrupo(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el dashboard HTML: ")
		bitacora.WriteString(err.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	if errHtml := os.WriteFile("subgrupo.html", dashboardHTMLBytes, 0644); errHtml != nil {
		bitacora.WriteString("\nError al construir el archivo HTML: ")
		bitacora.WriteString(errHtml.Error())
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

	log.Println("Proceso de reporte de subgrupo completado exitosamente")
}

func generarExcelSubGrupo(ventasData []types.SubGrupo) error {
	archivo := excelize.NewFile()
	defer archivo.Close()

	sheet := "Sheet1"
	encabezados := []string{
		"Sucursal", "Fecha", "Departamento", "Grupo", "SubGrupo", "Total",
		"NCantidad", "Cantidad", "Precio", "Subtotal", "NCosto", "UtilidadPer", "Utilidad", "CostoOferta",
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
		archivo.SetCellValue(sheet, fmt.Sprintf("A%d", fila), data.Sucursal)
		archivo.SetCellValue(sheet, fmt.Sprintf("B%d", fila), data.Fecha.Format("2006-01-02"))
		archivo.SetCellValue(sheet, fmt.Sprintf("C%d", fila), data.Departamento)
		archivo.SetCellValue(sheet, fmt.Sprintf("D%d", fila), data.Grupo)
		archivo.SetCellValue(sheet, fmt.Sprintf("E%d", fila), data.SubGrupo)
		archivo.SetCellValue(sheet, fmt.Sprintf("F%d", fila), data.Total)

		archivo.SetCellValue(sheet, fmt.Sprintf("G%d", fila), data.Ncatidad)
		archivo.SetCellValue(sheet, fmt.Sprintf("H%d", fila), data.Cantidad)
		archivo.SetCellValue(sheet, fmt.Sprintf("I%d", fila), data.Precio)
		archivo.SetCellValue(sheet, fmt.Sprintf("J%d", fila), data.Subtotal)
		archivo.SetCellValue(sheet, fmt.Sprintf("K%d", fila), data.Ncosto)
		archivo.SetCellValue(sheet, fmt.Sprintf("L%d", fila), data.UtilidadPer)
		archivo.SetCellValue(sheet, fmt.Sprintf("M%d", fila), data.Utilidad)
		archivo.SetCellValue(sheet, fmt.Sprintf("N%d", fila), data.CostoOferta)
	}

	agrupacion := make(map[string]AgrupacionSubGrupo)
	for _, v := range ventasData {
		ag := agrupacion[fmt.Sprintf("%s - %s - %s", v.Departamento, v.Grupo, v.SubGrupo)]
		ag.Monto += v.Subtotal
		ag.Costo += v.Ncosto
		ag.CostoOferta += v.Ncosto
		agrupacion[fmt.Sprintf("%s - %s - %s", v.Departamento, v.Grupo, v.SubGrupo)] = ag
	}

	encabezadosAgrupados := []string{
		"Departamento - Grupo - SubGrupo", "Monto", "Porcentaje Monto", "Costo", "Utilidad",
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

	return archivo.SaveAs("subgrupo.xlsx")
}

func generarHTMLDashboardSubGrupo(ventasData []types.SubGrupo) ([]byte, error) {
	type SubGrupoAgrupado struct {
		Monto    float64
		Costo    float64
		Utilidad float64
	}
	agrupacion := make(map[string]SubGrupoAgrupado)
	var totalMonto, totalCosto, totalUtilidad float64

	for _, v := range ventasData {
		ag := agrupacion[v.SubGrupo]
		ag.Monto += v.Subtotal
		ag.Costo += v.Ncosto
		ag.Utilidad += v.Utilidad
		agrupacion[v.SubGrupo] = ag

		totalMonto += v.Subtotal
		totalCosto += v.Ncosto
		totalUtilidad += v.Utilidad
	}

	type SubGrupoDashboardData struct {
		Nombre             string  `json:"nombre"`
		Monto              float64 `json:"monto"`
		Costo              float64 `json:"costo"`
		Utilidad           float64 `json:"utilidad"`
		PorcentajeMonto    float64 `json:"porcentajeMonto"`
		PorcentajeUtilidad float64 `json:"porcentajeUtilidad"`
	}

	var items []SubGrupoDashboardData
	for nombre, v := range agrupacion {
		porcentajeMonto := safeDivide(v.Monto, totalMonto) * 100
		porcentajeUtilidad := safeDivide(v.Utilidad, v.Monto) * 100

		items = append(items, SubGrupoDashboardData{
			Nombre:             nombre,
			Monto:              v.Monto,
			Costo:              v.Costo,
			Utilidad:           v.Utilidad,
			PorcentajeMonto:    porcentajeMonto,
			PorcentajeUtilidad: porcentajeUtilidad,
		})
	}

	dashboardData := map[string]interface{}{
		"titulo":         "Dashboard de Análisis por SubGrupo",
		"totalMonto":     totalMonto,
		"totalCosto":     totalCosto,
		"totalUtilidad":  totalUtilidad,
		"margenPromedio": safeDivide(totalUtilidad, totalMonto) * 100,
		"cantidad":       len(agrupacion),
		"items":          items,
		"fecha":          time.Now().Format("2006-01-02 15:04:05"),
	}

	dataJSON, err := json.Marshal(dashboardData)
	if err != nil {
		return nil, fmt.Errorf("error serializando datos del dashboard: %w", err)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard SubGrupos</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #121212; color: #e0e0e0; min-height: 100vh; padding: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background-color: #1e1e1e; border: 1px solid #333; border-radius: 12px; padding: 30px; margin-bottom: 20px; }
        .header h1 { color: #f97316; margin-bottom: 10px; }
        .header p { color: #a0a0a0; }
        .kpi-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .kpi-card { background-color: #1e1e1e; border: 1px solid #333; border-radius: 12px; padding: 25px; text-align: center; }
        .kpi-value { font-size: 2em; font-weight: bold; color: #f97316; margin: 10px 0; }
        .kpi-label { color: #888; font-size: 0.9em; text-transform: uppercase; letter-spacing: 1px; }
        .kpi-card.profit .kpi-value { color: #10b981; }
        .kpi-card.cost .kpi-value { color: #ef4444; }
        .charts-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(500px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .chart-card { background-color: #1e1e1e; border: 1px solid #333; border-radius: 12px; padding: 25px; }
        .chart-card h2 { color: #f97316; margin-bottom: 20px; font-size: 1.2em; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .chart-container { position: relative; height: 400px; }
        .footer { text-align: center; color: #666; padding: 20px; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 <span id="titulo"></span></h1>
            <p>Generado el: <strong id="fecha"></strong></p>
        </div>
        <div class="kpi-grid">
            <div class="kpi-card"><div class="kpi-label">Monto Total</div><div class="kpi-value" id="kpiMonto">$0</div></div>
            <div class="kpi-card cost"><div class="kpi-label">Costo Total</div><div class="kpi-value" id="kpiCosto">$0</div></div>
            <div class="kpi-card profit"><div class="kpi-label">Utilidad Total</div><div class="kpi-value" id="kpiUtilidad">$0</div></div>
            <div class="kpi-card"><div class="kpi-label">Margen Promedio</div><div class="kpi-value" id="kpiMargen">0%%</div></div>
            <div class="kpi-card"><div class="kpi-label">Total SubGrupos</div><div class="kpi-value" id="kpiCantidad">0</div></div>
        </div>
        <div class="charts-grid">
            <div class="chart-card"><h2>🥧 Distribución de Monto</h2><div class="chart-container"><canvas id="chartDona"></canvas></div></div>
            <div class="chart-card"><h2>📊 Monto vs Costo vs Utilidad</h2><div class="chart-container"><canvas id="chartBarras"></canvas></div></div>
            <div class="chart-card"><h2>💰 Margen de Utilidad (%%)</h2><div class="chart-container"><canvas id="chartMargen"></canvas></div></div>
            <div class="chart-card"><h2>📈 Ranking por Monto</h2><div class="chart-container"><canvas id="chartRanking"></canvas></div></div>
        </div>
        <div class="footer">Reporte generado automáticamente por Rud API</div>
    </div>
    <script>
        const rawData = %s;
        Chart.defaults.color = '#a0a0a0';
        Chart.defaults.borderColor = '#333';
        document.getElementById('titulo').textContent = rawData.titulo;
        document.getElementById('fecha').textContent = rawData.fecha;
        document.getElementById('kpiMonto').textContent = '$' + rawData.totalMonto.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiCosto').textContent = '$' + rawData.totalCosto.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiUtilidad').textContent = '$' + rawData.totalUtilidad.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiMargen').textContent = rawData.margenPromedio.toFixed(2) + '%%';
        document.getElementById('kpiCantidad').textContent = rawData.cantidad;

        const items = rawData.items.sort((a, b) => b.monto - a.monto);
        const nombres = items.map(d => d.nombre);
        const montos = items.map(d => d.monto);
        const costos = items.map(d => d.costo);
        const utilidades = items.map(d => d.utilidad);
        const pctMonto = items.map(d => d.porcentajeMonto);
        const pctUtilidad = items.map(d => d.porcentajeUtilidad);
        const colores = ['#f97316', '#fb923c', '#fdba74', '#38bdf8', '#34d399', '#a78bfa', '#f472b6'];

        new Chart(document.getElementById('chartDona'), { type: 'doughnut', data: { labels: nombres, datasets: [{ data: pctMonto, backgroundColor: colores.slice(0, nombres.length), borderWidth: 0 }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'right', labels: { color: '#e0e0e0' } } } } });
        new Chart(document.getElementById('chartBarras'), { type: 'bar', data: { labels: nombres, datasets: [ { label: 'Monto', data: montos, backgroundColor: '#f97316' }, { label: 'Costo', data: costos, backgroundColor: '#ef4444' }, { label: 'Utilidad', data: utilidades, backgroundColor: '#10b981' } ] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { labels: { color: '#e0e0e0' } } } } });
        new Chart(document.getElementById('chartMargen'), { type: 'bar', data: { labels: nombres, datasets: [{ label: 'Margen (%%)', data: pctUtilidad, backgroundColor: '#38bdf8' }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } } } });
        new Chart(document.getElementById('chartRanking'), { type: 'bar', data: { labels: nombres, datasets: [{ label: 'Monto', data: montos, backgroundColor: '#f97316' }] }, options: { indexAxis: 'y', responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } } } });
    </script>
</body>
</html>`, string(dataJSON))

	return []byte(html), nil
}
