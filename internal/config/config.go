// Package config loads geopilot settings from the environment.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	AWLURL      string        // base URL of the AWL controller, e.g. http://192.168.1.50
	AWLPasscode string        // AID passcode that unlocks getregs/putregs
	AWLAddr     int           // Modbus address of the ABC board
	Fast        time.Duration // poll cadence per tier
	Medium      time.Duration
	Slow        time.Duration
	BatchSize   int // max registers per getregs call

	WeatherProvider string        // "open-meteo" (default, public) or "westmoreland" (/api/latest)
	WeatherURL      string        // base URL override; empty = provider's own default
	WeatherLat      float64       // location for the open-meteo provider
	WeatherLon      float64       //
	WeatherPoll     time.Duration // how often to pull weather series

	BayouAPIKey     string        // Bayou Energy API key (bills); empty disables
	BayouCustomerID int           // Bayou customer id to pull bills for
	BayouPoll       time.Duration // how often to refresh bills
}

func Load() Config {
	return Config{
		DatabaseURL: env("DATABASE_URL", "postgres://geopilot:geopilot@localhost:5432/geopilot?sslmode=disable"),
		AWLURL:      env("AWL_URL", "http://192.168.1.50"),
		AWLPasscode: env("AWL_PASSCODE", "9999"),
		AWLAddr:     envInt("AWL_ADDR", 1),
		Fast:        envDur("POLL_FAST", 5*time.Second),
		Medium:      envDur("POLL_MEDIUM", 60*time.Second),
		Slow:        envDur("POLL_SLOW", 3600*time.Second),
		// 16 keeps the getregs URL under the controller's ~256-byte URI limit even
		// with 5-digit IZ2 register numbers (larger batches return HTTP 414).
		BatchSize:       envInt("BATCH_SIZE", 16),
		// Default to the public original source (Open-Meteo) so a fresh deployment
		// works for anyone; point WEATHER_PROVIDER=westmoreland + WEATHER_URL at a
		// private /api/latest instance to use that instead. Lat/Lon default to
		// Westmoreland, NY — change them for your own location.
		WeatherProvider: env("WEATHER_PROVIDER", "open-meteo"),
		WeatherURL:      env("WEATHER_URL", ""),
		WeatherLat:      envFloat("WEATHER_LAT", 43.104),
		WeatherLon:      envFloat("WEATHER_LON", -75.446),
		WeatherPoll:     envDur("POLL_WEATHER", 10*time.Minute),
		BayouAPIKey:     env("BAYOU_API_KEY", ""),
		BayouCustomerID: envInt("BAYOU_CUSTOMER_ID", 0),
		BayouPoll:       envDur("POLL_BAYOU", 12*time.Hour),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envDur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
