package consts

var (
	QUERY_PREPARE_CREATE_USUARIO_REPOPG = `
		INSERT INTO usuarios (id, username, email, password_hash, nombre, departamento, activo)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	QUERY_PREPARE_UPDATE_USUARIO_REPOPG = `
		UPDATE usuarios SET username = $1, email = $2, departamento = $3 WHERE id = $4
	`
	QUERY_PREPARE_DELETE_USUARIO_REPOPG = `
		DELETE FROM usuarios WHERE id = $1
	`
	QUERY_PREPARE_ACT_USUARIO_REPOPG = `
		UPDATE usuarios SET activo = $1 WHERE id = $2
	`
	QUERY_PREPARE_ACT_PASSWORD_REPOPG = `
		UPDATE usuarios SET password_hash = $1 WHERE id = $2
	`
	QUERY_PREPARE_GET_USARIOBY_USERNAME = `
		SELECT id, username, email, password_hash, nombre, departamento, activo FROM usuarios WHERE username = $1
	`
)
