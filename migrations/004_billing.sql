-- Structured utility bills from Bayou Energy. Bills are records (many fields per
-- bill), not a single timeseries value, so they live in their own table rather
-- than readings. The effective $/kWh derived here feeds the real-time cost model.

CREATE TABLE IF NOT EXISTS bills (
  id                bigint PRIMARY KEY,   -- Bayou bill id
  customer_id       integer,
  account_number    text,
  billed_on         date,
  period_from       date,
  period_to         date,
  kwh               double precision,     -- electricity_consumption / 1000
  electricity_cents integer,              -- electric charges only
  delivery_cents    integer,
  supply_cents      integer,
  total_cents       integer,              -- electric + gas
  file_url          text,
  fetched_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS bills_period ON bills (period_to DESC);

-- Per-bill effective electric rate; only bills with real usage.
CREATE OR REPLACE VIEW bill_rates AS
SELECT id, period_from, period_to, kwh,
       round((electricity_cents / 100.0) / NULLIF(kwh, 0)::numeric, 5) AS electric_usd_per_kwh,
       round((total_cents / 100.0)       / NULLIF(kwh, 0)::numeric, 5) AS allin_usd_per_kwh
FROM bills
WHERE kwh > 0
ORDER BY period_to DESC;
