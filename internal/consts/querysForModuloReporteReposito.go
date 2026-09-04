package consts

var (
	QUERY_PREPARE_GET_USUARIO_MODULO_REPORTES = `
		SELECT mr.id, mr.nombre, mr.descripcion, mr.params_size
		FROM modulos_reportes mr
		INNER JOIN usuario_modulos_reportes umr ON umr.modulo_reporte_id = mr.id
		WHERE umr.usuario_id = $1
		  AND mr.activo = TRUE
	`

	QUERY_PREPARE_GET_MODULO_REPORTE_BY_ID = `
		SELECT id, nombre, descripcion, params_size
		FROM modulos_reportes
		WHERE id = $1 AND activo = TRUE
	`

	QUERY_PREPARE_CREATE_REPORTE_GENERADO = `
		INSERT INTO reportes_generados (id, modulo_reporte_id, usuario_id, estado)
		VALUES ($1, $2, $3, 'PENDIENTE')
	`

	QUERY_PREPARE_HAS_MODULO_REPORTE = `
		SELECT COUNT(1)
		FROM usuario_modulos_reportes
		WHERE usuario_id = $1
		  AND modulo_reporte_id = $2
	`

	QUERY_PREPARE_ASSIGN_MODULO_REPORTE = `
		INSERT INTO usuario_modulos_reportes (usuario_id, modulo_reporte_id)
		VALUES ($1, $2)
		ON CONFLICT (usuario_id, modulo_reporte_id) DO NOTHING
	`

	QUERY_PREPARE_DELETE_ASIGNACION_MODULO_REPORTE = `
		DELETE FROM usuario_modulos_reportes
		WHERE usuario_id = $1
		  AND modulo_reporte_id = $2
	`

	QUERY_PREPARE_CREATE_MODULO_REPORTE = `
		INSERT INTO modulos_reportes (id, nombre, descripcion, params_size, query_plane, query_prepare)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	QUERY_PREPARE_GET_ALL_MODULO_REPORTES = `
		SELECT id, nombre, descripcion, params_size
		FROM modulos_reportes
		WHERE activo = TRUE
		ORDER BY nombre
	`

	QUERY_PREPARE_CANCEL_REPORTE_GENERADO = `
		UPDATE reportes_generados
		SET estado = 'NO_AVAILABLE'
		WHERE id = $1 AND usuario_id = $2
	`

	QUERY_PREPARE_GET_REPORTES_GENERADOS_BY_USUARIO = `
		SELECT id, modulo_reporte_id, usuario_id, estado, solicitado_en
		FROM reportes_generados
		WHERE usuario_id = $1
		  AND estado != 'NO_AVAILABLE'
		  AND ($2::varchar IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT 16
	`

	QUERY_PREPARE_GET_ALL_REPORTES_GENERADOS = `
		SELECT id, modulo_reporte_id, usuario_id, estado, solicitado_en
		FROM reportes_generados
		WHERE ($1::varchar IS NULL OR id < $1)
		ORDER BY id DESC
		LIMIT 16
	`
)
