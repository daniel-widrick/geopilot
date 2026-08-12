package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// latestAPI reads a westmoreland.app-style timeseries proxy: one GET per key to
// /api/latest?key=… returning {"t":<unix>,"v":<number>}. A 204 means "no value
// yet" and drops that key silently.
type latestAPI struct {
	base string
	http *http.Client
}

type latestPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

func (c *latestAPI) Fetch(ctx context.Context, keys []string) (map[string]float64, error) {
	out := make(map[string]float64, len(keys))
	for _, key := range keys {
		v, ok, err := c.one(ctx, key)
		if err != nil {
			return out, err
		}
		if ok {
			out[key] = v
		}
	}
	return out, nil
}

func (c *latestAPI) one(ctx context.Context, key string) (float64, bool, error) {
	u := c.base + "/api/latest?key=" + url.QueryEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return 0, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("weather %s: http %d", key, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, err
	}
	var p latestPoint
	if err := json.Unmarshal(b, &p); err != nil {
		return 0, false, err
	}
	return p.V, true, nil
}
