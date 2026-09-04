package types

import "time"

type ReporteSolicitadoEvent struct {
	ReporteID       string    `json:"reporte_id"`
	UsuarioID       string    `json:"usuario_id"`
	ModuloReporteID string    `json:"modulo_reporte_id"`
	Parametros      []string  `json:"parametros"`
	SolicitadoEn    time.Time `json:"solicitado_en"`
}
