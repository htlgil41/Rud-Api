package repositories

import (
	"context"
	"fmt"
	"rud-api/internal/consts"
	"rud-api/internal/types"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsuarioRepositoriePg struct {
	Pool *pgxpool.Pool
}

func (r *UsuarioRepositoriePg) RowsAffectedCommandResult(command pgconn.CommandTag) error {
	if rowsAffected := command.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("No se ha podido realizar la operacion")
	}

	return nil
}

func (r *UsuarioRepositoriePg) CreateUsuario(new_usuaro types.UsuarioRespositorie) (types.UsuarioRespositorie, error) {
	commandCreateUsuario, errCommandCreateUsuario := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_CREATE_USUARIO_REPOPG,
		new_usuaro.ID,
		new_usuaro.Usernme,
		new_usuaro.Email,
		new_usuaro.PasswordHash,
		new_usuaro.Nombre,
		new_usuaro.Departamento,
		new_usuaro.Activo,
	)
	if errCommandCreateUsuario != nil {
		return types.UsuarioRespositorie{}, errCommandCreateUsuario
	}
	can := r.RowsAffectedCommandResult(commandCreateUsuario)
	if can != nil {
		return new_usuaro, nil
	}
	return new_usuaro, can
}

func (r *UsuarioRepositoriePg) GetUsuarioByUsername(username string) (types.UsuarioRespositorie, error) {
	var value types.UsuarioRespositorie
	erroScanUsuario := r.Pool.QueryRow(
		context.Background(),
		consts.QUERY_PREPARE_GET_USARIOBY_USERNAME,
		username,
	).Scan(
		&value.ID,
		&value.Usernme,
		&value.Email,
		&value.PasswordHash,
		&value.Nombre,
		&value.Departamento,
		&value.Activo,
	)
	if erroScanUsuario != nil {
		return value, erroScanUsuario
	}

	return value, erroScanUsuario
}

func (r *UsuarioRepositoriePg) UpdateDetailUsuario(update_usuario types.UsuarioRespositorie) (types.UsuarioRespositorie, error) {
	commandUpdateUsuario, errCommandUpdateUsuario := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_UPDATE_USUARIO_REPOPG,
	)
	if errCommandUpdateUsuario != nil {
		return types.UsuarioRespositorie{}, errCommandUpdateUsuario
	}

	can := r.RowsAffectedCommandResult(commandUpdateUsuario)
	if can != nil {
		return update_usuario, nil
	}

	return update_usuario, can
}

func (r *UsuarioRepositoriePg) DeleteUsuario(usuario_id string) (bool, error) {
	commandDeleteUsuario, errCommandDeleteUsuario := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_DELETE_USUARIO_REPOPG,
		usuario_id,
	)
	if errCommandDeleteUsuario != nil {
		return false, errCommandDeleteUsuario
	}

	can := r.RowsAffectedCommandResult(commandDeleteUsuario)
	if can != nil {
		return true, nil
	}
	return false, r.RowsAffectedCommandResult(commandDeleteUsuario)
}

func (r *UsuarioRepositoriePg) ActUsuario(act bool, user_id string) (bool, error) {
	commandActUsuario, errCommandActUsuario := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_ACT_USUARIO_REPOPG,
		act, user_id,
	)
	if errCommandActUsuario != nil {
		return false, errCommandActUsuario
	}

	can := r.RowsAffectedCommandResult(commandActUsuario)
	if can != nil {
		return true, nil
	}
	return false, can
}

func (r *UsuarioRepositoriePg) ChangePassowordUsuario(password string, id_usuario string) (bool, error) {
	commandUpdatePassword, errCommandUpdatePassword := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_ACT_PASSWORD_REPOPG,
		password,
		id_usuario,
	)
	if errCommandUpdatePassword != nil {
		return false, errCommandUpdatePassword
	}

	can := r.RowsAffectedCommandResult(commandUpdatePassword)
	if can != nil {
		return true, nil
	}
	return false, can
}

func (r *UsuarioRepositoriePg) GetUsuarioModulos(usuarioID string) ([]types.Modulo, error) {
	rows, errRows := r.Pool.Query(
		context.Background(),
		consts.QUERY_PREPARE_GET_USUARIO_MODULOS,
		usuarioID,
	)
	if errRows != nil {
		return []types.Modulo{}, errRows
	}
	defer rows.Close()

	var modulos []types.Modulo
	for rows.Next() {
		var m types.Modulo
		if errScan := rows.Scan(&m.ID, &m.Nombre, &m.Tipo, &m.Descripcion); errScan != nil {
			continue
		}
		modulos = append(modulos, m)
	}

	if errRows := rows.Err(); errRows != nil {
		return []types.Modulo{}, errRows
	}

	return modulos, nil
}

func (r *UsuarioRepositoriePg) HasModulo(usuarioID string, moduloNombre string) (bool, error) {
	var count int
	err := r.Pool.QueryRow(
		context.Background(),
		consts.QUERY_PREPARE_HAS_MODULO,
		usuarioID,
		moduloNombre,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
