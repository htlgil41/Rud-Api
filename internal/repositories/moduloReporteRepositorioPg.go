package repositories

import (
	"context"
	"rud-api/internal/consts"
	"rud-api/internal/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModuloReporteRepositorioPg struct {
	Pool *pgxpool.Pool
}

func (r *ModuloReporteRepositorioPg) GetUsuarioModuloReportes(usuarioID string) ([]types.ModuloReporte, error) {
	rows, errRows := r.Pool.Query(
		context.Background(),
		consts.QUERY_PREPARE_GET_USUARIO_MODULO_REPORTES,
		usuarioID,
	)
	if errRows != nil {
		return []types.ModuloReporte{}, errRows
	}
	defer rows.Close()

	var reportes []types.ModuloReporte
	for rows.Next() {
		var mr types.ModuloReporte
		if errScan := rows.Scan(&mr.ID, &mr.Nombre, &mr.Descripcion, &mr.ParamsSize); errScan != nil {
			continue
		}
		reportes = append(reportes, mr)
	}

	if errRows := rows.Err(); errRows != nil {
		return []types.ModuloReporte{}, errRows
	}

	return reportes, nil
}

func (r *ModuloReporteRepositorioPg) GetModuloReporteByID(reporteID string) (types.ModuloReporte, error) {
	var mr types.ModuloReporte
	err := r.Pool.QueryRow(
		context.Background(),
		consts.QUERY_PREPARE_GET_MODULO_REPORTE_BY_ID,
		reporteID,
	).Scan(&mr.ID, &mr.Nombre, &mr.Descripcion, &mr.ParamsSize)
	if err != nil {
		return types.ModuloReporte{}, err
	}
	return mr, nil
}

func (r *ModuloReporteRepositorioPg) CreateReporteGenerado(reporte types.ReporteGenerado) error {
	_, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_CREATE_REPORTE_GENERADO,
		reporte.ID,
		reporte.ModuloReporteID,
		reporte.UsuarioID,
	)
	return err
}

func (r *ModuloReporteRepositorioPg) HasModuloReporte(usuarioID string, moduloReporteID string) (bool, error) {
	var count int
	err := r.Pool.QueryRow(
		context.Background(),
		consts.QUERY_PREPARE_HAS_MODULO_REPORTE,
		usuarioID,
		moduloReporteID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
