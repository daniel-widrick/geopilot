// Package store persists readings into TimescaleDB.
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Series struct {
	ID     int
	Source string
	Addr   int
	Reg    int
	Key    string
	Name   string
	Unit   string
	Scale  float64
	Signed bool
	Tier   string
}

type Reading struct {
	Time     time.Time
	SeriesID int
	Value    int64
}

type Store struct {
	pool *pgxpool.Pool
}

// Open connects, retrying until the database is reachable (Compose start order).
func Open(ctx context.Context, url string) (*Store, error) {
	var pool *pgxpool.Pool
	var err error
	for attempt := 0; attempt < 30; attempt++ {
		pool, err = pgxpool.New(ctx, url)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				return &Store{pool: pool}, nil
			}
			pool.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, err
}

func (s *Store) Close() { s.pool.Close() }

// LoadSeries returns every enabled series (the poll plan lives in the DB); the
// collector fans them out by source.
func (s *Store) LoadSeries(ctx context.Context) ([]Series, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, source, COALESCE(addr,0), COALESCE(reg,0), key,
		       COALESCE(name,''), COALESCE(unit,''), scale, signed, tier
		FROM series
		WHERE enabled
		ORDER BY source, reg`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Series
	for rows.Next() {
		var s Series
		if err := rows.Scan(&s.ID, &s.Source, &s.Addr, &s.Reg, &s.Key,
			&s.Name, &s.Unit, &s.Scale, &s.Signed, &s.Tier); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type Point struct {
	Time  time.Time `json:"t"`
	Value float64   `json:"v"`
}

// History returns 1-minute rollup points (decoded via the series scale/sign) for
// a series key over the last N hours, oldest first.
func (s *Store) History(ctx context.Context, key string, hours int) ([]Point, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.bucket,
		       (CASE WHEN se.signed AND a.avg >= 32768 THEN a.avg - 65536 ELSE a.avg END) * se.scale
		FROM readings_1m a
		JOIN series se ON se.id = a.series_id
		WHERE se.key = $1 AND a.bucket > now() - ($2 || ' hours')::interval
		ORDER BY a.bucket`, key, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Time, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type BillRow struct {
	ID               int64
	CustomerID       int
	AccountNumber    string
	BilledOn         *time.Time
	PeriodFrom       *time.Time
	PeriodTo         *time.Time
	KWh              float64
	ElectricityCents int64
	DeliveryCents    int64
	SupplyCents      int64
	TotalCents       int64
	FileURL          string
}

// UpsertBills inserts/updates bills by id.
func (s *Store) UpsertBills(ctx context.Context, rows []BillRow) error {
	for _, b := range rows {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO bills (id, customer_id, account_number, billed_on, period_from, period_to,
			                   kwh, electricity_cents, delivery_cents, supply_cents, total_cents, file_url, fetched_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now())
			ON CONFLICT (id) DO UPDATE SET
			  kwh=EXCLUDED.kwh, electricity_cents=EXCLUDED.electricity_cents,
			  delivery_cents=EXCLUDED.delivery_cents, supply_cents=EXCLUDED.supply_cents,
			  total_cents=EXCLUDED.total_cents, file_url=EXCLUDED.file_url, fetched_at=now()`,
			b.ID, b.CustomerID, b.AccountNumber, b.BilledOn, b.PeriodFrom, b.PeriodTo,
			b.KWh, b.ElectricityCents, b.DeliveryCents, b.SupplyCents, b.TotalCents, b.FileURL)
		if err != nil {
			return err
		}
	}
	return nil
}

// CurrentRate returns the most recent bill's effective electric $/kWh (0 if none).
func (s *Store) CurrentRate(ctx context.Context) (float64, error) {
	var r float64
	err := s.pool.QueryRow(ctx, `SELECT electric_usd_per_kwh FROM bill_rates LIMIT 1`).Scan(&r)
	if err != nil {
		return 0, nil // no bills yet is not an error
	}
	return r, nil
}

// CostToday returns the geo's spend so far today (local time), from the total-power
// series (reg 1153 = total watts) integrated over the monitored window × rate.
func (s *Store) CostToday(ctx context.Context, rate float64) (float64, bool, error) {
	if rate <= 0 {
		return 0, false, nil
	}
	var avgW, hours float64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(avg(r.value), 0),
		       COALESCE(EXTRACT(EPOCH FROM (max(r.time) - min(r.time))) / 3600.0, 0)
		FROM readings r JOIN series se ON se.id = r.series_id
		WHERE se.key = 'furnace:1:1153'
		  AND r.time >= date_trunc('day', now() AT TIME ZONE 'America/New_York') AT TIME ZONE 'America/New_York'`).
		Scan(&avgW, &hours)
	if err != nil || hours <= 0 {
		return 0, false, err
	}
	return avgW * hours / 1000.0 * rate, true, nil
}

// DisableSeries marks series as not-enabled so they are skipped on future loads
// (used to permanently retire registers the board reports as invalid).
func (s *Store) DisableSeries(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE series SET enabled = false WHERE id = ANY($1)`, ids)
	return err
}

// Insert writes a batch of readings efficiently via COPY.
func (s *Store) Insert(ctx context.Context, rs []Reading) error {
	if len(rs) == 0 {
		return nil
	}
	_, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"readings"},
		[]string{"time", "series_id", "value"},
		pgx.CopyFromSlice(len(rs), func(i int) ([]any, error) {
			return []any{rs[i].Time, rs[i].SeriesID, rs[i].Value}, nil
		}),
	)
	return err
}
