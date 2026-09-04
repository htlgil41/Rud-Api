package repositories

import (
	"context"
	"fmt"
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

func (r *ModuloReporteRepositorioPg) MarcarReporteNoAutorizado(reporteID string, detalle string) error {
	commandTag, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_MARCAR_REPORTE_NO_AUTORIZADO,
		reporteID,
		detalle,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("Reporte no encontrado")
	}
	return nil
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

func (r *ModuloReporteRepositorioPg) AsignarModuloReporte(usuarioID string, moduloReporteID string) error {
	commandTag, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_ASSIGN_MODULO_REPORTE,
		usuarioID,
		moduloReporteID,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("El modulo de reporte ya se encuentra asignado al usuario o no existe")
	}
	return nil
}

func (r *ModuloReporteRepositorioPg) EliminarModuloReporte(usuarioID string, moduloReporteID string) error {
	commandTag, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_DELETE_ASIGNACION_MODULO_REPORTE,
		usuarioID,
		moduloReporteID,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("El modulo de reporte no se encuentra asignado al usuario")
	}
	return nil
}

func (r *ModuloReporteRepositorioPg) CreateModuloReporte(mr types.ModuloReporte) error {
	_, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_CREATE_MODULO_REPORTE,
		mr.ID,
		mr.Nombre,
		mr.Descripcion,
		mr.ParamsSize,
		mr.QueryPlane,
		mr.QueryPrepare,
	)
	return err
}

func (r *ModuloReporteRepositorioPg) GetAllModuloReportes() ([]types.ModuloReporte, error) {
	rows, errRows := r.Pool.Query(
		context.Background(),
		consts.QUERY_PREPARE_GET_ALL_MODULO_REPORTES,
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

func (r *ModuloReporteRepositorioPg) CancelReporteGenerado(reporteID string, usuarioID string) error {
	commandTag, err := r.Pool.Exec(
		context.Background(),
		consts.QUERY_PREPARE_CANCEL_REPORTE_GENERADO,
		reporteID,
		usuarioID,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("Reporte no encontrado o no pertenece al usuario")
	}
	return nil
}

func (r *ModuloReporteRepositorioPg) GetReportesGeneradosByUsuario(usuarioID string, cursor *string) (types.ReportesGeneradosPaginados, error) {
	rows, errRows := r.Pool.Query(
		context.Background(),
		consts.QUERY_PREPARE_GET_REPORTES_GENERADOS_BY_USUARIO,
		usuarioID,
		cursor,
	)
	if errRows != nil {
		return types.ReportesGeneradosPaginados{}, errRows
	}
	defer rows.Close()

	var reportes []types.ReporteGenerado
	for rows.Next() {
		var rg types.ReporteGenerado
		if errScan := rows.Scan(&rg.ID, &rg.ModuloReporteID, &rg.UsuarioID, &rg.Estado, &rg.SolicitadoEn); errScan != nil {
			continue
		}
		reportes = append(reportes, rg)
	}

	if errRows := rows.Err(); errRows != nil {
		return types.ReportesGeneradosPaginados{}, errRows
	}

	result := types.ReportesGeneradosPaginados{
		Reportes: reportes,
		HasMore:  len(reportes) > 15,
	}

	if len(reportes) > 15 {
		result.Reportes = reportes[:15]
		result.NextCursor = reportes[15].ID
	}

	return result, nil
}

func (r *ModuloReporteRepositorioPg) GetAllReportesGenerados(cursor *string) (types.ReportesGeneradosPaginados, error) {
	rows, errRows := r.Pool.Query(
		context.Background(),
		consts.QUERY_PREPARE_GET_ALL_REPORTES_GENERADOS,
		cursor,
	)
	if errRows != nil {
		return types.ReportesGeneradosPaginados{}, errRows
	}
	defer rows.Close()

	var reportes []types.ReporteGenerado
	for rows.Next() {
		var rg types.ReporteGenerado
		if errScan := rows.Scan(&rg.ID, &rg.ModuloReporteID, &rg.UsuarioID, &rg.Estado, &rg.SolicitadoEn); errScan != nil {
			continue
		}
		reportes = append(reportes, rg)
	}

	if errRows := rows.Err(); errRows != nil {
		return types.ReportesGeneradosPaginados{}, errRows
	}

	result := types.ReportesGeneradosPaginados{
		Reportes: reportes,
		HasMore:  len(reportes) > 15,
	}

	if len(reportes) > 15 {
		result.Reportes = reportes[:15]
		result.NextCursor = reportes[15].ID
	}

	return result, nil
}
