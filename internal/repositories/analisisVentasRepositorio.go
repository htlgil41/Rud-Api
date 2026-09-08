package repositories

import (
	"context"
	"database/sql"
	"rud-api/internal/consts"
	"rud-api/internal/helpers"
	"rud-api/internal/types"
	"time"
)

type AnalisisVentasRepositorie struct {
	Db *sql.DB
}

func (r *AnalisisVentasRepositorie) GetDepartamentosCodigos() ([]string, error) {
	var resutado []string = []string{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		consts.QUERY_DEPARTAMENTO,
	)
	if errRows != nil {
		return resutado, errRows
	}

	for rows.Next() {
		var v string
		if errScan := rows.Scan(&v); errScan != nil {
			continue
		}
		resutado = append(resutado, v)
	}

	if ErrNext := rows.Err(); errRows != nil {
		return resutado, ErrNext
	}

	return resutado, nil
}

func (r *AnalisisVentasRepositorie) GetGrupoCodigos() ([]string, error) {
	var resutado []string = []string{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		consts.QUERY_GRUPO,
	)
	if errRows != nil {
		return resutado, errRows
	}

	for rows.Next() {
		var v string
		if errScan := rows.Scan(&v); errScan != nil {
			continue
		}
		resutado = append(resutado, v)
	}

	if ErrNext := rows.Err(); errRows != nil {
		return resutado, ErrNext
	}

	return resutado, nil
}

func (r *AnalisisVentasRepositorie) GetSubGrupoCodigos() ([]string, error) {
	var resutado []string = []string{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		consts.QUERY_SUBGRUPO,
	)
	if errRows != nil {
		return resutado, errRows
	}

	for rows.Next() {
		var v string
		if errScan := rows.Scan(&v); errScan != nil {
			continue
		}
		resutado = append(resutado, v)
	}

	if ErrNext := rows.Err(); errRows != nil {
		return resutado, ErrNext
	}

	return resutado, nil
}

func (r *AnalisisVentasRepositorie) GetVentasDepartamento(
	query string,
	fecha_start string,
	fecha_end string,
	departamento_filters string,
) ([]types.VentasDepartamento, error) {
	var resultado []types.VentasDepartamento = []types.VentasDepartamento{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		helpers.TranformQuerysAddParametersStringsFlag(
			query,
			[]any{departamento_filters, departamento_filters},
		),
		fecha_start,
		fecha_end,
	)
	if errRows != nil {
		return resultado, errRows
	}
	for rows.Next() {
		var v types.VentasDepartamento
		rows.Scan(
			&v.Fecha,
			&v.Sucursal,
			&v.Departamento,
			&v.Diferencia,
			&v.Subtotal,
			&v.Cantidad,
			&v.Utilidad,
			&v.NCosto,
			&v.Precio,
			&v.Costo,
			&v.UtilidadPer,
			&v.Total,
			&v.CostoOferta,
		)
		resultado = append(resultado, v)
	}

	if eRows := rows.Err(); eRows != nil {
		return resultado, errRows
	}

	return resultado, nil
}

func (r *AnalisisVentasRepositorie) GetVentasGrupo(
	query string,
	fecha_start string,
	fecha_end string,
	departamento_filters string,
	grupo_filters string,
) ([]types.VentasGrupo, error) {
	var resultado []types.VentasGrupo = []types.VentasGrupo{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		helpers.TranformQuerysAddParametersStringsFlag(
			query,
			[]any{departamento_filters, grupo_filters, departamento_filters, grupo_filters},
		),
		fecha_start,
		fecha_end,
	)
	if errRows != nil {
		return resultado, errRows
	}
	for rows.Next() {
		var v types.VentasGrupo
		rows.Scan(
			&v.Fecha,
			&v.Sucursal,
			&v.Grupo,
			&v.Departamento,
			&v.Total,
			&v.Precio,
			&v.Subtotal,
			&v.NCantidad,
			&v.NCosto,
			&v.UtilidadPer,
			&v.Cantidad,
			&v.Utilidad,
			&v.CostoOferta,
		)
		resultado = append(resultado, v)
	}

	if eRows := rows.Err(); eRows != nil {
		return resultado, errRows
	}

	return resultado, nil
}

func (r *AnalisisVentasRepositorie) GetVentasSubGrupo(
	query string,
	fecha_start string,
	fecha_end string,
	departamento_filters string,
	grupo_filters string,
	subGrupo_filters string,
) ([]types.SubGrupo, error) {
	var resultado []types.SubGrupo = []types.SubGrupo{}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCtx()

	rows, errRows := r.Db.QueryContext(
		ctx,
		helpers.TranformQuerysAddParametersStringsFlag(
			query,
			[]any{
				departamento_filters, grupo_filters, subGrupo_filters,
				departamento_filters, grupo_filters, subGrupo_filters,
			},
		),
		fecha_start,
		fecha_end,
	)
	if errRows != nil {
		return resultado, errRows
	}
	for rows.Next() {
		var v types.SubGrupo
		rows.Scan(
			&v.Fecha,
			&v.Sucursal,
			&v.Departamento,
			&v.Grupo,
			&v.SubGrupo,
			&v.Total,
			&v.Ncatidad,
			&v.Cantidad,
			&v.Precio,
			&v.Subtotal,
			&v.Ncosto,
			&v.UtilidadPer,
			&v.Utilidad,
			&v.CostoOferta,
		)
		resultado = append(resultado, v)
	}

	if eRows := rows.Err(); eRows != nil {
		return resultado, errRows
	}

	return resultado, nil
}
