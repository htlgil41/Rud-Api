package consts

var (
	QUERY_PREPARE_GET_USUARIO_MODULOS = `
		SELECT m.id, m.nombre, m.tipo, m.descripcion
		FROM modulos m
	 INNER JOIN usuario_modulos um ON um.modulo_id = m.id
	 WHERE um.usuario_id = ?
	   AND m.activo = TRUE
	`

	QUERY_PREPARE_HAS_MODULO = `
		SELECT COUNT(1)
		FROM usuario_modulos um
	 INNER JOIN modulos m ON m.id = um.modulo_id
	 WHERE um.usuario_id = ?
	   AND m.nombre = ?
	   AND m.activo = TRUE
	`

	QUERY_PREPARE_ASSIGN_MODULO = `
		INSERT INTO usuario_modulos (usuario_id, modulo_id)
		VALUES (?, ?)
		ON CONFLICT (usuario_id, modulo_id) DO NOTHING
	`

	QUERY_PREPARE_GET_MODULO_BY_NOMBRE = `
		SELECT id, nombre, tipo, descripcion
		FROM modulos
		WHERE nombre = ? AND activo = TRUE
	`
)
