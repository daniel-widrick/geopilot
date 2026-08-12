-- geopilot schema — source-agnostic timeseries backbone on TimescaleDB.
-- A "series" is any single measured signal: a furnace Modbus register today,
-- a water-heater probe / power price / weather metric later. Readings store the
-- RAW integer value; scale + signedness live on the series and are applied at
-- read time, so storage stays lossless and compresses hard.

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS series (
  id      integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source  text    NOT NULL,                 -- 'furnace', 'waterheater', 'power', 'weather', ...
  addr    integer,                          -- furnace: Modbus device address
  reg     integer,                          -- furnace: register number
  key     text    NOT NULL UNIQUE,          -- stable id, e.g. 'furnace:1:1117'
  name    text,
  unit    text,
  scale   double precision NOT NULL DEFAULT 1,   -- display value = raw * scale
  signed  boolean NOT NULL DEFAULT false,        -- interpret raw as signed 16-bit
  tier    text    NOT NULL DEFAULT 'slow',        -- poll cadence: fast|medium|slow
  enabled boolean NOT NULL DEFAULT true,
  notes   text
);

CREATE TABLE IF NOT EXISTS readings (
  time      timestamptz NOT NULL,
  series_id integer     NOT NULL REFERENCES series(id),
  value     bigint      NOT NULL
);

SELECT create_hypertable('readings', 'time',
  chunk_time_interval => interval '1 day', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS readings_series_time ON readings (series_id, time DESC);

-- Columnar compression: group each chunk by series_id so runs of one register's
-- values sit together and delta-compress. 30 days stay hot (uncompressed).
ALTER TABLE readings SET (
  timescaledb.compress,
  timescaledb.compress_segmentby = 'series_id',
  timescaledb.compress_orderby   = 'time DESC'
);
SELECT add_compression_policy('readings', interval '30 days', if_not_exists => TRUE);

-- Keep 5 years of raw readings.
SELECT add_retention_policy('readings', interval '5 years', if_not_exists => TRUE);

-- Rollups for fast long-range queries.
CREATE MATERIALIZED VIEW IF NOT EXISTS readings_1m
WITH (timescaledb.continuous) AS
SELECT series_id,
       time_bucket('1 minute', time) AS bucket,
       avg(value)::double precision   AS avg,
       min(value)                     AS min,
       max(value)                     AS max,
       last(value, time)              AS last,
       count(*)                       AS n
FROM readings
GROUP BY series_id, bucket
WITH NO DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS readings_1h
WITH (timescaledb.continuous) AS
SELECT series_id,
       time_bucket('1 hour', time) AS bucket,
       avg(value)::double precision AS avg,
       min(value)                   AS min,
       max(value)                   AS max,
       last(value, time)            AS last,
       count(*)                     AS n
FROM readings
GROUP BY series_id, bucket
WITH NO DATA;

SELECT add_continuous_aggregate_policy('readings_1m',
  start_offset => interval '3 hours', end_offset => interval '1 minute',
  schedule_interval => interval '1 minute', if_not_exists => TRUE);
SELECT add_continuous_aggregate_policy('readings_1h',
  start_offset => interval '3 days', end_offset => interval '1 hour',
  schedule_interval => interval '10 minutes', if_not_exists => TRUE);

-- Convenience: latest decoded value per series.
CREATE OR REPLACE VIEW latest AS
SELECT s.key, s.name, s.unit, s.tier, s.source, r.time, r.value AS raw,
       (CASE WHEN s.signed AND r.value >= 32768 THEN r.value - 65536 ELSE r.value END)
         * s.scale AS value
FROM series s
JOIN LATERAL (
  SELECT time, value FROM readings
  WHERE series_id = s.id ORDER BY time DESC LIMIT 1
) r ON true;
