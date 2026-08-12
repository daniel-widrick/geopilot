// Command collector runs the geopilot furnace poller and dashboard server.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daniel-widrick/geopilot/internal/awl"
	"github.com/daniel-widrick/geopilot/internal/bayou"
	"github.com/daniel-widrick/geopilot/internal/collector"
	"github.com/daniel-widrick/geopilot/internal/config"
	"github.com/daniel-widrick/geopilot/internal/live"
	"github.com/daniel-widrick/geopilot/internal/store"
	"github.com/daniel-widrick/geopilot/internal/weather"
	"github.com/daniel-widrick/geopilot/internal/web"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	series, err := db.LoadSeries(ctx)
	if err != nil {
		log.Fatalf("load series: %v", err)
	}
	log.Printf("loaded %d furnace series", len(series))

	snap := live.New()
	client := awl.New(cfg.AWLURL, cfg.AWLPasscode, cfg.AWLAddr)
	var wx weather.Provider
	if w, err := weather.New(weather.Options{
		Provider: cfg.WeatherProvider, BaseURL: cfg.WeatherURL,
		Lat: cfg.WeatherLat, Lon: cfg.WeatherLon,
	}); err != nil {
		log.Printf("weather disabled: %v", err)
	} else {
		wx = w
		log.Printf("weather: provider %q", cfg.WeatherProvider)
	}
	var by *bayou.Client
	if cfg.BayouAPIKey != "" {
		by = bayou.New(cfg.BayouAPIKey)
	}

	col := collector.New(cfg, client, wx, by, db, snap, series)
	go col.Run(ctx)

	srv := web.New(snap, db, 2*time.Second)
	addr := os.Getenv("WEB_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	go func() {
		if err := srv.ListenAndServe(ctx, addr); err != nil && ctx.Err() == nil {
			log.Fatalf("web: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
}
