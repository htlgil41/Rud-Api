package consts

var (
	QUERY_PREPARE_CREATE_USUARIO_REPOPG = `
		INSERT INTO usuarios (id, username, email, password_hash, nombre, departamento, activo)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	QUERY_PREPARE_UPDATE_USUARIO_REPOPG = `
		UPDATE SET usuarios SET username = '', email = '', departamento = '' WHERE id = ?
	`
	QUERY_PREPARE_DELETE_USUARIO_REPOPG = `
		DELETE FROM usuarios WHERE id = ?
	`
	QUERY_PREPARE_ACT_USUARIO_REPOPG = `
		UPDATE SET usuarios SET activo = ? WHERE id = ?
	`
	QUERY_PREPARE_ACT_PASSWORD_REPOPG = `
		UPDATE SET usuarios SET password_hash = ? WHERE id = ?
	`
	QUERY_PREPARE_GET_USARIOBY_USERNAME = `
		SELECT id, username, email, password_hash, nombre, departamento, activo FROM usuarios WHERE username = ?
	`
)
