package types

import "time"

type UsuarioRespositorie struct {
	ID            string
	Usernme       string
	Email         string
	PasswordHash  string
	Nombre        string
	Departamento  string
	Activo        bool
	CreadoEn      *time.Time
	ActualizadoEn *time.Time
}
