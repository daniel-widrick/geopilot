package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// openMeteo reads current conditions from the public Open-Meteo forecast API in a
// single request, mapping each requested series key to its Open-Meteo "current"
// variable. Units are requested imperial to match our stored schema; pressure is
// the one exception (Open-Meteo only offers hPa) and is converted to inHg here.
type openMeteo struct {
	base     string
	lat, lon float64
	http     *http.Client
}

// hPaToInHg converts millibars/hectopascals to inches of mercury.
const hPaToInHg = 0.0295299830714

type omResponse struct {
	Current map[string]json.RawMessage `json:"current"`
}

func (c *openMeteo) Fetch(ctx context.Context, keys []string) (map[string]float64, error) {
	// collect the distinct Open-Meteo variables behind the requested keys
	varSet := map[string]bool{}
	for _, k := range keys {
		varSet[baseVar(k)] = true
	}
	vars := make([]string, 0, len(varSet))
	for v := range varSet {
		vars = append(vars, v)
	}
	sort.Strings(vars) // stable request URL (nice for caching/debugging)

	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(c.lat, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(c.lon, 'f', -1, 64))
	q.Set("current", strings.Join(vars, ","))
	q.Set("temperature_unit", "fahrenheit")
	q.Set("wind_speed_unit", "mph")
	q.Set("precipitation_unit", "inch")
	q.Set("timezone", "UTC")
	u := c.base + "/v1/forecast?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo: http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var r omResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}

	out := make(map[string]float64, len(keys))
	for _, k := range keys {
		v := baseVar(k)
		raw, ok := r.Current[v]
		if !ok {
			continue // variable not returned (e.g. unknown var); skip
		}
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			continue // non-numeric (e.g. the "time" field never matches a var)
		}
		if v == "pressure_msl" {
			f *= hPaToInHg
		}
		out[k] = f
	}
	return out, nil
}
