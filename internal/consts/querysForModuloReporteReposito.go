package consts

var (
	QUERY_PREPARE_GET_USUARIO_MODULO_REPORTES = `
		SELECT mr.id, mr.nombre, mr.descripcion, mr.params_size
		FROM modulos_reportes mr
		INNER JOIN usuario_modulos_reportes umr ON umr.modulo_reporte_id = mr.id
		WHERE umr.usuario_id = ?
		  AND mr.activo = TRUE
	`

	QUERY_PREPARE_GET_MODULO_REPORTE_BY_ID = `
		SELECT id, nombre, descripcion, params_size
		FROM modulos_reportes
		WHERE id = ? AND activo = TRUE
	`

	QUERY_PREPARE_CREATE_REPORTE_GENERADO = `
		INSERT INTO reportes_generados (id, modulo_reporte_id, usuario_id, estado)
		VALUES (?, ?, ?, 'PENDIENTE')
	`

	QUERY_PREPARE_HAS_MODULO_REPORTE = `
		SELECT COUNT(1)
		FROM usuario_modulos_reportes
		WHERE usuario_id = ?
		  AND modulo_reporte_id = ?
	`
)
