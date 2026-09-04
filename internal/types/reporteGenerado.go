package types

import "time"

type ReporteGenerado struct {
	ID                  string
	ModuloReporteID     string
	UsuarioID           string
	Estado              string
	DetalleConstruccion string
	NombreArchivo       string
	RutaArchivo         string
	SolicitadoEn        *time.Time
	IniciadoEn          *time.Time
	CompletadoEn        *time.Time
}

type ReportesGeneradosPaginados struct {
	Reportes   []ReporteGenerado
	NextCursor string
	HasMore    bool
}
