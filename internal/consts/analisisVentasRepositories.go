package consts

var QUERY_DEPARTAMENTO = `SELECT C_CODIGO FROM VAD10..MA_DEPARTAMENTOS`

var QUERY_GRUPO = `SELECT C_CODIGO FROM VAD10..MA_GRUPOS`

var QUERY_SUBGRUPO = `SELECT C_CODIGO FROM VAD10..MA_SUBGRUPOS`

var QUERY_PREPARE_DEPARTAMENTO = `
DECLARE
    @start NVARCHAR(100),
    @end   NVARCHAR(10);

SET @start = @p1;
SET @end = @p2;

SELECT
	CAST(@start AS DATE) FECHA,
	(SELECT TOP 1 TRIM(SSS.c_descripcion) FROM VAD10..MA_SUCURSALES SSS) SUCURSAL,
	DD.c_Descripcio,
	DD.diferencia,
	DD.subtotal,
	DD.cantidad,
	DD.utilidad,
	DD.n_Costo,
	DD.precio,
	DD.costo,
	DD.Utilidad_Per,
	DD.TOTAL,
	DD.COSTOOFERTA
	FROM (
		SELECT c_Descripcio,
		   sum(DIFERENCIA)AS diferencia,
		   sum(subtotal)AS subtotal,
		   sum(cantidad)AS cantidad,
		   sum(utilidad)AS utilidad,
		   sum(n_Costo)AS n_Costo,
		   sum(precio)AS precio,
		   sum(n_Costo)AS costo,
		   SUM(n_Costo) AS Utilidad_Per,
		   sum(TOTAL)AS TOTAL,
		   SUM(CostoOferta)AS COSTOOFERTA
	FROM (
		(SELECT ma_productos.c_Departamento,
				SUM((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)-(TR_INVENTARIO.n_Subtotal)) AS DIFERENCIA,
				MA_DEPARTAMENTOS.c_Descripcio AS c_Descripcio,
				MA_PRODUCTOS.c_Departamento AS c_Codigo,
				SUM(TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto) AS TOTAL,
				SUM(TR_INVENTARIO.n_Precio) AS precio,
				SUM(TR_INVENTARIO.n_Subtotal) AS subtotal,
				SUM(TR_INVENTARIO.n_Costo * TR_INVENTARIO.n_Cantidad) AS n_Costo,
				SUM(TR_INVENTARIO.n_Cantidad) AS CANTIDAD,
				sum(TR_INVENTARIO.n_Precio - TR_INVENTARIO.n_Costo) AS UTILIDAD,
				sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad)AS CostoOferta
			FROM
			(SELECT c_Linea,
					c_Concepto,
					c_Documento,
					c_Deposito,
					c_CodArticulo,
					n_Cantidad,
					c_TipoMov,
					n_Cant_Teorica,
					n_Cant_Diferencia,
					f_Fecha,
					c_CodLocalidad,
					n_FactorCambio,
					c_Descripcion,
					c_Compuesto,
					CodConcepto,
					c_Documento_Origen,
					c_TipoDoc_Origen,
					n_CantidadFac,
					cs_ComprobanteContable,
					cs_CodLocalidad,
					ns_CantidadEmpaque,
					FactorTransaccion,
					FactorHistoricoDestino,
					((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
					(n_Subtotal * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS n_Subtotal,
					(n_Impuesto * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS n_Impuesto,
					(n_Total * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS n_Total,
					(n_Precio * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS n_Precio,
					(n_Precio_Original * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Precio_Original,
					(n_DescuentoGeneral * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_DescuentoGeneral,
					(n_DescuentoEspecifico * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_DescuentoEspecifico,
					(ns_Descuento * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS ns_Descuento,
					(Impuesto * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS Impuesto
			FROM
				(SELECT TI.*,
						MA.d_Fecha AS TmpFechaHoraDoc,
						MA.c_CodMoneda AS CodMonOrig,
						isNULL(
								(SELECT TOP 1 n_FactorPeriodo
								FROM MA_HISTORICO_MONEDAS
								WHERE c_CodMoneda = MON_DEST.c_CodMoneda
									AND d_FechaCambioActual <= MA.d_Fecha
								ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
						TI.n_FactorCambio AS FactorTransaccion,
						MON_DEST.n_Factor AS FactorActualDestino,
						MON_DEST.n_Decimales AS DecMonedaDestino,
						MON_DEST.c_CodMoneda AS CodMonDest,
						CASE
							WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
							ELSE TI.c_CodArticulo
						END AS CodigoProductoHistorico
				FROM TR_INVENTARIO TI
				INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
				AND TI.c_Concepto = MA.c_Concepto
				AND TI.c_CodLocalidad = MA.c_CodLocalidad
				CROSS JOIN
					(SELECT *
					FROM MA_MONEDAS
					WHERE c_CodMoneda = 'USD') MON_DEST
				LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
				LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
				WHERE TI.c_Concepto IN ('VEN',
										'DEV')
					AND TI.f_Fecha BETWEEN @start AND @end +' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
			LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
			LEFT JOIN VAD10.DBO.MA_DEPARTAMENTOS AS MA_DEPARTAMENTOS ON MA_DEPARTAMENTOS.c_Codigo = MA_PRODUCTOS.c_Departamento
			WHERE (TR_INVENTARIO.c_Concepto = 'VEN')
			AND tr_inventario.n_Subtotal > 0
			AND ma_departamentos.c_Codigo IN (%s)
			AND f_Fecha BETWEEN  @start AND @end
			AND LEFT(tr_inventario.c_Documento, 1) = 'P'
			GROUP BY MA_DEPARTAMENTOS.c_Descripcio,
					ma_productos.c_Departamento
			UNION SELECT ma_productos.c_Departamento,
						SUM(((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)-(TR_INVENTARIO.n_Subtotal))*-1) AS DIFERENCIA,
						MA_DEPARTAMENTOS.c_Descripcio AS c_Descripcio,
						MA_PRODUCTOS.c_Departamento AS c_Codigo,
						SUM((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)*-1) AS TOTAL,
						SUM(TR_INVENTARIO.n_Precio*-1) AS precio,
						SUM(TR_INVENTARIO.n_Subtotal*-1) AS subtotal,
						SUM(TR_INVENTARIO.n_Costo * TR_INVENTARIO.n_Cantidad*-1) AS n_Costo,
						SUM(TR_INVENTARIO.n_Cantidad) AS CANTIDAD,
						sum((TR_INVENTARIO.n_Precio - TR_INVENTARIO.n_Costo)*-1) AS UTILIDAD,
						sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad*-1)AS CostoOferta
			FROM
			(SELECT c_Linea,
					c_Concepto,
					c_Documento,
					c_Deposito,
					c_CodArticulo,
					n_Cantidad,
					c_TipoMov,
					n_Cant_Teorica,
					n_Cant_Diferencia,
					f_Fecha,
					c_CodLocalidad,
					n_FactorCambio,
					c_Descripcion,
					c_Compuesto,
					CodConcepto,
					c_Documento_Origen,
					c_TipoDoc_Origen,
					n_CantidadFac,
					cs_ComprobanteContable,
					cs_CodLocalidad,
					ns_CantidadEmpaque,
					FactorTransaccion,
					FactorHistoricoDestino,
					((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
					(n_Subtotal * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS n_Subtotal,
					(n_Impuesto * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS n_Impuesto,
					(n_Total * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS n_Total,
					(n_Precio * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS n_Precio,
					(n_Precio_Original * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Precio_Original,
					(n_DescuentoGeneral * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_DescuentoGeneral,
					(n_DescuentoEspecifico * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_DescuentoEspecifico,
					(ns_Descuento * (CASE
										WHEN CodMonOrig = CodMonDest THEN 1
										ELSE FactorTransaccion / FactorHistoricoDestino
									END)) AS ns_Descuento,
					(Impuesto * (CASE
									WHEN CodMonOrig = CodMonDest THEN 1
									ELSE FactorTransaccion / FactorHistoricoDestino
								END)) AS Impuesto
			FROM
				(SELECT TI.*,
						MA.d_Fecha AS TmpFechaHoraDoc,
						MA.c_CodMoneda AS CodMonOrig,
						isNULL(
								(SELECT TOP 1 n_FactorPeriodo
								FROM MA_HISTORICO_MONEDAS
								WHERE c_CodMoneda = MON_DEST.c_CodMoneda
									AND d_FechaCambioActual <= MA.d_Fecha
								ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
						TI.n_FactorCambio AS FactorTransaccion,
						MON_DEST.n_Factor AS FactorActualDestino,
						MON_DEST.n_Decimales AS DecMonedaDestino,
						MON_DEST.c_CodMoneda AS CodMonDest,
						CASE
							WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
							ELSE TI.c_CodArticulo
						END AS CodigoProductoHistorico
				FROM TR_INVENTARIO TI
				INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
				AND TI.c_Concepto = MA.c_Concepto
				AND TI.c_CodLocalidad = MA.c_CodLocalidad
				CROSS JOIN
					(SELECT *
					FROM MA_MONEDAS
					WHERE c_CodMoneda = 'USD') MON_DEST
				LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
				LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
				WHERE TI.c_Concepto IN ('VEN',
										'DEV')
					AND TI.f_Fecha BETWEEN @start AND @end + ' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
			LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
			LEFT JOIN VAD10.DBO.MA_DEPARTAMENTOS AS MA_DEPARTAMENTOS ON MA_DEPARTAMENTOS.c_Codigo = MA_PRODUCTOS.c_Departamento
			WHERE (TR_INVENTARIO.c_Concepto = 'DEV')
			AND tr_inventario.n_Cantidad > 0
			AND TR_INVENTARIO.n_Subtotal > 0
			AND ma_departamentos.c_Codigo IN (%s)
			AND f_Fecha BETWEEN  @start AND @end
			AND LEFT(tr_inventario.c_Documento, 1) = 'P'
			GROUP BY MA_DEPARTAMENTOS.c_Descripcio,
					ma_productos.c_Departamento)) AS TR_INVENTARIO
	GROUP BY c_Descripcio
	) DD
`

var QUERY_PREPARE_GRUPO = `
DECLARE
    @start NVARCHAR(100),
    @end   NVARCHAR(10);

SET @start = @p1;
SET @end = @p2;
SELECT
	CAST(@start AS DATE) fecha,
	(SELECT TOP 1 TRIM(SSS.c_descripcion) FROM VAD10..MA_SUCURSALES SSS) SUCURSAL,
	SUB.c_Descripcio grupo,
	ISNULL((SELECT D.C_DESCRIPCIO FROM VAD10..MA_DEPARTAMENTOS D WHERE D.C_CODIGO = SUB.c_Departamento), 'ANY') departamento,
	SUB.total,
	SUB.precio,
	SUB.subtotal,
	SUB.n_Cantidad,
	SUB.n_Costo,
	SUB.Utilidad_Per,
	SUB.cantidad,
	SUB.utilidad,
	SUB.costooferta
	FROM (
			SELECT
				c_Descripcio,
				c_Departamento,
				sum(total)AS total,
				sum(precio)AS precio,
				sum(subtotal)AS subtotal,
				sum(n_Cantidad)AS n_Cantidad,
				sum(n_Costo)AS n_Costo,
				SUM(n_Costo) AS Utilidad_Per,
				sum(cantidad)AS cantidad,
				sum(utilidad)AS utilidad,
				sum(costooferta)AS costooferta
		FROM (
						(SELECT (ma_productos.c_Departamento + ma_productos.c_Grupo) AS SLLAVEG,
										isNULL(MA_GRUPOS.c_Codigo, 'N/A') AS c_Codigo,
										ma_productos.c_Departamento AS c_Departamento,
										isNULL(MA_GRUPOS.c_Descripcio, 'N/A') AS c_Descripcio,
										SUM(TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto) AS TOTAL,
										SUM(TR_INVENTARIO.n_Cantidad) AS n_Cantidad,
										SUM(TR_INVENTARIO.n_Precio) AS precio,
										SUM(TR_INVENTARIO.n_Subtotal) AS subtotal,
										SUM(TR_INVENTARIO.n_Costo*tr_inventario.n_Cantidad) AS n_Costo,
										SUM (TR_INVENTARIO.n_Cantidad) AS CANTIDAD,
												sum(TR_INVENTARIO.n_Precio-TR_INVENTARIO.n_Costo) AS UTILIDAD,
												sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad)AS CostoOferta
						FROM
						(SELECT c_Linea,
										c_Concepto,
										c_Documento,
										c_Deposito,
										c_CodArticulo,
										n_Cantidad,
										c_TipoMov,
										n_Cant_Teorica,
										n_Cant_Diferencia,
										f_Fecha,
										c_CodLocalidad,
										n_FactorCambio,
										c_Descripcion,
										c_Compuesto,
										CodConcepto,
										c_Documento_Origen,
										c_TipoDoc_Origen,
										n_CantidadFac,
										cs_ComprobanteContable,
										cs_CodLocalidad,
										ns_CantidadEmpaque,
										FactorTransaccion,
										FactorHistoricoDestino,
										((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
										(n_Subtotal * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Subtotal,
										(n_Impuesto * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Impuesto,
										(n_Total * (CASE
																		WHEN CodMonOrig = CodMonDest THEN 1
																		ELSE FactorTransaccion / FactorHistoricoDestino
																END)) AS n_Total,
										(n_Precio * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Precio,
										(n_Precio_Original * (CASE
																								WHEN CodMonOrig = CodMonDest THEN 1
																								ELSE FactorTransaccion / FactorHistoricoDestino
																						END)) AS n_Precio_Original,
										(n_DescuentoGeneral * (CASE
																								WHEN CodMonOrig = CodMonDest THEN 1
																								ELSE FactorTransaccion / FactorHistoricoDestino
																						END)) AS n_DescuentoGeneral,
										(n_DescuentoEspecifico * (CASE
																										WHEN CodMonOrig = CodMonDest THEN 1
																										ELSE FactorTransaccion / FactorHistoricoDestino
																								END)) AS n_DescuentoEspecifico,
										(ns_Descuento * (CASE
																						WHEN CodMonOrig = CodMonDest THEN 1
																						ELSE FactorTransaccion / FactorHistoricoDestino
																				END)) AS ns_Descuento,
										(Impuesto * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS Impuesto
								FROM
								(SELECT TI.*,
												MA.d_Fecha AS TmpFechaHoraDoc,
												MA.c_CodMoneda AS CodMonOrig,
												isNULL(
																(SELECT TOP 1 n_FactorPeriodo
																		FROM MA_HISTORICO_MONEDAS
																		WHERE c_CodMoneda = MON_DEST.c_CodMoneda
																		AND d_FechaCambioActual <= MA.d_Fecha
																		ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
												TI.n_FactorCambio AS FactorTransaccion,
												MON_DEST.n_Factor AS FactorActualDestino,
												MON_DEST.n_Decimales AS DecMonedaDestino,
												MON_DEST.c_CodMoneda AS CodMonDest,
												CASE
														WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
														ELSE TI.c_CodArticulo
												END AS CodigoProductoHistorico
								FROM TR_INVENTARIO TI
								INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
								AND TI.c_Concepto = MA.c_Concepto
								AND TI.c_CodLocalidad = MA.c_CodLocalidad
								CROSS JOIN
										(SELECT *
										FROM MA_MONEDAS
										WHERE c_CodMoneda = 'USD') MON_DEST
								LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
								LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
								WHERE TI.c_Concepto IN ('VEN',
																				'DEV')
										AND TI.f_Fecha BETWEEN @start AND @end +' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
						LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
						LEFT JOIN VAD10.DBO.MA_GRUPOS AS MA_GRUPOS ON MA_GRUPOS.c_Codigo = MA_PRODUCTOS.c_Grupo
						AND MA_GRUPOS.c_Departamento = MA_PRODUCTOS.c_Departamento
						WHERE (tr_inventario.c_Concepto = 'VEN')
						AND tr_inventario.n_Subtotal > 0
						AND ma_grupos.c_Departamento IN (%s)
						AND ma_grupos.c_Codigo IN (%s)
						AND f_Fecha BETWEEN @start AND @end
						AND LEFT(tr_inventario.c_Documento, 1) = 'P'
						GROUP BY MA_GRUPOS.c_Codigo,
										MA_GRUPOS.c_Departamento,
										MA_GRUPOS.c_Descripcio,
										ma_productos.c_Departamento,
										ma_productos.c_Grupo
						UNION SELECT (ma_productos.c_Departamento + ma_productos.c_Grupo) AS SLLAVEG,
												isNULL(MA_GRUPOS.c_Codigo, 'N/A') AS c_Codigo,
												ma_productos.c_Departamento AS c_Departamento,
												isNULL(MA_GRUPOS.c_Descripcio, 'N/A') AS c_Descripcio,
												SUM((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)*-1) AS TOTAL,
												SUM(TR_INVENTARIO.n_Cantidad*-1) AS n_Cantidad,
												SUM(TR_INVENTARIO.n_Precio*-1) AS precio,
												SUM(TR_INVENTARIO.n_Subtotal*-1) AS subtotal,
												SUM(TR_INVENTARIO.n_Costo*tr_inventario.n_Cantidad*-1) AS n_Costo,
												SUM (TR_INVENTARIO.n_Cantidad*-1) AS CANTIDAD,
														sum((TR_INVENTARIO.n_Precio-TR_INVENTARIO.n_Costo)*-1) AS UTILIDAD,
														sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad*-1)AS CostoOferta
						FROM
						(SELECT c_Linea,
										c_Concepto,
										c_Documento,
										c_Deposito,
										c_CodArticulo,
										n_Cantidad,
										c_TipoMov,
										n_Cant_Teorica,
										n_Cant_Diferencia,
										f_Fecha,
										c_CodLocalidad,
										n_FactorCambio,
										c_Descripcion,
										c_Compuesto,
										CodConcepto,
										c_Documento_Origen,
										c_TipoDoc_Origen,
										n_CantidadFac,
										cs_ComprobanteContable,
										cs_CodLocalidad,
										ns_CantidadEmpaque,
										FactorTransaccion,
										FactorHistoricoDestino,
										((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
										(n_Subtotal * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Subtotal,
										(n_Impuesto * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Impuesto,
										(n_Total * (CASE
																		WHEN CodMonOrig = CodMonDest THEN 1
																		ELSE FactorTransaccion / FactorHistoricoDestino
																END)) AS n_Total,
										(n_Precio * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS n_Precio,
										(n_Precio_Original * (CASE
																								WHEN CodMonOrig = CodMonDest THEN 1
																								ELSE FactorTransaccion / FactorHistoricoDestino
																						END)) AS n_Precio_Original,
										(n_DescuentoGeneral * (CASE
																								WHEN CodMonOrig = CodMonDest THEN 1
																								ELSE FactorTransaccion / FactorHistoricoDestino
																						END)) AS n_DescuentoGeneral,
										(n_DescuentoEspecifico * (CASE
																										WHEN CodMonOrig = CodMonDest THEN 1
																										ELSE FactorTransaccion / FactorHistoricoDestino
																								END)) AS n_DescuentoEspecifico,
										(ns_Descuento * (CASE
																						WHEN CodMonOrig = CodMonDest THEN 1
																						ELSE FactorTransaccion / FactorHistoricoDestino
																				END)) AS ns_Descuento,
										(Impuesto * (CASE
																				WHEN CodMonOrig = CodMonDest THEN 1
																				ELSE FactorTransaccion / FactorHistoricoDestino
																		END)) AS Impuesto
								FROM
								(SELECT TI.*,
												MA.d_Fecha AS TmpFechaHoraDoc,
												MA.c_CodMoneda AS CodMonOrig,
												isNULL(
																(SELECT TOP 1 n_FactorPeriodo
																		FROM MA_HISTORICO_MONEDAS
																		WHERE c_CodMoneda = MON_DEST.c_CodMoneda
																		AND d_FechaCambioActual <= MA.d_Fecha
																		ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
												TI.n_FactorCambio AS FactorTransaccion,
												MON_DEST.n_Factor AS FactorActualDestino,
												MON_DEST.n_Decimales AS DecMonedaDestino,
												MON_DEST.c_CodMoneda AS CodMonDest,
												CASE
														WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
														ELSE TI.c_CodArticulo
												END AS CodigoProductoHistorico
								FROM TR_INVENTARIO TI
								INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
								AND TI.c_Concepto = MA.c_Concepto
								AND TI.c_CodLocalidad = MA.c_CodLocalidad
								CROSS JOIN
										(SELECT *
										FROM MA_MONEDAS
										WHERE c_CodMoneda = 'USD') MON_DEST
								LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
								LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
								WHERE TI.c_Concepto IN ('VEN',
																				'DEV')
										AND TI.f_Fecha BETWEEN @start AND @end +' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
						LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
						LEFT JOIN VAD10.DBO.MA_GRUPOS AS MA_GRUPOS ON MA_GRUPOS.c_Codigo = MA_PRODUCTOS.c_Grupo
						AND MA_GRUPOS.c_Departamento = MA_PRODUCTOS.c_Departamento
						WHERE (tr_inventario.c_Concepto = 'DEV')
						AND tr_inventario.n_Cantidad > 0
						AND tr_inventario.n_Subtotal > 0
						AND ma_grupos.c_Departamento IN (%s)
						AND ma_grupos.c_Codigo IN (%s)
						AND f_Fecha BETWEEN @start AND @end
						AND LEFT(tr_inventario.c_Documento, 1) = 'P'
						GROUP BY MA_GRUPOS.c_Codigo,
										MA_GRUPOS.c_Departamento,
										MA_GRUPOS.c_Descripcio,
										ma_productos.c_Departamento,
										ma_productos.c_Grupo)) AS tr_inventario
		GROUP BY
			c_Departamento,
			c_Descripcio
	) SUB;
`

var QUERY_PREPAPRE_SUBGRUPO = `
DECLARE
	@start NVARCHAR(100),
	@end   NVARCHAR(10);

SET @start = @p1;
SET @end = @p2;

SELECT
	CAST(@start AS DATE) FECHA,
	(SELECT TOP 1 TRIM(SSS.c_descripcion) FROM VAD10..MA_SUCURSALES SSS) SUCURSAL,
	SUB.DEPARTAMENTO,
	SUB.GRUPO,
	SUB.c_Descripcio SUBGRUPO,
	SUB.total,
	SUB.n_Cantidad,
	SUB.cantidad,
	SUB.precio,
	SUB.subtotal,
	SUB.n_Costo,
	SUB.Utilidad_Per,
	SUB.UTILIDAD,
	SUB.costooferta
	FROM (
		SELECT
			ISNULL((SELECT DD.C_DESCRIPCIO FROM VAD10..MA_DEPARTAMENTOS DD WHERE DD.C_CODIGO = SB.c_in_departamento), 'ANY') DEPARTAMENTO,
			ISNULL((SELECT G.C_DESCRIPCIO FROM VAD10..MA_GRUPOS G WHERE G.c_departamento = SB.c_in_departamento AND G.c_CODIGO = SB.c_in_grupo), 'ANY') GRUPO,
			SB.c_Descripcio,
			SB.total,
			SB.n_Cantidad,
			SB.cantidad,
			SB.precio,
			SB.subtotal,
			SB.n_Costo,
			SB.Utilidad_Per,
			SB.UTILIDAD,
			SB.costooferta
			FROM (
				SELECT 
					c_Subgrupo c_in_departamento,
					c_in_grupo,
					c_Descripcio,
					sum(total)AS total,
					sum(n_Cantidad)AS n_Cantidad,
					sum(cantidad)AS cantidad,
					sum(precio)AS precio,
					sum(subtotal)AS subtotal,
					sum(n_Costo)AS n_Costo,
					SUM(n_Costo) AS Utilidad_Per,
					sum(utilidad)AS UTILIDAD,
					sum(costooferta)AS costooferta
				FROM(
					(SELECT (ma_productos.c_Departamento + ma_productos.c_Grupo) AS SLLAVE1,
							(ma_productos.c_Departamento + ma_productos.c_Grupo + ma_productos.c_Subgrupo) AS SLLAVE,
							MA_PRODUCTOS.c_Subgrupo AS c_Subgrupo,
							ma_productos.c_Departamento AS c_in_departamento,
							MA_SUBGRUPOS.C_IN_GRUPO AS c_in_grupo,
							isNULL(MA_SUBGRUPOS.c_Codigo, 'N/A') AS c_Codigo,
							isNULL(MA_SUBGRUPOS.c_Descripcio, 'N/A') AS c_Descripcio,
							SUM(TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto) AS TOTAL,
							SUM(TR_INVENTARIO.n_Cantidad) AS n_Cantidad,
							SUM(TR_INVENTARIO.n_Precio) AS precio,
							SUM(TR_INVENTARIO.n_Subtotal) AS subtotal,
							SUM(TR_INVENTARIO.n_Costo * TR_INVENTARIO.n_Cantidad) AS n_Costo,
							SUM(TR_INVENTARIO.n_Cantidad) AS CANTIDAD,
							sum((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)-TR_INVENTARIO.n_Costo) AS UTILIDAD,
							sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad)AS CostoOferta
						FROM
						(SELECT c_Linea,
								c_Concepto,
								c_Documento,
								c_Deposito,
								c_CodArticulo,
								n_Cantidad,
								c_TipoMov,
								n_Cant_Teorica,
								n_Cant_Diferencia,
								f_Fecha,
								c_CodLocalidad,
								n_FactorCambio,
								c_Descripcion,
								c_Compuesto,
								CodConcepto,
								c_Documento_Origen,
								c_TipoDoc_Origen,
								n_CantidadFac,
								cs_ComprobanteContable,
								cs_CodLocalidad,
								ns_CantidadEmpaque,
								FactorTransaccion,
								FactorHistoricoDestino,
								((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
								(n_Subtotal * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_Subtotal,
								(n_Impuesto * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_Impuesto,
								(n_Total * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Total,
								(n_Precio * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Precio,
								(n_Precio_Original * (CASE
															WHEN CodMonOrig = CodMonDest THEN 1
															ELSE FactorTransaccion / FactorHistoricoDestino
														END)) AS n_Precio_Original,
								(n_DescuentoGeneral * (CASE
															WHEN CodMonOrig = CodMonDest THEN 1
															ELSE FactorTransaccion / FactorHistoricoDestino
														END)) AS n_DescuentoGeneral,
								(n_DescuentoEspecifico * (CASE
																WHEN CodMonOrig = CodMonDest THEN 1
																ELSE FactorTransaccion / FactorHistoricoDestino
															END)) AS n_DescuentoEspecifico,
								(ns_Descuento * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS ns_Descuento,
								(Impuesto * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS Impuesto
						FROM
							(SELECT TI.*,
									MA.d_Fecha AS TmpFechaHoraDoc,
									MA.c_CodMoneda AS CodMonOrig,
									isNULL(
											(SELECT TOP 1 n_FactorPeriodo
											FROM MA_HISTORICO_MONEDAS
											WHERE c_CodMoneda = MON_DEST.c_CodMoneda
												AND d_FechaCambioActual <= MA.d_Fecha
											ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
									TI.n_FactorCambio AS FactorTransaccion,
									MON_DEST.n_Factor AS FactorActualDestino,
									MON_DEST.n_Decimales AS DecMonedaDestino,
									MON_DEST.c_CodMoneda AS CodMonDest,
									CASE
										WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
										ELSE TI.c_CodArticulo
									END AS CodigoProductoHistorico
							FROM TR_INVENTARIO TI
							INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
							AND TI.c_Concepto = MA.c_Concepto
							AND TI.c_CodLocalidad = MA.c_CodLocalidad
							CROSS JOIN
								(SELECT *
								FROM MA_MONEDAS
								WHERE c_CodMoneda = 'USD') MON_DEST
							LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
							LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
							WHERE TI.c_Concepto IN ('VEN',
													'DEV')
								AND TI.f_Fecha BETWEEN @start AND @end +' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
						LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
						LEFT JOIN VAD10.DBO.MA_SUBGRUPOS AS MA_SUBGRUPOS ON MA_SUBGRUPOS.c_Codigo = MA_PRODUCTOS.c_Subgrupo
						AND MA_SUBGRUPOS.C_IN_DEPARTAMENTO = MA_PRODUCTOS.c_Departamento
						AND MA_SUBGRUPOS.C_IN_GRUPO = MA_PRODUCTOS.c_Grupo
						WHERE (TR_INVENTARIO.c_Concepto = 'VEN')
						AND tr_inventario.n_Subtotal > 0
						AND ma_subgrupos.c_in_departamento IN (%s)
						AND ma_subgrupos.c_in_grupo IN (%s)
						AND ma_subgrupos.c_Codigo IN (%s)
						AND f_Fecha BETWEEN @start AND @end
						AND LEFT(tr_inventario.c_Documento, 1) = 'P'
						GROUP BY MA_SUBGRUPOS.c_Codigo,
								MA_SUBGRUPOS.C_IN_DEPARTAMENTO,
								MA_SUBGRUPOS.C_IN_GRUPO,
								MA_SUBGRUPOS.c_Descripcio,
								ma_productos.c_Departamento,
								ma_productos.c_Grupo,
								ma_productos.c_Subgrupo
						UNION SELECT (ma_productos.c_Departamento + ma_productos.c_Grupo) AS SLLAVE1,
									(ma_productos.c_Departamento + ma_productos.c_Grupo + ma_productos.c_Subgrupo) AS SLLAVE,
									MA_PRODUCTOS.c_Subgrupo AS c_Subgrupo,
									MA_SUBGRUPOS.C_IN_DEPARTAMENTO AS c_in_departamento,
									MA_SUBGRUPOS.C_IN_GRUPO AS c_in_grupo,
									isNULL(MA_SUBGRUPOS.c_Codigo, 'N/A') AS c_Codigo,
									isNULL(MA_SUBGRUPOS.c_Descripcio, 'N/A') AS c_Descripcio,
									SUM((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)*-1) AS TOTAL,
									SUM(TR_INVENTARIO.n_Cantidad*-1) AS n_Cantidad,
									SUM(TR_INVENTARIO.n_Precio*-1) AS precio,
									SUM(TR_INVENTARIO.n_Subtotal*-1) AS subtotal,
									SUM(TR_INVENTARIO.n_Costo * TR_INVENTARIO.n_Cantidad*-1) AS n_Costo,
									SUM(TR_INVENTARIO.n_Cantidad*-1) AS CANTIDAD,
									sum(((TR_INVENTARIO.n_Total - TR_INVENTARIO.n_Impuesto)-TR_INVENTARIO.n_Costo)*-1) AS UTILIDAD,
									sum((TR_INVENTARIO.n_Precio_Original - TR_INVENTARIO.n_Precio)* TR_INVENTARIO.n_Cantidad*-1)AS CostoOferta
						FROM
						(SELECT c_Linea,
								c_Concepto,
								c_Documento,
								c_Deposito,
								c_CodArticulo,
								n_Cantidad,
								c_TipoMov,
								n_Cant_Teorica,
								n_Cant_Diferencia,
								f_Fecha,
								c_CodLocalidad,
								n_FactorCambio,
								c_Descripcion,
								c_Compuesto,
								CodConcepto,
								c_Documento_Origen,
								c_TipoDoc_Origen,
								n_CantidadFac,
								cs_ComprobanteContable,
								cs_CodLocalidad,
								ns_CantidadEmpaque,
								FactorTransaccion,
								FactorHistoricoDestino,
								((n_Costo * FactorTransaccion) / FactorHistoricoDestino) AS n_Costo,
								(n_Subtotal * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_Subtotal,
								(n_Impuesto * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS n_Impuesto,
								(n_Total * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Total,
								(n_Precio * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS n_Precio,
								(n_Precio_Original * (CASE
															WHEN CodMonOrig = CodMonDest THEN 1
															ELSE FactorTransaccion / FactorHistoricoDestino
														END)) AS n_Precio_Original,
								(n_DescuentoGeneral * (CASE
															WHEN CodMonOrig = CodMonDest THEN 1
															ELSE FactorTransaccion / FactorHistoricoDestino
														END)) AS n_DescuentoGeneral,
								(n_DescuentoEspecifico * (CASE
																WHEN CodMonOrig = CodMonDest THEN 1
																ELSE FactorTransaccion / FactorHistoricoDestino
															END)) AS n_DescuentoEspecifico,
								(ns_Descuento * (CASE
													WHEN CodMonOrig = CodMonDest THEN 1
													ELSE FactorTransaccion / FactorHistoricoDestino
												END)) AS ns_Descuento,
								(Impuesto * (CASE
												WHEN CodMonOrig = CodMonDest THEN 1
												ELSE FactorTransaccion / FactorHistoricoDestino
											END)) AS Impuesto
						FROM
							(SELECT TI.*,
									MA.d_Fecha AS TmpFechaHoraDoc,
									MA.c_CodMoneda AS CodMonOrig,
									isNULL(
											(SELECT TOP 1 n_FactorPeriodo
											FROM MA_HISTORICO_MONEDAS
											WHERE c_CodMoneda = MON_DEST.c_CodMoneda
												AND d_FechaCambioActual <= MA.d_Fecha
											ORDER BY d_FechaCambioActual DESC, ID DESC), MON_DEST.n_Factor) AS FactorHistoricoDestino,
									TI.n_FactorCambio AS FactorTransaccion,
									MON_DEST.n_Factor AS FactorActualDestino,
									MON_DEST.n_Decimales AS DecMonedaDestino,
									MON_DEST.c_CodMoneda AS CodMonDest,
									CASE
										WHEN PADRE.c_Cod_Plantilla IN ('301000001') THEN PROD.c_Codigo_Base
										ELSE TI.c_CodArticulo
									END AS CodigoProductoHistorico
							FROM TR_INVENTARIO TI
							INNER JOIN MA_VENTAS MA ON TI.c_Documento = MA.c_Documento
							AND TI.c_Concepto = MA.c_Concepto
							AND TI.c_CodLocalidad = MA.c_CodLocalidad
							CROSS JOIN
								(SELECT *
								FROM MA_MONEDAS
								WHERE c_CodMoneda = 'USD') MON_DEST
							LEFT JOIN MA_PRODUCTOS PROD ON TI.c_CodArticulo = PROD.c_Codigo
							LEFT JOIN MA_PRODUCTOS PADRE ON PADRE.c_Codigo = PROD.c_Codigo_Base
							WHERE TI.c_Concepto IN ('VEN',
													'DEV')
								AND TI.f_Fecha BETWEEN @start AND @end +' 23:59:59.000') AS TR_INVENTA) AS TR_INVENTARIO
						LEFT JOIN VAD10.DBO.MA_PRODUCTOS AS MA_PRODUCTOS ON TR_INVENTARIO.c_CodArticulo = MA_PRODUCTOS.c_Codigo
						LEFT JOIN VAD10.DBO.MA_SUBGRUPOS AS MA_SUBGRUPOS ON MA_SUBGRUPOS.c_Codigo = MA_PRODUCTOS.c_Subgrupo
						AND MA_SUBGRUPOS.C_IN_DEPARTAMENTO = MA_PRODUCTOS.c_Departamento
						AND MA_SUBGRUPOS.C_IN_GRUPO = MA_PRODUCTOS.c_Grupo
						WHERE (TR_INVENTARIO.c_Concepto = 'DEV')
						AND tr_inventario.n_Cantidad > 0
						AND tr_inventario.n_Subtotal > 0
						AND ma_subgrupos.c_in_departamento IN (%s)
						AND ma_subgrupos.c_in_grupo IN (%s)
						AND ma_subgrupos.c_Codigo IN (%s)
						AND f_Fecha BETWEEN @start AND @end
						AND LEFT(tr_inventario.c_Documento, 1) = 'P'
						GROUP BY MA_SUBGRUPOS.c_Codigo,
								MA_SUBGRUPOS.C_IN_DEPARTAMENTO,
								MA_SUBGRUPOS.C_IN_GRUPO,
								MA_SUBGRUPOS.c_Descripcio,
								ma_productos.c_Departamento,
								ma_productos.c_Grupo,
								ma_productos.c_Subgrupo)) AS tr_inventario
				GROUP BY 
						c_Descripcio,
						c_in_departamento,
						c_in_grupo,
						c_Subgrupo
			) SB
	) SUB
	WHERE SUB.DEPARTAMENTO != 'ANY'
	AND SUB.GRUPO != 'ANY'
`
