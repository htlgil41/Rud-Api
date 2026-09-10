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
	dataJSON, err := json.Marshal(ventasData)
	if err != nil {
		return nil, fmt.Errorf("error serializando datos del dashboard: %w", err)
	}

	html := strings.Replace(
		`
            <!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Dashboard Comparativo por Departamento</title>

  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    window.tailwind = window.tailwind || {};
    window.tailwind.config = { darkMode: 'class' };
  </script>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
  <script src="https://cdn.sheetjs.com/xlsx-0.20.2/package/dist/xlsx.full.min.js"></script>

  <style>
    .card {
      background: rgba(15, 23, 42, 0.72);
      border: 1px solid rgb(30 41 59);
      border-radius: 1rem;
      padding: 1rem;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
    }
    .thead-cell {
      padding: 0.5rem 0.75rem;
      text-align: left;
      white-space: nowrap;
    }
    .thead-cell-right {
      padding: 0.5rem 0.75rem;
      text-align: right;
      white-space: nowrap;
    }
    .tbody-cell {
      padding: 0.5rem 0.75rem;
      white-space: nowrap;
    }
    .tbody-cell-right {
      padding: 0.5rem 0.75rem;
      text-align: right;
      white-space: nowrap;
    }
  </style>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen">

  <div id="loader" class="fixed inset-0 z-50 bg-slate-950/95 flex flex-col items-center justify-center gap-4 px-6">
    <div class="w-14 h-14 border-4 border-slate-700 border-t-emerald-400 rounded-full animate-spin"></div>
    <div id="loaderMsg" class="text-sm text-slate-300">Procesando data...</div>
    <div class="w-80 max-w-[90vw] h-2 bg-slate-800 rounded-full overflow-hidden">
      <div id="progress" class="h-full bg-emerald-400 transition-all duration-200" style="width:0%"></div>
    </div>
  </div>

  <header class="sticky top-0 z-30 bg-slate-950/90 backdrop-blur border-b border-slate-800 px-4 py-3">
    <div class="flex flex-wrap items-center gap-3">
      <div class="min-w-[220px]">
        <h1 class="text-lg font-bold leading-tight">Dashboard Comparativo por Departamento</h1>
        <p id="subtitle" class="text-xs text-slate-400">Cargando...</p>
      </div>

      <div class="ml-auto flex flex-wrap items-end gap-2">
        <label class="text-xs text-slate-300 flex flex-col gap-1">
          Sucursal
          <select id="branchFilter" class="bg-slate-900 border border-slate-700 rounded-lg px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500">
            <option value="all">Todas</option>
          </select>
        </label>

        <label class="text-xs text-slate-300 flex flex-col gap-1">
          Métrica
          <select id="metric" class="bg-slate-900 border border-slate-700 rounded-lg px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500">
            <option value="monto">Ventas</option>
            <option value="utilidad">Utilidad</option>
            <option value="margen">Margen %</option>
            <option value="crecimiento">Crecimiento %</option>
          </select>
        </label>

        <button id="exportExcel" class="bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold rounded-lg px-3 py-2 text-sm transition-colors">
          Exportar Excel
        </button>
      </div>
    </div>
  </header>

  <main id="dashboard" class="p-4 space-y-4 opacity-0 transition-opacity duration-300">

    <section id="kpis" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5"></section>

    <section class="grid gap-4 xl:grid-cols-2">
      <div class="card xl:col-span-2">
        <div class="flex items-center justify-between gap-3 mb-3">
          <h2 class="font-semibold">Comparativo de Ventas por Sucursal</h2>
        </div>
        <div class="h-80">
          <canvas id="chartBranchComparison"></canvas>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-2">
      <div class="card">
        <h2 class="font-semibold mb-3">Crecimiento por Sucursal</h2>
        <div class="h-72">
          <canvas id="chartBranchGrowth"></canvas>
        </div>
      </div>

      <div class="card">
        <h2 class="font-semibold mb-3">Top Sucursales · <span id="topBranchMetric">Ventas</span></h2>
        <div class="h-72">
          <canvas id="chartTopBranches"></canvas>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-2">
      <div class="card">
        <h2 class="font-semibold mb-3">Top Departamento · <span id="topMetricLabel">Ventas</span></h2>
        <div class="h-72">
          <canvas id="chartDepartamentoTop"></canvas>
        </div>
      </div>

      <div class="card">
        <h2 class="font-semibold mb-3">Bottom Departamento · <span id="bottomMetricLabel">Ventas</span></h2>
        <div class="h-72">
          <canvas id="chartDepartamentoBottom"></canvas>
        </div>
      </div>
    </section>

    <section class="card">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3">
        <h2 class="font-semibold">Comparativo por Sucursal</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="text-slate-300 text-xs uppercase">
            <tr>
              <th class="thead-cell">Sucursal</th>
              <th class="thead-cell-right">Ventas</th>
              <th class="thead-cell-right">Costo</th>
              <th class="thead-cell-right">Utilidad</th>
              <th class="thead-cell-right">Margen %</th>
              <th class="thead-cell-right">Part. %</th>
              <th class="thead-cell-right">Crecimiento %</th>
              <th class="thead-cell-right">Departamentos</th>
            </tr>
          </thead>
          <tbody id="branchTableBody"></tbody>
        </table>
      </div>
    </section>

    <section class="card">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3">
        <h2 class="font-semibold">Resultados por Departamento</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead id="departamentoHead" class="text-slate-300 text-xs uppercase">
            <tr>
              <th data-sort="nombre" class="thead-cell cursor-pointer hover:text-white">Departamento</th>
              <th data-sort="monto" class="thead-cell-right cursor-pointer hover:text-white">Ventas</th>
              <th data-sort="costo" class="thead-cell-right cursor-pointer hover:text-white">Costo</th>
              <th data-sort="utilidad" class="thead-cell-right cursor-pointer hover:text-white">Utilidad</th>
              <th data-sort="margen" class="thead-cell-right cursor-pointer hover:text-white">Margen %</th>
              <th data-sort="participacion" class="thead-cell-right cursor-pointer hover:text-white">Part. %</th>
            </tr>
          </thead>
          <tbody id="departamentoBody"></tbody>
        </table>
      </div>
    </section>

  </main>

  <script id="raw-data" type="application/json">__RAW_DATA__</script>

  <script>
    (function () {
      function $(id) {
        return document.getElementById(id);
      }

      var els = {
        loader: $('loader'),
        loaderMsg: $('loaderMsg'),
        progress: $('progress'),
        dashboard: $('dashboard'),
        subtitle: $('subtitle'),
        branchFilter: $('branchFilter'),
        metric: $('metric'),
        exportExcel: $('exportExcel'),
        kpis: $('kpis'),
        branchTableBody: $('branchTableBody'),
        departamentoBody: $('departamentoBody'),
        topBranchMetric: $('topBranchMetric'),
        topMetricLabel: $('topMetricLabel'),
        bottomMetricLabel: $('bottomMetricLabel'),
        charts: {}
      };

      var palette = [
        '#34d399', '#60a5fa', '#fbbf24', '#f87171', '#a78bfa',
        '#f472b6', '#2dd4bf', '#fb923c', '#c084fc', '#38bdf8'
      ];

      var metricLabels = {
        monto: 'Ventas',
        utilidad: 'Utilidad',
        margen: 'Margen %',
        crecimiento: 'Crecimiento %'
      };

      var state = {
        rawData: [],
        branches: [],
        departamentos: [],
        totalMonto: 0,
        totalCosto: 0,
        totalUtilidad: 0,
        charts: {}
      };

      init();

      function init() {
        console.log('=== INICIANDO DASHBOARD ===');
        
        var rawNode = $('raw-data');
        var rawText = rawNode ? rawNode.textContent.trim() : '[]';

        if (!rawText || rawText === '__RAW_' + 'DATA__') {
          showError('No hay datos para procesar.');
          return;
        }

        try {
          var data = JSON.parse(rawText);
          
          if (Array.isArray(data)) {
            state.rawData = data;
          } else {
            state.rawData = [data];
          }

          console.log('Total registros:', state.rawData.length);
          if (state.rawData.length > 0) {
            console.log('Primer registro:', state.rawData[0]);
            console.log('Keys:', Object.keys(state.rawData[0]));
          }

          processData();
          bindEvents();
          renderAll();
          
          els.dashboard.classList.remove('opacity-0');
          hideLoader();
        } catch (err) {
          console.error('Error:', err);
          showError('Error procesando datos: ' + err.message);
        }
      }

      function processData() {
        var branchMap = {};
        var departamentoMap = {};
        var totalMonto = 0;
        var totalCosto = 0;
        var totalUtilidad = 0;

        state.rawData.forEach(function(item, idx) {
          var sucursal = item.Sucursal || item.sucursal || 'Sin sucursal';
          var departamento = item.Departamento || item.departamento || 'Sin departamento';
          var monto = num(item.Subtotal || item.subtotal || item.Total || item.total);
          var costo = num(item.NCosto || item.ncosto || item.Costo || item.costo);
          var utilidad = num(item.Utilidad || item.utilidad);
          if (utilidad === 0 && (monto || costo)) utilidad = monto - costo;

          if (!branchMap[sucursal]) {
            branchMap[sucursal] = {
              nombre: sucursal,
              monto: 0,
              costo: 0,
              utilidad: 0,
              departamentos: new Set(),
              items: []
            };
          }
          branchMap[sucursal].monto += monto;
          branchMap[sucursal].costo += costo;
          branchMap[sucursal].utilidad += utilidad;
          branchMap[sucursal].departamentos.add(departamento);
          branchMap[sucursal].items.push({monto: monto, idx: idx});

          if (!departamentoMap[departamento]) {
            departamentoMap[departamento] = {
              nombre: departamento,
              monto: 0,
              costo: 0,
              utilidad: 0,
              sucursales: new Set()
            };
          }
          departamentoMap[departamento].monto += monto;
          departamentoMap[departamento].costo += costo;
          departamentoMap[departamento].utilidad += utilidad;
          departamentoMap[departamento].sucursales.add(sucursal);

          totalMonto += monto;
          totalCosto += costo;
          totalUtilidad += utilidad;
        });

        state.branches = Object.keys(branchMap).map(function(nombre, idx) {
          var b = branchMap[nombre];
          var margen = safeDiv(b.utilidad, b.monto) * 100;
          var participacion = safeDiv(b.monto, totalMonto) * 100;
          var growth = calculateGrowth(b.items);

          return {
            nombre: nombre,
            monto: b.monto,
            costo: b.costo,
            utilidad: b.utilidad,
            margen: margen,
            participacion: participacion,
            crecimiento: growth,
            departamentos: b.departamentos.size,
            color: palette[idx % palette.length]
          };
        });

        state.departamentos = Object.keys(departamentoMap).map(function(nombre) {
          var d = departamentoMap[nombre];
          return {
            nombre: nombre,
            monto: d.monto,
            costo: d.costo,
            utilidad: d.utilidad,
            margen: safeDiv(d.utilidad, d.monto) * 100,
            participacion: safeDiv(d.monto, totalMonto) * 100,
            alcance: d.sucursales.size
          };
        });

        state.totalMonto = totalMonto;
        state.totalCosto = totalCosto;
        state.totalUtilidad = totalUtilidad;

        console.log('Sucursales detectadas:', state.branches.length);
        console.log('Departamentos detectados:', state.departamentos.length);
      }

      function calculateGrowth(items) {
        if (items.length < 2) return 0;
        
        var sorted = items.slice().sort(function(a, b) {
          return a.idx - b.idx;
        });

        var mid = Math.floor(sorted.length / 2);
        var firstHalf = sorted.slice(0, mid);
        var secondHalf = sorted.slice(mid);

        var firstSum = firstHalf.reduce(function(sum, item) { return sum + item.monto; }, 0);
        var secondSum = secondHalf.reduce(function(sum, item) { return sum + item.monto; }, 0);

        if (firstSum === 0) return secondSum > 0 ? 100 : 0;
        return ((secondSum - firstSum) / firstSum) * 100;
      }

      function bindEvents() {
        els.branchFilter.addEventListener('change', renderAll);
        els.metric.addEventListener('change', function() {
          renderBranchCharts();
          renderDepartamentoCharts();
        });
        els.exportExcel.addEventListener('click', exportExcel);

        Array.prototype.forEach.call(document.querySelectorAll('#departamentoHead th[data-sort]'), function (th) {
          th.addEventListener('click', function () {
            var key = th.getAttribute('data-sort');
            renderDepartamentoTable();
          });
        });
      }

      function renderAll() {
        renderKPIs();
        populateBranchFilter();
        renderBranchCharts();
        renderDepartamentoCharts();
        renderBranchTable();
        renderDepartamentoTable();
      }

      function renderKPIs() {
        var filteredBranches = getFilteredBranches();
        var totalMonto = filteredBranches.reduce(function(sum, b) { return sum + b.monto; }, 0);
        var totalUtilidad = filteredBranches.reduce(function(sum, b) { return sum + b.utilidad; }, 0);
        var avgMargin = safeDiv(totalUtilidad, totalMonto) * 100;

        var cards = [
          ['Ventas Totales', fmtNum(totalMonto, 0)],
          ['Utilidad Total', fmtNum(totalUtilidad, 0)],
          ['Margen Promedio', fmtPct(avgMargin)],
          ['Sucursales', fmtNum(filteredBranches.length, 0)],
          ['Departamentos', fmtNum(state.departamentos.length, 0)]
        ];

        els.kpis.innerHTML = cards.map(function (c) {
          return '<div class="card"><div class="text-xs text-slate-400">' + c[0] + '</div><div class="text-xl font-bold mt-1">' + c[1] + '</div></div>';
        }).join('');

        els.subtitle.textContent = filteredBranches.length + ' sucursales · ' + state.departamentos.length + ' departamentos';
      }

      function populateBranchFilter() {
        var current = els.branchFilter.value;
        els.branchFilter.innerHTML = '<option value="all">Todas</option>' + 
          state.branches.map(function(b) {
            return '<option value="' + escapeHtml(b.nombre) + '">' + escapeHtml(b.nombre) + '</option>';
          }).join('');
        
        if (current && current !== 'all') {
          els.branchFilter.value = current;
        }
      }

      function getFilteredBranches() {
        var filter = els.branchFilter.value;
        if (filter === 'all') return state.branches;
        return state.branches.filter(function(b) { return b.nombre === filter; });
      }

      function renderBranchCharts() {
        renderBranchComparisonChart();
        renderBranchGrowthChart();
        renderTopBranchesChart();
      }

      function renderBranchComparisonChart() {
        destroyChart('branchComparison');
        
        var branches = getFilteredBranches();

        var ctx = $('chartBranchComparison');
        state.charts.branchComparison = new Chart(ctx, {
          type: 'bar',
          data: {
            labels: branches.map(function(b) { return b.nombre; }),
            datasets: [
              {
                label: 'Ventas',
                data: branches.map(function(b) { return b.monto; }),
                backgroundColor: '#34d399',
                borderRadius: 6
              },
              {
                label: 'Utilidad',
                data: branches.map(function(b) { return b.utilidad; }),
                backgroundColor: '#60a5fa',
                borderRadius: 6
              }
            ]
          },
          options: {
            maintainAspectRatio: false,
            plugins: {
              legend: { position: 'bottom' },
              tooltip: {
                callbacks: {
                  label: function(ctx) {
                    return ctx.dataset.label + ': ' + fmtNum(ctx.parsed.y, 0);
                  }
                }
              }
            },
            scales: {
              y: {
                ticks: {
                  callback: function(value) { return fmtNum(value, 0); }
                }
              }
            }
          }
        });
      }

      function renderBranchGrowthChart() {
        destroyChart('branchGrowth');
        
        var branches = getFilteredBranches();
        var sorted = branches.slice().sort(function(a, b) { return b.crecimiento - a.crecimiento; });

        var ctx = $('chartBranchGrowth');
        state.charts.branchGrowth = new Chart(ctx, {
          type: 'bar',
          data: {
            labels: sorted.map(function(b) { return b.nombre; }),
            datasets: [{
              label: 'Crecimiento %',
              data: sorted.map(function(b) { return b.crecimiento; }),
              backgroundColor: sorted.map(function(b) {
                return b.crecimiento >= 0 ? '#34d399' : '#f87171';
              }),
              borderRadius: 6
            }]
          },
          options: {
            maintainAspectRatio: false,
            plugins: {
              legend: { display: false },
              tooltip: {
                callbacks: {
                  label: function(ctx) {
                    return 'Crecimiento: ' + fmtPct(ctx.parsed.y);
                  }
                }
              }
            },
            scales: {
              y: {
                ticks: {
                  callback: function(value) { return value + '%'; }
                }
              }
            }
          }
        });
      }

      function renderTopBranchesChart() {
        destroyChart('topBranches');
        
        var branches = getFilteredBranches();
        var metric = els.metric.value;
        els.topBranchMetric.textContent = metricLabels[metric];

        var sorted = branches.slice().sort(function(a, b) {
          return (b[metric] || 0) - (a[metric] || 0);
        }).slice(0, 10);

        var ctx = $('chartTopBranches');
        state.charts.topBranches = new Chart(ctx, {
          type: 'bar',
          data: {
            labels: sorted.map(function(b) { return b.nombre; }),
            datasets: [{
              label: metricLabels[metric],
              data: sorted.map(function(b) { return b[metric] || 0; }),
              backgroundColor: palette,
              borderRadius: 6
            }]
          },
          options: {
            indexAxis: 'y',
            maintainAspectRatio: false,
            plugins: {
              legend: { display: false }
            },
            scales: {
              x: {
                ticks: {
                  callback: function(value) { return fmtNum(value, 0); }
                }
              }
            }
          }
        });
      }

      function renderDepartamentoCharts() {
        destroyChart('departamentoTop');
        destroyChart('departamentoBottom');

        var metric = els.metric.value;
        els.topMetricLabel.textContent = metricLabels[metric];
        els.bottomMetricLabel.textContent = metricLabels[metric];

        var sorted = state.departamentos.slice().sort(function(a, b) {
          return (b[metric] || 0) - (a[metric] || 0);
        });

        var top = sorted.slice(0, 10);
        var bottom = sorted.slice(-10).reverse();

        createHorizontalChart('chartDepartamentoTop', top, metric, true);
        createHorizontalChart('chartDepartamentoBottom', bottom, metric, false);
      }

      function createHorizontalChart(canvasId, rows, metric, isTop) {
        var ctx = $(canvasId);
        var key = canvasId === 'chartDepartamentoTop' ? 'departamentoTop' : 'departamentoBottom';

        state.charts[key] = new Chart(ctx, {
          type: 'bar',
          data: {
            labels: rows.map(function(r) { return r.nombre; }),
            datasets: [{
              label: metricLabels[metric],
              data: rows.map(function(r) { return r[metric] || 0; }),
              backgroundColor: isTop ? '#60a5fa' : '#fbbf24',
              borderRadius: 6
            }]
          },
          options: {
            indexAxis: 'y',
            maintainAspectRatio: false,
            plugins: {
              legend: { display: false }
            },
            scales: {
              x: {
                ticks: {
                  callback: function(value) { return fmtNum(value, 0); }
                }
              }
            }
          }
        });
      }

      function renderBranchTable() {
        var branches = getFilteredBranches().slice().sort(function(a, b) {
          return b.monto - a.monto;
        });

        var html = branches.map(function(b) {
          return '<tr class="hover:bg-slate-800/40">' +
            '<td class="tbody-cell font-medium">' + escapeHtml(b.nombre) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(b.monto, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(b.costo, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(b.utilidad, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtPct(b.margen) + '</td>' +
            '<td class="tbody-cell-right">' + fmtPct(b.participacion) + '</td>' +
            '<td class="tbody-cell-right">' + fmtPct(b.crecimiento) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(b.departamentos, 0) + '</td>' +
            '</tr>';
        }).join('');

        els.branchTableBody.innerHTML = html;
      }

      function renderDepartamentoTable() {
        var rows = state.departamentos.slice().sort(function(a, b) {
          return b.monto - a.monto;
        });

        var html = rows.map(function(r) {
          return '<tr class="hover:bg-slate-800/40">' +
            '<td class="tbody-cell font-medium">' + escapeHtml(r.nombre) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(r.monto, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(r.costo, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtNum(r.utilidad, 2) + '</td>' +
            '<td class="tbody-cell-right">' + fmtPct(r.margen) + '</td>' +
            '<td class="tbody-cell-right">' + fmtPct(r.participacion) + '</td>' +
            '</tr>';
        }).join('');

        els.departamentoBody.innerHTML = html;
      }

      function exportExcel() {
        if (typeof XLSX === 'undefined') {
          alert('No se pudo cargar SheetJS.');
          return;
        }

        var wb = XLSX.utils.book_new();

        var branchData = getFilteredBranches().map(function(b) {
          return {
            'Sucursal': b.nombre,
            'Ventas': round2(b.monto),
            'Costo': round2(b.costo),
            'Utilidad': round2(b.utilidad),
            'Margen %': round2(b.margen),
            'Participación %': round2(b.participacion),
            'Crecimiento %': round2(b.crecimiento),
            'Departamentos': b.departamentos
          };
        });

        var wsBranch = XLSX.utils.json_to_sheet(branchData);
        XLSX.utils.book_append_sheet(wb, wsBranch, 'Sucursales');

        var departamentoData = state.departamentos.map(function(d) {
          return {
            'Departamento': d.nombre,
            'Ventas': round2(d.monto),
            'Costo': round2(d.costo),
            'Utilidad': round2(d.utilidad),
            'Margen %': round2(d.margen),
            'Participación %': round2(d.participacion),
            'Alcance Sucursales': d.alcance
          };
        });

        var wsDepartamento = XLSX.utils.json_to_sheet(departamentoData);
        XLSX.utils.book_append_sheet(wb, wsDepartamento, 'Departamentos');

        XLSX.writeFile(wb, 'dashboard_departamento.xlsx');
      }

      function num(v) {
        if (v === null || v === undefined) return 0;
        if (typeof v === 'number') return isFinite(v) ? v : 0;
        var n = Number(v);
        return isFinite(n) ? n : 0;
      }

      function safeDiv(a, b) {
        return b ? a / b : 0;
      }

      function round2(n) {
        return Math.round((isFinite(n) ? n : 0) * 100) / 100;
      }

      function fmtNum(n, dec) {
        dec = (dec === null || dec === undefined) ? 2 : dec;
        var value = isFinite(n) ? n : 0;
        try {
          return new Intl.NumberFormat('es', {
            minimumFractionDigits: 0,
            maximumFractionDigits: dec
          }).format(value);
        } catch (e) {
          return value.toFixed(dec);
        }
      }

      function fmtPct(n) {
        if (n === null || n === undefined || !isFinite(n)) return 'N/A';
        return fmtNum(n, 2) + '%';
      }

      function escapeHtml(s) {
        return String(s).replace(/[&<>"']/g, function(c) {
          if (c === '&') return '&amp;';
          if (c === '<') return '&lt;';
          if (c === '>') return '&gt;';
          if (c === '"') return '&quot;';
          return '&#39;';
        });
      }

      function destroyChart(key) {
        if (state.charts[key]) {
          state.charts[key].destroy();
          delete state.charts[key];
        }
      }

      function hideLoader() {
        if (els.loader) els.loader.classList.add('hidden');
      }

      function showError(msg) {
        console.error('Error:', msg);
        if (els.loader) {
          els.loader.classList.remove('hidden');
          var errorDiv = document.createElement('div');
          errorDiv.className = 'text-rose-400 text-sm mt-4';
          errorDiv.textContent = msg;
          els.loader.appendChild(errorDiv);
        }
      }
    })();
  </script>
</body>
</html>
        `,
		"__RAW_DATA__",
		string(dataJSON),
		1,
	)

	return []byte(html), nil
}
