package types

type VentasDepartamento struct {
	Fecha        string
	Sucursal     string
	Departamento string
	Diferencia   float64
	Subtotal     float64
	Cantidad     float64
	Utilidad     float64
	NCosto       float64
	Precio       float64
	Costo        float64
	UtilidadPer  float64
	Total        float64
	CostoOferta  float64
}

type VentasGrupo struct {
	Fecha        string
	Sucursal     string
	Grupo        string
	Departamento string
	Total        float64
	Precio       float64
	Subtotal     float64
	NCantidad    float64
	NCosto       float64
	UtilidadPer  float64
	Cantidad     float64
	Utilidad     float64
	CostoOferta  float64
}

type SubGrupo struct {
	Sucursal     string
	Fecha        string
	Departamento string
	Grupo        string
	SubGrupo     string
	Total        float64
	Ncatidad     float64
	Cantidad     float64
	Precio       float64
	Subtotal     float64
	Ncosto       float64
	UtilidadPer  float64
	Utilidad     float64
	CostoOferta  float64
}
