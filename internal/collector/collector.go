// Package collector polls the furnace on tiered intervals and writes every
// reading to both the database (history) and the in-memory live snapshot (the
// dashboard's real-time source).
package collector

import (
	"context"
	"log"
	"math"
	"strings"
	"time"

	"github.com/daniel-widrick/geopilot/internal/awl"
	"github.com/daniel-widrick/geopilot/internal/bayou"
	"github.com/daniel-widrick/geopilot/internal/config"
	"github.com/daniel-widrick/geopilot/internal/live"
	"github.com/daniel-widrick/geopilot/internal/store"
	"github.com/daniel-widrick/geopilot/internal/weather"
)

// weatherScale must match migrations/003_seed_weather.sql (values stored ×100).
const weatherScale = 0.01

type Collector struct {
	cfg     config.Config
	client  *awl.Client
	weather weather.Provider
	bayou   *bayou.Client
	db      *store.Store
	snap    *live.Snapshot

	tiers    map[string][]store.Series // furnace: tier -> series
	weathers []store.Series            // weather series
}

func New(cfg config.Config, client *awl.Client, wx weather.Provider, by *bayou.Client, db *store.Store, snap *live.Snapshot, series []store.Series) *Collector {
	c := &Collector{cfg: cfg, client: client, weather: wx, bayou: by, db: db, snap: snap,
		tiers: map[string][]store.Series{}}
	for _, s := range series {
		switch s.Source {
		case "weather":
			c.weathers = append(c.weathers, s)
		case "furnace":
			c.tiers[s.Tier] = append(c.tiers[s.Tier], s)
		}
	}
	return c
}

// Run probes for invalid registers, then starts one goroutine per tier and
// blocks until ctx is cancelled.
func (c *Collector) Run(ctx context.Context) {
	if err := c.client.Auth(ctx); err != nil {
		log.Printf("collector: initial auth failed (will retry on demand): %v", err)
	}
	c.probe(ctx)
	go c.runTier(ctx, "fast", c.cfg.Fast)
	go c.runTier(ctx, "medium", c.cfg.Medium)
	go c.runTier(ctx, "slow", c.cfg.Slow)
	go c.runWeather(ctx)
	go c.runBayou(ctx)
	<-ctx.Done()
}

// runBayou refreshes utility bills from Bayou on a slow cadence (they change ~monthly).
func (c *Collector) runBayou(ctx context.Context) {
	if c.bayou == nil || c.cfg.BayouCustomerID == 0 {
		return
	}
	log.Printf("collector: bayou customer %d every %s", c.cfg.BayouCustomerID, c.cfg.BayouPoll)
	c.pollBayou(ctx)
	t := time.NewTicker(c.cfg.BayouPoll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.pollBayou(ctx)
		}
	}
}

func (c *Collector) pollBayou(ctx context.Context) {
	bills, err := c.bayou.GetBills(ctx, c.cfg.BayouCustomerID)
	if err != nil {
		log.Printf("collector: bayou bills failed: %v", err)
		return
	}
	rows := make([]store.BillRow, 0, len(bills))
	for _, b := range bills {
		rows = append(rows, store.BillRow{
			ID: b.ID, CustomerID: c.cfg.BayouCustomerID, AccountNumber: b.AccountNumber,
			BilledOn: parseDate(b.BilledOn), PeriodFrom: parseDate(b.BillingPeriodFrom), PeriodTo: parseDate(b.BillingPeriodTo),
			KWh: b.KWh(), ElectricityCents: b.ElectricityAmount, DeliveryCents: b.DeliveryCharge,
			SupplyCents: b.SupplyCharge, TotalCents: b.TotalAmount, FileURL: b.FileURL,
		})
	}
	if err := c.db.UpsertBills(ctx, rows); err != nil {
		log.Printf("collector: bayou upsert failed (%d bills): %v", len(rows), err)
		return
	}
	if r, _ := c.db.CurrentRate(ctx); r > 0 {
		log.Printf("collector: bayou %d bills; current rate $%.4f/kWh", len(rows), r)
	}
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// runWeather pulls the weather series from the external API on its own cadence.
func (c *Collector) runWeather(ctx context.Context) {
	if c.weather == nil || len(c.weathers) == 0 {
		return
	}
	log.Printf("collector: weather %d series every %s", len(c.weathers), c.cfg.WeatherPoll)
	c.pollWeather(ctx)
	t := time.NewTicker(c.cfg.WeatherPoll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.pollWeather(ctx)
		}
	}
}

func (c *Collector) pollWeather(ctx context.Context) {
	now := time.Now().UTC()
	keys := make([]string, len(c.weathers))
	byKey := make(map[string]store.Series, len(c.weathers))
	for i, s := range c.weathers {
		keys[i] = s.Key
		byKey[s.Key] = s
	}
	vals, err := c.weather.Fetch(ctx, keys)
	if err != nil {
		log.Printf("collector: weather fetch failed: %v", err)
		// fall through: a partial map may still have arrived before the error
	}
	var readings []store.Reading
	for key, v := range vals {
		s := byKey[key]
		raw := int64(math.Round(v / weatherScale))
		c.snap.Set(s.Key, raw, now)
		readings = append(readings, store.Reading{Time: now, SeriesID: s.ID, Value: raw})
	}
	if err := c.db.Insert(ctx, readings); err != nil {
		log.Printf("collector: weather insert failed (%d rows): %v", len(readings), err)
	}
}

// isBadAddr reports whether an error means "this register does not exist on the
// board" (a Modbus exception / invalid request) rather than a transient failure.
func isBadAddr(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	return strings.Contains(m, "Exception") || strings.Contains(m, "Invalid Request")
}

// probe reads every series once; any register the board rejects as invalid is
// retired (disabled in the DB and dropped from the poll plan) so it never again
// fails a batch and takes good neighbours down with it. Runs once at startup;
// retired registers stay disabled across restarts.
func (c *Collector) probe(ctx context.Context) {
	var all []store.Series
	for _, ss := range c.tiers {
		all = append(all, ss...)
	}
	var dead []int
	deadSet := map[int]bool{}
	for _, batch := range chunkSeries(all, c.cfg.BatchSize) {
		if ctx.Err() != nil {
			return
		}
		regs := make([]int, len(batch))
		for i, s := range batch {
			regs[i] = s.Reg
		}
		if _, err := c.client.GetRegs(ctx, regs); err == nil {
			continue // whole batch is valid
		} else if !isBadAddr(err) {
			continue // transient; leave enabled and let normal polling handle it
		}
		// a bad address hid in this batch — find the culprit(s) individually
		for _, s := range batch {
			if _, err := c.client.GetRegs(ctx, []int{s.Reg}); err != nil && isBadAddr(err) {
				dead = append(dead, s.ID)
				deadSet[s.ID] = true
			}
		}
	}
	if len(dead) == 0 {
		return
	}
	if err := c.db.DisableSeries(ctx, dead); err != nil {
		log.Printf("collector: probe: failed to retire %d registers: %v", len(dead), err)
	}
	for tier, ss := range c.tiers {
		kept := ss[:0]
		for _, s := range ss {
			if !deadSet[s.ID] {
				kept = append(kept, s)
			}
		}
		c.tiers[tier] = kept
	}
	log.Printf("collector: probe retired %d invalid registers", len(dead))
}

func chunkSeries(ss []store.Series, size int) [][]store.Series {
	if size <= 0 {
		size = 16
	}
	var out [][]store.Series
	for i := 0; i < len(ss); i += size {
		end := i + size
		if end > len(ss) {
			end = len(ss)
		}
		out = append(out, ss[i:end])
	}
	return out
}

func (c *Collector) runTier(ctx context.Context, tier string, interval time.Duration) {
	series := c.tiers[tier]
	if len(series) == 0 {
		return
	}
	log.Printf("collector: tier %-6s %d series every %s", tier, len(series), interval)
	// prime immediately, then on the interval
	c.poll(ctx, tier, series)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.poll(ctx, tier, series)
		}
	}
}

func (c *Collector) poll(ctx context.Context, tier string, series []store.Series) {
	now := time.Now().UTC()
	byReg := make(map[int]store.Series, len(series))
	regs := make([]int, 0, len(series))
	for _, s := range series {
		byReg[s.Reg] = s
		regs = append(regs, s.Reg)
	}

	var readings []store.Reading
	for _, chunk := range chunkRegs(regs, c.cfg.BatchSize) {
		vals, err := c.client.GetRegs(ctx, chunk)
		if err != nil {
			log.Printf("collector: tier %s read failed (%d regs): %v", tier, len(chunk), err)
			continue
		}
		for reg, v := range vals {
			s := byReg[reg]
			c.snap.Set(s.Key, v, now)
			readings = append(readings, store.Reading{Time: now, SeriesID: s.ID, Value: v})
		}
	}
	if err := c.db.Insert(ctx, readings); err != nil {
		log.Printf("collector: tier %s insert failed (%d rows): %v", tier, len(readings), err)
	}
}

func chunkRegs(regs []int, size int) [][]int {
	if size <= 0 {
		size = 40
	}
	var out [][]int
	for i := 0; i < len(regs); i += size {
		end := i + size
		if end > len(regs) {
			end = len(regs)
		}
		out = append(out, regs[i:end])
	}
	return out
}
