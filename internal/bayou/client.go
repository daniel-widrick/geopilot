// Package bayou is a small, standalone client for the Bayou Energy utility-data
// API (https://docs.bayou.energy). It has no geopilot dependencies and can be
// lifted into any project. Auth is HTTP Basic with the API key as the username.
package bayou

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://bayou.energy/api/v2"

type Client struct {
	apiKey string
	base   string
	http   *http.Client
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey, base: DefaultBaseURL, http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) WithBaseURL(u string) *Client { c.base = strings.TrimRight(u, "/"); return c }

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.apiKey, "") // API key as username, empty password
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("bayou %s %s: http %d: %s", method, path, resp.StatusCode, trunc(b))
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("bayou decode %s: %w", path, err)
		}
	}
	return nil
}

// ---- customers -----------------------------------------------------------

type Customer struct {
	ID                       int    `json:"id"`
	Email                    string `json:"email"`
	Utility                  string `json:"utility"`
	HasFilledCredentials     bool   `json:"has_filled_credentials"`
	IsCurrentlyAuthenticated bool   `json:"is_currently_authenticated"`
	BillsAreReady            bool   `json:"bills_are_ready"`
	IntervalsAreReady        bool   `json:"intervals_are_ready"`
	UtilityHasIntervals      bool   `json:"utility_has_intervals"`
	AccountHasIntervals      bool   `json:"account_has_intervals"`
	OnboardingLink           string `json:"onboarding_link"`
	OnboardingToken          string `json:"onboarding_token"`
}

type CreateCustomerOpts struct {
	Email      string `json:"email,omitempty"`
	Utility    string `json:"utility,omitempty"` // api_code, e.g. "speculoos" for the sandbox
	ExternalID string `json:"external_id,omitempty"`
}

func (c *Client) CreateCustomer(ctx context.Context, o CreateCustomerOpts) (*Customer, error) {
	b, _ := json.Marshal(o)
	var cust Customer
	if err := c.do(ctx, http.MethodPost, "/customers", strings.NewReader(string(b)), &cust); err != nil {
		return nil, err
	}
	return &cust, nil
}

func (c *Client) GetCustomer(ctx context.Context, id int) (*Customer, error) {
	var cust Customer
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/customers/%d", id), nil, &cust); err != nil {
		return nil, err
	}
	return &cust, nil
}

// ---- bills ---------------------------------------------------------------

type Bill struct {
	ID                     int64  `json:"id"`
	AccountNumber          string `json:"account_number"`
	Status                 string `json:"status"`
	BilledOn               string `json:"billed_on"`
	BillingPeriodFrom      string `json:"billing_period_from"`
	BillingPeriodTo        string `json:"billing_period_to"`
	ElectricityConsumption int64  `json:"electricity_consumption"` // watt-hours
	ElectricityAmount      int64  `json:"electricity_amount"`      // cents (electric only)
	DeliveryCharge         int64  `json:"delivery_charge"`         // cents
	SupplyCharge           int64  `json:"supply_charge"`           // cents
	GasConsumption         string `json:"gas_consumption"`         // string in the API (e.g. "50.000")
	GasAmount              int64  `json:"gas_amount"`
	TotalAmount            int64  `json:"total_amount"` // cents (electric + gas)
	FileURL                string `json:"file_url"`
}

func (c *Client) GetBills(ctx context.Context, customerID int) ([]Bill, error) {
	var bills []Bill
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/customers/%d/bills", customerID), nil, &bills); err != nil {
		return nil, err
	}
	return bills, nil
}

// KWh returns billed electricity in kWh.
func (b Bill) KWh() float64 {
	return float64(b.ElectricityConsumption) / 1000.0
}

// DollarsPerKWh is the effective ELECTRIC rate (electricity_amount / kWh) — not
// total_amount, which can include gas on a combined bill.
func (b Bill) DollarsPerKWh() float64 {
	kwh := b.KWh()
	if kwh == 0 {
		return 0
	}
	return float64(b.ElectricityAmount) / 100.0 / kwh
}

// ---- intervals -----------------------------------------------------------

type Interval struct {
	Start                     time.Time `json:"start"`
	End                       time.Time `json:"end"`
	ElectricityConsumption    *int64    `json:"electricity_consumption"` // watt-hours
	NetElectricityConsumption *int64    `json:"net_electricity_consumption"`
	GeneratedElectricity      *int64    `json:"generated_electricity"`
	ElectricityDemand         *int64    `json:"electricity_demand"` // watts
	GasConsumption            *float64  `json:"gas_consumption"`
}

type IntervalMeter struct {
	ID        string     `json:"id"`
	Intervals []Interval `json:"intervals"`
}

type Intervals struct {
	FirstIntervalDiscovered time.Time       `json:"first_interval_discovered"`
	LastIntervalDiscovered  time.Time       `json:"last_interval_discovered"`
	Granularities           []int           `json:"granularities"`
	Meters                  []IntervalMeter `json:"meters"`
}

func (c *Client) GetIntervals(ctx context.Context, customerID int, params url.Values) (*Intervals, error) {
	p := fmt.Sprintf("/customers/%d/intervals", customerID)
	if len(params) > 0 {
		p += "?" + params.Encode()
	}
	var iv Intervals
	if err := c.do(ctx, http.MethodGet, p, nil, &iv); err != nil {
		return nil, err
	}
	return &iv, nil
}

func trunc(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
