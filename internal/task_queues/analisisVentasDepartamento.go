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

type AgrupacionDepartamento struct {
	Monto                   float64
	PorcentajeMonto         float64
	Costo                   float64
	Utilidad                float64
	PorcentajeUtilidadMonto float64
	PorcentajeUtilidad      float64
	CostoOferta             float64
}

type DashboardData struct {
	TotalMonto      float64                     `json:"totalMonto"`
	TotalCosto      float64                     `json:"totalCosto"`
	TotalUtilidad   float64                     `json:"totalUtilidad"`
	MargenPromedio  float64                     `json:"margenPromedio"`
	CantidadDeptos  int                         `json:"cantidadDeptos"`
	Departamentos   []DepartamentoDashboardData `json:"departamentos"`
	FechaGeneracion string                      `json:"fechaGeneracion"`
}

type DepartamentoDashboardData struct {
	Nombre             string  `json:"nombre"`
	Monto              float64 `json:"monto"`
	Costo              float64 `json:"costo"`
	Utilidad           float64 `json:"utilidad"`
	PorcentajeMonto    float64 `json:"porcentajeMonto"`
	PorcentajeUtilidad float64 `json:"porcentajeUtilidad"`
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

	dashboardHTMLBytes, err := generarHTMLDashboard(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el dashboard HTML: ")
		bitacora.WriteString(err.Error())
		estadoFinal = consts.EstadoReporteFallo
		message.Ack(false)
		return
	}

	if errHtml := os.WriteFile("reporte.html", dashboardHTMLBytes, 0644); errHtml != nil {
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

	log.Println("Proceso de reporte de departamento completado exitosamente")
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

func generarHTMLDashboard(ventasData []types.VentasDepartamento) ([]byte, error) {
	agrupacion := make(map[string]AgrupacionDepartamento)
	var totalMonto float64

	for _, v := range ventasData {
		ag := agrupacion[v.Departamento]
		ag.Monto += v.Subtotal
		ag.Costo += v.Costo
		ag.CostoOferta += v.Costo
		agrupacion[v.Departamento] = ag
		totalMonto += v.Subtotal
	}

	var totalCosto, totalUtilidad float64
	var departamentos []DepartamentoDashboardData

	for nombre, v := range agrupacion {
		utilidad := v.Monto - v.Costo
		totalCosto += v.Costo
		totalUtilidad += utilidad

		porcentajeMonto := safeDivide(v.Monto, totalMonto) * 100
		porcentajeUtilidad := safeDivide(utilidad, v.Monto) * 100

		departamentos = append(departamentos, DepartamentoDashboardData{
			Nombre:             nombre,
			Monto:              v.Monto,
			Costo:              v.Costo,
			Utilidad:           utilidad,
			PorcentajeMonto:    porcentajeMonto,
			PorcentajeUtilidad: porcentajeUtilidad,
		})
	}

	dashboardData := DashboardData{
		TotalMonto:      totalMonto,
		TotalCosto:      totalCosto,
		TotalUtilidad:   totalUtilidad,
		MargenPromedio:  safeDivide(totalUtilidad, totalMonto) * 100,
		CantidadDeptos:  len(agrupacion),
		Departamentos:   departamentos,
		FechaGeneracion: time.Now().Format("2006-01-02 15:04:05"),
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
    <title>Dashboard Análisis de Departamentos</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
            color: #333;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        .header {
            background: white;
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        .header h1 { color: #667eea; margin-bottom: 10px; }
        .header p { color: #666; }
        .kpi-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .kpi-card {
            background: white;
            border-radius: 12px;
            padding: 25px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            text-align: center;
        }
        .kpi-value {
            font-size: 2em;
            font-weight: bold;
            color: #667eea;
            margin: 10px 0;
        }
        .kpi-label { color: #888; font-size: 0.9em; text-transform: uppercase; letter-spacing: 1px; }
        .kpi-card.profit .kpi-value { color: #10b981; }
        .kpi-card.cost .kpi-value { color: #ef4444; }
        .charts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .chart-card {
            background: white;
            border-radius: 12px;
            padding: 25px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
        }
        .chart-card h2 {
            color: #333;
            margin-bottom: 20px;
            font-size: 1.2em;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }
        .chart-container { position: relative; height: 400px; }
        .footer { text-align: center; color: white; padding: 20px; opacity: 0.9; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Dashboard de Análisis por Departamento</h1>
            <p>Generado el: <strong id="fechaGeneracion"></strong></p>
        </div>
        <div class="kpi-grid">
            <div class="kpi-card"><div class="kpi-label">Monto Total</div><div class="kpi-value" id="kpiMonto">$0</div></div>
            <div class="kpi-card cost"><div class="kpi-label">Costo Total</div><div class="kpi-value" id="kpiCosto">$0</div></div>
            <div class="kpi-card profit"><div class="kpi-label">Utilidad Total</div><div class="kpi-value" id="kpiUtilidad">$0</div></div>
            <div class="kpi-card"><div class="kpi-label">Margen Promedio</div><div class="kpi-value" id="kpiMargen">0%%</div></div>
            <div class="kpi-card"><div class="kpi-label">Departamentos</div><div class="kpi-value" id="kpiDeptos">0</div></div>
        </div>
        <div class="charts-grid">
            <div class="chart-card"><h2>🥧 Distribución de Monto por Departamento</h2><div class="chart-container"><canvas id="chartDona"></canvas></div></div>
            <div class="chart-card"><h2>📊 Monto vs Costo vs Utilidad</h2><div class="chart-container"><canvas id="chartBarras"></canvas></div></div>
            <div class="chart-card"><h2>💰 Margen de Utilidad por Departamento</h2><div class="chart-container"><canvas id="chartMargen"></canvas></div></div>
            <div class="chart-card"><h2>📈 Ranking por Monto</h2><div class="chart-container"><canvas id="chartRanking"></canvas></div></div>
        </div>
        <div class="footer"><p>Reporte generado automáticamente por Rud API</p></div>
    </div>
    <script>
        const rawData = %s;
        document.getElementById('fechaGeneracion').textContent = rawData.fechaGeneracion;
        document.getElementById('kpiMonto').textContent = '$' + rawData.totalMonto.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiCosto').textContent = '$' + rawData.totalCosto.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiUtilidad').textContent = '$' + rawData.totalUtilidad.toLocaleString('es-MX', {maximumFractionDigits: 2});
        document.getElementById('kpiMargen').textContent = rawData.margenPromedio.toFixed(2) + '%%';
        document.getElementById('kpiDeptos').textContent = rawData.cantidadDeptos;

        const deptos = rawData.departamentos.sort((a, b) => b.monto - a.monto);
        const nombres = deptos.map(d => d.nombre);
        const montos = deptos.map(d => d.monto);
        const costos = deptos.map(d => d.costo);
        const utilidades = deptos.map(d => d.utilidad);
        const porcentajesMonto = deptos.map(d => d.porcentajeMonto);
        const porcentajesUtilidad = deptos.map(d => d.porcentajeUtilidad);
        const colores = ['#667eea', '#764ba2', '#f093fb', '#4facfe', '#00f2fe', '#43e97b', '#fa709a', '#fee140', '#30cfd0', '#a8edea'];

        new Chart(document.getElementById('chartDona'), { type: 'doughnut', data: { labels: nombres, datasets: [{ data: porcentajesMonto, backgroundColor: colores.slice(0, nombres.length), borderWidth: 2, borderColor: '#fff' }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'right' }, tooltip: { callbacks: { label: (ctx) => ctx.label + ': ' + ctx.parsed.toFixed(2) + '%%' } } } } });
        new Chart(document.getElementById('chartBarras'), { type: 'bar', data: { labels: nombres, datasets: [ { label: 'Monto', data: montos, backgroundColor: '#667eea' }, { label: 'Costo', data: costos, backgroundColor: '#ef4444' }, { label: 'Utilidad', data: utilidades, backgroundColor: '#10b981' } ] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'top' }, tooltip: { callbacks: { label: (ctx) => ctx.dataset.label + ': $' + ctx.parsed.y.toLocaleString('es-MX', {maximumFractionDigits: 2}) } } }, scales: { y: { ticks: { callback: (v) => '$' + v.toLocaleString('es-MX') } } } } });
        new Chart(document.getElementById('chartMargen'), { type: 'bar', data: { labels: nombres, datasets: [{ label: 'Margen de Utilidad (%%)', data: porcentajesUtilidad, backgroundColor: colores.slice(0, nombres.length) }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false }, tooltip: { callbacks: { label: (ctx) => ctx.parsed.y.toFixed(2) + '%%' } } }, scales: { y: { ticks: { callback: (v) => v + '%%' } } } } });
        new Chart(document.getElementById('chartRanking'), { type: 'bar', data: { labels: nombres, datasets: [{ label: 'Monto Total', data: montos, backgroundColor: '#764ba2' }] }, options: { indexAxis: 'y', responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false }, tooltip: { callbacks: { label: (ctx) => '$' + ctx.parsed.x.toLocaleString('es-MX', {maximumFractionDigits: 2}) } } }, scales: { x: { ticks: { callback: (v) => '$' + v.toLocaleString('es-MX') } } } } });
    </script>
</body>
</html>`, string(dataJSON))

	return []byte(html), nil
}
