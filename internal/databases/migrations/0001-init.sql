CREATE TABLE usuarios (
    id VARCHAR(255) PRIMARY KEY NOT NULL UNIQUE,
    usernme VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE modulos (
    id VARCHAR(255) PRIMARY KEY NOT NULL UNIQUE,
    nombre VARCHAR(100) NOT NULL UNIQUE,
    tipo VARCHAR(50) NOT NULL,
    descripcion TEXT,
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE usuario_modulos (
    usuario_id VARCHAR(255) NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    modulo_id VARCHAR(255) NOT NULL REFERENCES modulos(id) ON DELETE CASCADE,
    asignado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (usuario_id, modulo_id)
);

CREATE TABLE modulos_reportes (
    id VARCHAR(255) PRIMARY KEY NOT NULL UNIQUE,
    nombre VARCHAR(150) NOT NULL,
    descripcion TEXT,
    query_plane TEXT NOT NULL,
    query_prepare TEXT NOT NULL,
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE usuario_modulos_reportes (
    usuario_id VARCHAR(255) NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    modulo_reporte_id VARCHAR(255) NOT NULL REFERENCES modulos_reportes(id) ON DELETE CASCADE,
    asignado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (usuario_id, modulo_reporte_id)
);

CREATE TABLE reportes_generados (
    id VARCHAR(255) PRIMARY KEY NOT NULL UNIQUE,
    modulo_reporte_id VARCHAR(255) NOT NULL REFERENCES modulos_reportes(id) ON DELETE SET NULL,
    usuario_id VARCHAR(255) NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    estado VARCHAR(100) NOT NULL DEFAULT 'PENDIENTE',
    detalle_construccion TEXT,
    parametros TEXT DEFAULT '',
    nombre_archivo VARCHAR(255),
    ruta_archivo TEXT,
    
    solicitado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    iniciado_en TIMESTAMPTZ,
    completado_en TIMESTAMPTZ
);

-- =========================================================================
-- ÍNDICES
-- =========================================================================

CREATE UNIQUE INDEX idx_usuarios_email_lower ON usuarios (LOWER(email));
CREATE INDEX idx_usuarios_activo ON usuarios (activo) WHERE activo = TRUE;

CREATE INDEX idx_modulos_tipo ON modulos (tipo);
CREATE INDEX idx_modulos_activo ON modulos (activo) WHERE activo = TRUE;
CREATE INDEX idx_usuario_modulos_modulo_id ON usuario_modulos (modulo_id);

CREATE INDEX idx_modulos_reportes_nombre ON modulos_reportes (nombre);
CREATE INDEX idx_modulos_reportes_activo ON modulos_reportes (activo) WHERE activo = TRUE;
CREATE INDEX idx_usuario_modulos_reportes_reporte_id ON usuario_modulos_reportes (modulo_reporte_id);

CREATE INDEX idx_reportes_generados_usuario_fecha 
ON reportes_generados (usuario_id, solicitado_en DESC);


CREATE INDEX idx_reportes_generados_modulo_reporte_id ON reportes_generados (modulo_reporte_id);
CREATE INDEX idx_reportes_generados_en_progreso ON reportes_generados (estado);