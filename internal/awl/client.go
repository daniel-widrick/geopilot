// Package awl talks to the WaterFurnace AWL controller's request.cgi Modbus
// passthrough. The unlock granted by cmd=auth is a device-global flag that clears
// on reboot, so the client transparently re-auths whenever a read comes back
// Unauthorized (e.g. after a power outage).
package awl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	base     string
	passcode string
	addr     int
	http     *http.Client

	mu sync.Mutex
	id int

	// reqMu serializes access to the board: request.cgi is a single Modbus
	// bridge that tracks a busy state and mishandles concurrent reads, so only
	// one getregs/auth may be in flight at a time.
	reqMu sync.Mutex
}

func New(base, passcode string, addr int) *Client {
	return &Client{
		base:     strings.TrimRight(base, "/"),
		passcode: passcode,
		addr:     addr,
		http:     &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *Client) nextID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id++
	if c.id > 99 {
		c.id = 1
	}
	return c.id
}

// get issues one request.cgi call and returns the raw body. Serialized so the
// single-threaded Modbus bridge only ever sees one request at a time.
func (c *Client) get(ctx context.Context, q url.Values) (string, error) {
	c.reqMu.Lock()
	defer c.reqMu.Unlock()
	u := c.base + "/request.cgi?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	return string(b), nil
}

// Auth sends the AID passcode to unlock arbitrary getregs/putregs.
func (c *Client) Auth(ctx context.Context) error {
	id := c.nextID()
	q := url.Values{}
	q.Set("cmd", "auth")
	q.Set("id", strconv.Itoa(id))
	q.Set("set", strconv.Itoa(id))
	q.Set("addr", strconv.Itoa(c.addr))
	q.Set("passcode", c.passcode)
	body, err := c.get(ctx, q)
	if err != nil {
		return err
	}
	if !strings.Contains(body, "err=") || strings.Contains(body, "Unauthorized") || strings.Contains(body, "Invalid Passcode") {
		return fmt.Errorf("auth rejected: %s", strings.TrimSpace(body))
	}
	return nil
}

// GetRegs reads the given registers (1:1 with the returned values). It re-auths
// once and retries if the controller reports Unauthorized.
func (c *Client) GetRegs(ctx context.Context, regs []int) (map[int]int64, error) {
	if len(regs) == 0 {
		return map[int]int64{}, nil
	}
	out, err := c.getRegsOnce(ctx, regs)
	if err == errUnauthorized {
		if aerr := c.Auth(ctx); aerr != nil {
			return nil, fmt.Errorf("re-auth: %w", aerr)
		}
		out, err = c.getRegsOnce(ctx, regs)
	}
	return out, err
}

var errUnauthorized = fmt.Errorf("unauthorized")

func (c *Client) getRegsOnce(ctx context.Context, regs []int) (map[int]int64, error) {
	id := c.nextID()
	parts := make([]string, len(regs))
	for i, r := range regs {
		parts[i] = strconv.Itoa(r)
	}
	q := url.Values{}
	q.Set("cmd", "getregs")
	q.Set("id", strconv.Itoa(id))
	q.Set("set", strconv.Itoa(id))
	q.Set("addr", strconv.Itoa(c.addr))
	q.Set("regs", strings.Join(parts, ";"))

	body, err := c.get(ctx, q)
	if err != nil {
		return nil, err
	}
	// verify the reply matches our request before trusting it
	want := fmt.Sprintf("id=%d&set=%d", id, id)
	if !strings.Contains(body, want) {
		return nil, fmt.Errorf("id mismatch (stale reply): %q", trunc(body))
	}
	if strings.Contains(body, "err=Unauthorized") {
		return nil, errUnauthorized
	}
	if !strings.Contains(body, "err=&") {
		return nil, fmt.Errorf("read error: %q", trunc(body))
	}
	vals := valuesOf(body)
	if len(vals) != len(regs) {
		return nil, fmt.Errorf("got %d values for %d regs", len(vals), len(regs))
	}
	out := make(map[int]int64, len(regs))
	for i, r := range regs {
		out[r] = vals[i]
	}
	return out, nil
}

// valuesOf extracts the comma-separated ints after "values=".
func valuesOf(body string) []int64 {
	i := strings.Index(body, "values=")
	if i < 0 {
		return nil
	}
	rest := body[i+len("values="):]
	if j := strings.IndexAny(rest, "\r\n&"); j >= 0 {
		rest = rest[:j]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil
	}
	fields := strings.Split(rest, ",")
	out := make([]int64, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.ParseInt(strings.TrimSpace(f), 10, 64)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func trunc(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
