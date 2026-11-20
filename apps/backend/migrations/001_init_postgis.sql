-- 001_init_postgis.sql
-- Inicialização da extensão espacial PostGIS e criação das tabelas com índices GIST.

CREATE EXTENSION IF NOT EXISTS postgis;

-- 1. Segmentos de Vias Públicas (OpenStreetMap)
CREATE TABLE IF NOT EXISTS road_segments (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    estimated_spots INT NOT NULL DEFAULT 0 CHECK (estimated_spots >= 0),
    parking_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    geom GEOMETRY(Geometry, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_road_segments_geom
    ON road_segments USING GIST (geom);

-- 2. Estacionamentos Privados Comerciais
CREATE TABLE IF NOT EXISTS private_parkings (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    capacity INT NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    is_open BOOLEAN NOT NULL DEFAULT TRUE,
    geom GEOMETRY(Point, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_private_parkings_geom
    ON private_parkings USING GIST (geom);

-- 3. Eventos Regionais que Impactam a Demanda
CREATE TABLE IF NOT EXISTS events (
    id VARCHAR(100) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    expected_attendance INT NOT NULL DEFAULT 0 CHECK (expected_attendance >= 0),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    geom GEOMETRY(Point, 4326) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_geom
    ON events USING GIST (geom);

CREATE INDEX IF NOT EXISTS idx_events_time_window
    ON events (start_time, end_time);
