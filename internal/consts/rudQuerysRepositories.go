package consts

var INIT_DB_MIGRATION_RUD_SQLITE = `
CREATE TABLE IF NOT EXISTS ventas_departamento (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fecha DATETIME NOT NULL,
    sucursal TEXT NOT NULL,
    departamento TEXT NOT NULL,
    diferencia REAL NOT NULL,
    subtotal REAL NOT NULL,
    cantidad REAL NOT NULL,
    utilidad REAL NOT NULL,
    ncosto REAL NOT NULL,
    precio REAL NOT NULL,
    costo REAL NOT NULL,
    utilidad_per REAL NOT NULL,
    total REAL NOT NULL,
    costo_oferta REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS ventas_grupo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fecha DATETIME NOT NULL,
    sucursal TEXT NOT NULL,
    grupo TEXT NOT NULL,
    departamento TEXT NOT NULL,
    total REAL NOT NULL,
    precio REAL NOT NULL,
    subtotal REAL NOT NULL,
    ncantidad REAL NOT NULL,
    ncosto REAL NOT NULL,
    utilidad_per REAL NOT NULL,
    cantidad REAL NOT NULL,
    utilidad REAL NOT NULL,
    costo_oferta REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS sub_grupo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sucursal TEXT NOT NULL,
    fecha DATETIME NOT NULL,
    departamento TEXT NOT NULL,
    grupo TEXT NOT NULL,
    sub_grupo TEXT NOT NULL,
    total REAL NOT NULL,
    ncatidad REAL NOT NULL,
    cantidad REAL NOT NULL,
    precio REAL NOT NULL,
    subtotal REAL NOT NULL,
    ncosto REAL NOT NULL,
    utilidad_per REAL NOT NULL,
    utilidad REAL NOT NULL,
    costo_oferta REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ventas_dept_fecha 
    ON ventas_departamento (fecha);

CREATE INDEX IF NOT EXISTS idx_ventas_dept_sucursal_dept 
    ON ventas_departamento (sucursal, departamento);

CREATE INDEX IF NOT EXISTS idx_ventas_dept_busqueda 
    ON ventas_departamento (fecha, sucursal, departamento);

CREATE INDEX IF NOT EXISTS idx_ventas_grupo_fecha 
    ON ventas_grupo (fecha);

CREATE INDEX IF NOT EXISTS idx_ventas_grupo_busqueda 
    ON ventas_grupo (fecha, sucursal, departamento, grupo);

CREATE INDEX IF NOT EXISTS idx_sub_grupo_fecha 
    ON sub_grupo (fecha);

CREATE INDEX IF NOT EXISTS idx_sub_grupo_busqueda 
    ON sub_grupo (fecha, sucursal, departamento, grupo, sub_grupo);
`

var ROLLBACK_DATABASE_RUD = `
    DELETE FROM ventas_departamento;
    DELETE FROM ventas_grupo;
    DELETE FROM sub_grupo;
`
