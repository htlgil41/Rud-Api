package taskqueues

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xuri/excelize/v2"

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

type WorkerResultWithVitacora struct {
	Departamento []types.VentasDepartamento
	Bitacora     *strings.Builder
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
	analisisRepos []*repositories.AnalisisVentasRepositorie,
	evento types.ReporteSolicitadoEvent,
) {
	var bitacora strings.Builder

	tamCanal := len(analisisRepos)
	if tamCanal == 0 {
		tamCanal = 1
	}

	ventasCh := make(chan WorkerResultWithVitacora, tamCanal)
	var wg sync.WaitGroup

	for i, analisisRepo := range analisisRepos {
		wg.Add(1)
		go func(idx int, ar *repositories.AnalisisVentasRepositorie) {
			defer wg.Done()
			localBitacora := &strings.Builder{}
			fmt.Fprintf(localBitacora, "\n-----\nSUCURSAL BITACORA %s\n------\n", ar.Sucursal)
			valor := RecolentAnalisisSucursal(
				query,
				evento,
				ar,
				localBitacora,
			)

			ventasCh <- WorkerResultWithVitacora{
				Departamento: valor,
				Bitacora:     localBitacora,
			}
		}(i, analisisRepo)
	}

	go func() {
		wg.Wait()
		close(ventasCh)
	}()

	var ventasData []types.VentasDepartamento
	for lote := range ventasCh {
		bitacora.WriteString(lote.Bitacora.String())
		ventasData = append(ventasData, lote.Departamento...)
	}

	log.Printf("Generando proceso de archivos and dashboards")
	err := generarExcelDepartamentos(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el archivo Excel: ")
		bitacora.WriteString(err.Error())
		message.Ack(false)
		return
	}

	dashboardHTMLBytes, err := generarHTMLDashboard(ventasData)
	if err != nil {
		bitacora.WriteString("\nError al construir el dashboard HTML: ")
		bitacora.WriteString(err.Error())
		message.Ack(false)
		return
	}

	if errHtml := os.WriteFile("reporte.html", dashboardHTMLBytes, 0644); errHtml != nil {
		bitacora.WriteString("\nError al construir el archivo HTML: ")
		bitacora.WriteString(errHtml.Error())
	}

	bitacora.WriteString("\nRud ha construido el reporte correctamente")

	if err := message.Ack(false); err != nil {
		log.Printf("Error confirmando el mensaje (Ack): %v", err)

		if errNack := message.Nack(false, true); errNack != nil {
			log.Printf("Error crítico: no se pudo confirmar ni reencolar el mensaje (Nack): %v", errNack)
		}

		return
	}

	if err := repo.ActualizarEstadoReporte(evento.ReporteID, "Error", bitacora.String()); err != nil {
		log.Printf("Error crítico actualizando estado final del reporte %s: %v", evento.ReporteID, err)
	}
	log.Println("Proceso de reporte de departamento completado exitosamente")
}

func RecolentAnalisisSucursal(
	query string,
	evento types.ReporteSolicitadoEvent,
	analisisRepo *repositories.AnalisisVentasRepositorie,
	bitacora *strings.Builder,
) []types.VentasDepartamento {
	var ventasData []types.VentasDepartamento
	bitacora.WriteString("Evento recibido para su procesamiento")
	start, errStart := time.Parse("2006-01-02", evento.Parametros[0])
	end, errEnd := time.Parse("2006-01-02", evento.Parametros[1])
	if errStart != nil || errEnd != nil {
		bitacora.WriteString("\nError al parsear los parámetros a fecha")
		return ventasData
	}

	rangoFechas, errDates := helpers.GeneratesDatesNoMayorToday(start, end)
	if errDates != nil {
		bitacora.WriteString("\nError generando rango de fechas: ")
		bitacora.WriteString(errDates.Error())
		return ventasData
	}
	fmt.Fprintf(bitacora, "\nFechas generadas correctamente (%d días)", len(rangoFechas))

	departamentos, errDep := analisisRepo.GetDepartamentosCodigos()
	if errDep != nil {
		bitacora.WriteString("\nError obteniendo departamentos: ")
		bitacora.WriteString(errDep.Error())
		return ventasData
	}
	bitacora.WriteString("\nDepartamentos obtenidos correctamente")

	deptosInSQL := helpers.TransformSliceToInSqlString(departamentos)
	datosDia, err := analisisRepo.GetVentasDepartamento(query, start.Format("20060102"), end.Format("20060102"), deptosInSQL)
	if err != nil {
		fmt.Fprintf(bitacora, "\nAdvertencia: error obteniendo ventas para %s: %v", start.Format("20060102"), err)
	}

	ventasData = append(ventasData, datosDia...)

	if len(ventasData) == 0 {
		bitacora.WriteString("\nNo se encontraron datos para el rango de fechas seleccionado")
		return ventasData
	}
	bitacora.WriteString("\nInformación recolectada correctamente. Procediendo a crear el Excel.")
	return ventasData
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

		archivo.SetCellValue(sheet, fmt.Sprintf("A%d", fila), data.Fecha)
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
	// Cálculo del resumen por departamento (se mantiene igual)
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
		// NO se incluye Ventas aquí
	}

	dataJSON, err := json.Marshal(dashboardData)
	if err != nil {
		return nil, fmt.Errorf("error serializando datos del dashboard: %w", err)
	}

	ventasJSON, err := json.Marshal(ventasData)
	if err != nil {
		return nil, fmt.Errorf("error serializando datos de ventas: %w", err)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard Análisis de Departamentos</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/xlsx/0.18.5/xlsx.full.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
            color: #333;
        }
        .container { max-width: 1600px; margin: 0 auto; }
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
        .filters {
            background: white;
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            display: flex;
            flex-wrap: wrap;
            gap: 15px;
            align-items: flex-end;
        }
        .filter-group {
            display: flex;
            flex-direction: column;
            min-width: 150px;
        }
        .filter-group label {
            font-size: 0.85em;
            color: #666;
            margin-bottom: 5px;
            font-weight: 500;
        }
        .filter-group select, .filter-group input {
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 0.95em;
            background: white;
        }
        .btn-export {
            background: #10b981;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 6px;
            cursor: pointer;
            font-weight: 600;
            font-size: 0.95em;
            transition: background 0.2s;
        }
        .btn-export:hover { background: #059669; }
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
        .table-card {
            background: white;
            border-radius: 12px;
            padding: 25px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            margin-bottom: 20px;
            overflow: hidden;
        }
        .table-card h2 {
            color: #333;
            margin-bottom: 20px;
            font-size: 1.2em;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }
        .table-wrapper {
            max-height: 500px;
            overflow-y: auto;
            border: 1px solid #eee;
            border-radius: 8px;
        }
        table {
            width: 100%%;
            border-collapse: collapse;
            font-size: 0.9em;
        }
        th, td {
            padding: 10px 12px;
            text-align: left;
            border-bottom: 1px solid #eee;
            white-space: nowrap;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
            position: sticky;
            top: 0;
            z-index: 10;
            box-shadow: 0 2px 2px rgba(0,0,0,0.05);
        }
        tr:hover td { background: #f8f9fa; }
        .footer { text-align: center; color: white; padding: 20px; opacity: 0.9; }
        @media (max-width: 768px) {
            .charts-grid { grid-template-columns: 1fr; }
            .filters { flex-direction: column; align-items: stretch; }
            .filter-group { width: 100%%; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Dashboard de Análisis por Departamento</h1>
            <p>Generado el: <strong id="fechaGeneracion"></strong></p>
        </div>
        <div class="kpi-grid" id="kpiContainer"></div>
        <div class="filters">
            <div class="filter-group">
                <label for="sucursalSelect">Sucursal</label>
                <select id="sucursalSelect"><option value="todas">Todas</option></select>
            </div>
            <div class="filter-group">
                <label for="fechaDesde">Fecha desde</label>
                <input type="date" id="fechaDesde">
            </div>
            <div class="filter-group">
                <label for="fechaHasta">Fecha hasta</label>
                <input type="date" id="fechaHasta">
            </div>
            <div class="filter-group">
                <label for="departamentoSelect">Departamento</label>
                <select id="departamentoSelect"><option value="todos">Todos</option></select>
            </div>
            <button class="btn-export" onclick="exportToExcel()">📥 Exportar a Excel</button>
        </div>
        <div class="charts-grid">
            <div class="chart-card"><h2>🥧 Distribución de Monto por Departamento</h2><div class="chart-container"><canvas id="chartDona"></canvas></div></div>
            <div class="chart-card"><h2>📊 Monto vs Costo vs Utilidad</h2><div class="chart-container"><canvas id="chartBarras"></canvas></div></div>
            <div class="chart-card"><h2>🏢 Comparativa por Sucursal</h2><div class="chart-container"><canvas id="chartSucursal"></canvas></div></div>
            <div class="chart-card"><h2>💰 Margen de Utilidad por Departamento</h2><div class="chart-container"><canvas id="chartMargen"></canvas></div></div>
        </div>
        <div class="table-card">
            <h2>📋 Datos Detallados <span id="registroCount" style="font-size:0.8em;color:#888;"></span></h2>
            <div class="table-wrapper">
                <table id="dataTable">
                    <thead>
                        <tr>
                            <th>Fecha</th>
                            <th>Sucursal</th>
                            <th>Departamento</th>
                            <th>Diferencia</th>
                            <th>Subtotal</th>
                            <th>Cantidad</th>
                            <th>Utilidad</th>
                            <th>NCosto</th>
                            <th>Precio</th>
                            <th>Costo</th>
                            <th>Utilidad%%</th>
                            <th>Total</th>
                            <th>CostoOferta</th>
                        </tr>
                    </thead>
                    <tbody id="tableBody"></tbody>
                </table>
            </div>
        </div>
        <div class="footer"><p>Reporte generado automáticamente por Rud API</p></div>
    </div>
    <script>
        const rawData = %s;      // Datos agregados del dashboard (opcional)
        const rawVentas = %s;    // Datos crudos de ventas (array de VentasDepartamento)

        let filteredData = [];
        let charts = {};

        document.addEventListener('DOMContentLoaded', function() {
            document.getElementById('fechaGeneracion').textContent = rawData.fechaGeneracion;
            populateSelects();
            applyFilters();
            createCharts();
            updateCharts();
        });

        function populateSelects() {
            const sucursales = [...new Set(rawVentas.map(v => v.Sucursal))].sort();
            const deptos = [...new Set(rawVentas.map(v => v.Departamento))].sort();
            
            const sucSelect = document.getElementById('sucursalSelect');
            sucSelect.innerHTML = '<option value="todas">Todas</option>';
            sucursales.forEach(s => {
                const opt = document.createElement('option');
                opt.value = s;
                opt.textContent = s;
                sucSelect.appendChild(opt);
            });

            const depSelect = document.getElementById('departamentoSelect');
            depSelect.innerHTML = '<option value="todos">Todos</option>';
            deptos.forEach(d => {
                const opt = document.createElement('option');
                opt.value = d;
                opt.textContent = d;
                depSelect.appendChild(opt);
            });

            const fechas = rawVentas.map(v => new Date(v.Fecha));
            if (fechas.length > 0) {
                const minDate = new Date(Math.min(...fechas));
                const maxDate = new Date(Math.max(...fechas));
                document.getElementById('fechaDesde').min = minDate.toISOString().split('T')[0];
                document.getElementById('fechaDesde').max = maxDate.toISOString().split('T')[0];
                document.getElementById('fechaHasta').min = minDate.toISOString().split('T')[0];
                document.getElementById('fechaHasta').max = maxDate.toISOString().split('T')[0];
            }
        }

        function applyFilters() {
            const sucursal = document.getElementById('sucursalSelect').value;
            const depto = document.getElementById('departamentoSelect').value;
            const fechaDesde = document.getElementById('fechaDesde').value;
            const fechaHasta = document.getElementById('fechaHasta').value;

            filteredData = rawVentas.filter(v => {
                if (sucursal !== 'todas' && v.Sucursal !== sucursal) return false;
                if (depto !== 'todos' && v.Departamento !== depto) return false;
                const fecha = new Date(v.Fecha);
                if (fechaDesde) {
                    const desde = new Date(fechaDesde + 'T00:00:00');
                    if (fecha < desde) return false;
                }
                if (fechaHasta) {
                    const hasta = new Date(fechaHasta + 'T23:59:59');
                    if (fecha > hasta) return false;
                }
                return true;
            });

            renderTable();
            updateCharts();
        }

        function renderTable() {
            const tbody = document.getElementById('tableBody');
            tbody.innerHTML = '';
            document.getElementById('registroCount').textContent = '(' + filteredData.length + ' registros)';

            filteredData.forEach(v => {
                const tr = document.createElement('tr');
                const fecha = new Date(v.Fecha);
                const fechaStr = fecha.toLocaleDateString('es-MX') + ' ' + fecha.toLocaleTimeString('es-MX', {hour:'2-digit', minute:'2-digit'});
                tr.innerHTML = '<td>' + fechaStr + '</td><td>' + v.Sucursal + '</td><td>' + v.Departamento + '</td><td>' + v.Diferencia.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>$' + v.Subtotal.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>' + v.Cantidad.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>$' + v.Utilidad.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>' + v.NCosto.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>$' + v.Precio.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>$' + v.Costo.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>' + v.UtilidadPer.toFixed(2) + '%</td><td>$' + v.Total.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td><td>$' + v.CostoOferta.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</td>';
                tbody.appendChild(tr);
            });
        }

        function renderKPIs() {
            const totalMonto = filteredData.reduce((sum, v) => sum + v.Subtotal, 0);
            const totalCosto = filteredData.reduce((sum, v) => sum + v.Costo, 0);
            const totalUtilidad = filteredData.reduce((sum, v) => sum + v.Utilidad, 0);
            const margenPromedio = totalMonto > 0 ? (totalUtilidad / totalMonto) * 100 : 0;
            const deptosUnicos = new Set(filteredData.map(v => v.Departamento)).size;

            document.getElementById('kpiContainer').innerHTML = '<div class="kpi-card"><div class="kpi-label">Monto Total</div><div class="kpi-value">$' + totalMonto.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</div></div><div class="kpi-card cost"><div class="kpi-label">Costo Total</div><div class="kpi-value">$' + totalCosto.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</div></div><div class="kpi-card profit"><div class="kpi-label">Utilidad Total</div><div class="kpi-value">$' + totalUtilidad.toLocaleString('es-MX', {maximumFractionDigits:2}) + '</div></div><div class="kpi-card"><div class="kpi-label">Margen Promedio</div><div class="kpi-value">' + margenPromedio.toFixed(2) + '%</div></div><div class="kpi-card"><div class="kpi-label">Departamentos</div><div class="kpi-value">' + deptosUnicos + '</div></div>';
        }

        function createCharts() {
            const ctxDona = document.getElementById('chartDona').getContext('2d');
            const ctxBarras = document.getElementById('chartBarras').getContext('2d');
            const ctxSucursal = document.getElementById('chartSucursal').getContext('2d');
            const ctxMargen = document.getElementById('chartMargen').getContext('2d');

            charts.dona = new Chart(ctxDona, { type: 'doughnut', data: { labels: [], datasets: [{ data: [], backgroundColor: [] }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'right' } } } });
            charts.barras = new Chart(ctxBarras, { type: 'bar', data: { labels: [], datasets: [] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'top' } }, scales: { y: { ticks: { callback: (v) => '$' + v.toLocaleString('es-MX') } } } } });
            charts.sucursal = new Chart(ctxSucursal, { type: 'bar', data: { labels: [], datasets: [{ label: 'Monto Total', data: [], backgroundColor: '#667eea' }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { ticks: { callback: (v) => '$' + v.toLocaleString('es-MX') } } } } });
            charts.margen = new Chart(ctxMargen, { type: 'bar', data: { labels: [], datasets: [{ label: 'Margen %%', data: [], backgroundColor: [] }] }, options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { ticks: { callback: (v) => v + '%%' } } } } });
        }

        function updateCharts() {
            renderKPIs();
            const deptoMap = {};
            filteredData.forEach(v => {
                if (!deptoMap[v.Departamento]) deptoMap[v.Departamento] = { monto: 0, costo: 0, utilidad: 0 };
                deptoMap[v.Departamento].monto += v.Subtotal;
                deptoMap[v.Departamento].costo += v.Costo;
                deptoMap[v.Departamento].utilidad += v.Utilidad;
            });
            const deptos = Object.keys(deptoMap).sort((a,b) => deptoMap[b].monto - deptoMap[a].monto);
            const montosDepto = deptos.map(d => deptoMap[d].monto);
            const costosDepto = deptos.map(d => deptoMap[d].costo);
            const utilidadesDepto = deptos.map(d => deptoMap[d].utilidad);
            const margenDepto = deptos.map(d => deptoMap[d].monto > 0 ? (deptoMap[d].utilidad / deptoMap[d].monto) * 100 : 0);
            const colores = ['#667eea', '#764ba2', '#f093fb', '#4facfe', '#00f2fe', '#43e97b', '#fa709a', '#fee140', '#30cfd0', '#a8edea'];

            charts.dona.data.labels = deptos;
            charts.dona.data.datasets[0].data = montosDepto;
            charts.dona.data.datasets[0].backgroundColor = colores.slice(0, deptos.length);
            charts.dona.update();

            charts.barras.data.labels = deptos;
            charts.barras.data.datasets = [
                { label: 'Monto', data: montosDepto, backgroundColor: '#667eea' },
                { label: 'Costo', data: costosDepto, backgroundColor: '#ef4444' },
                { label: 'Utilidad', data: utilidadesDepto, backgroundColor: '#10b981' }
            ];
            charts.barras.update();

            const sucursalMap = {};
            filteredData.forEach(v => {
                if (!sucursalMap[v.Sucursal]) sucursalMap[v.Sucursal] = { monto: 0 };
                sucursalMap[v.Sucursal].monto += v.Subtotal;
            });
            const sucursales = Object.keys(sucursalMap).sort((a,b) => sucursalMap[b].monto - sucursalMap[a].monto);
            const montosSucursal = sucursales.map(s => sucursalMap[s].monto);
            charts.sucursal.data.labels = sucursales;
            charts.sucursal.data.datasets[0].data = montosSucursal;
            charts.sucursal.update();

            charts.margen.data.labels = deptos;
            charts.margen.data.datasets[0].data = margenDepto;
            charts.margen.data.datasets[0].backgroundColor = colores.slice(0, deptos.length);
            charts.margen.update();
        }

        function exportToExcel() {
            if (filteredData.length === 0) {
                alert('No hay datos para exportar');
                return;
            }
            const dataForExcel = filteredData.map(v => ({
                'Fecha': new Date(v.Fecha).toLocaleString('es-MX'),
                'Sucursal': v.Sucursal,
                'Departamento': v.Departamento,
                'Diferencia': v.Diferencia,
                'Subtotal': v.Subtotal,
                'Cantidad': v.Cantidad,
                'Utilidad': v.Utilidad,
                'NCosto': v.NCosto,
                'Precio': v.Precio,
                'Costo': v.Costo,
                'Utilidad%%': v.UtilidadPer,
                'Total': v.Total,
                'CostoOferta': v.CostoOferta
            }));
            const ws = XLSX.utils.json_to_sheet(dataForExcel);
            const wb = XLSX.utils.book_new();
            XLSX.utils.book_append_sheet(wb, ws, 'Ventas');
            XLSX.writeFile(wb, 'reporte_ventas.xlsx');
        }

        document.getElementById('sucursalSelect').addEventListener('change', applyFilters);
        document.getElementById('departamentoSelect').addEventListener('change', applyFilters);
        document.getElementById('fechaDesde').addEventListener('change', applyFilters);
        document.getElementById('fechaHasta').addEventListener('change', applyFilters);
    </script>
</body>
</html>`, string(dataJSON), string(ventasJSON))

	return []byte(html), nil
}
