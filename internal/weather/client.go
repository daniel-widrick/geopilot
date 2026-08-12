// Package weather pulls current conditions from a configurable source. Two
// providers implement the same contract:
//
//   - "open-meteo"   the public original source (api.open-meteo.com); the default,
//     so a fresh deployment works for anyone with just a lat/lon.
//   - "westmoreland" a private timeseries proxy exposing /api/latest?key=… ->
//     {"t":<unix>,"v":<number>} (the same Open-Meteo variables, re-served).
//
// The series keys are Open-Meteo variable names suffixed with a location id
// (e.g. weather:temperature_2m@1), so both providers speak the same vocabulary.
package weather

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// userAgent identifies geopilot to the upstream API so its traffic can be
// filtered out of the source's own metrics.
const userAgent = "geopilot/0.1 (geothermal-monitor; +https://github.com/daniel-widrick/geopilot)"

// Provider fetches current values for a set of series keys. Keys that the source
// has no value for are simply omitted from the returned map (not an error).
type Provider interface {
	Fetch(ctx context.Context, keys []string) (map[string]float64, error)
}

// Options selects and configures a provider.
type Options struct {
	Provider string  // "open-meteo" (default) or "westmoreland"
	BaseURL  string  // override; empty uses the provider's own default
	Lat, Lon float64 // location, for open-meteo
}

// New builds the configured weather provider.
func New(o Options) (Provider, error) {
	hc := &http.Client{Timeout: 15 * time.Second}
	switch strings.ToLower(strings.TrimSpace(o.Provider)) {
	case "", "open-meteo", "openmeteo", "open_meteo":
		base := o.BaseURL
		if base == "" {
			base = "https://api.open-meteo.com"
		}
		if o.Lat == 0 && o.Lon == 0 {
			return nil, fmt.Errorf("open-meteo provider needs WEATHER_LAT/WEATHER_LON")
		}
		return &openMeteo{base: strings.TrimRight(base, "/"), lat: o.Lat, lon: o.Lon, http: hc}, nil
	case "westmoreland", "latest":
		base := o.BaseURL
		if base == "" {
			base = "https://westmoreland.app"
		}
		return &latestAPI{base: strings.TrimRight(base, "/"), http: hc}, nil
	default:
		return nil, fmt.Errorf("unknown weather provider %q (want open-meteo or westmoreland)", o.Provider)
	}
}

// baseVar strips the "weather:" prefix and the "@<loc>" suffix from a series key,
// leaving the bare Open-Meteo variable name (e.g. "temperature_2m").
func baseVar(key string) string {
	v := strings.TrimPrefix(key, "weather:")
	if i := strings.IndexByte(v, '@'); i >= 0 {
		v = v[:i]
	}
	return v
}
