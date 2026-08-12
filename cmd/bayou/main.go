// Command bayou exercises the Bayou Energy client — create a (sandbox) customer,
// print the onboarding link, and dump bills/intervals. Reads BAYOU_API_KEY from
// the environment.
//
//	export BAYOU_API_KEY=...          (or put it in .env alongside)
//	go run ./cmd/bayou --create --utility speculoos --email iamvalid@bayou.energy
//	go run ./cmd/bayou --customer 123 --status
//	go run ./cmd/bayou --customer 123 --bills
//	go run ./cmd/bayou --customer 123 --intervals
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/daniel-widrick/geopilot/internal/bayou"
)

// loadDotenv sets vars from a .env file (real env wins); keeps secrets off the CLI.
func loadDotenv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
}

func main() {
	var (
		create    = flag.Bool("create", false, "create a customer and print the onboarding link")
		utility   = flag.String("utility", "speculoos", "utility api_code (sandbox = speculoos)")
		email     = flag.String("email", "", "customer email (sandbox: iamvalid@bayou.energy)")
		customer  = flag.Int("customer", 0, "existing customer id")
		status    = flag.Bool("status", false, "print customer connection status")
		bills     = flag.Bool("bills", false, "print bills")
		intervals = flag.Bool("intervals", false, "print interval summary")
	)
	flag.Parse()

	loadDotenv(".env")
	key := os.Getenv("BAYOU_API_KEY")
	if key == "" {
		log.Fatal("set BAYOU_API_KEY (env or ./.env)")
	}
	c := bayou.New(key)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	if *create {
		cust, err := c.CreateCustomer(ctx, bayou.CreateCustomerOpts{Utility: *utility, Email: *email})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("customer id : %d\n", cust.ID)
		fmt.Printf("onboarding  : %s\n", cust.OnboardingLink)
		fmt.Println("-> open that link, authorize the utility account, then re-run with --customer", cust.ID, "--status")
		return
	}

	if *customer == 0 {
		log.Fatal("need --create or --customer <id>")
	}

	if *status {
		cust, err := c.GetCustomer(ctx, *customer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("customer %d: authed=%v bills_ready=%v intervals_ready=%v (utility_has_intervals=%v)\n",
			cust.ID, cust.IsCurrentlyAuthenticated, cust.BillsAreReady, cust.IntervalsAreReady, cust.UtilityHasIntervals)
	}

	if *bills {
		bs, err := c.GetBills(ctx, *customer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("=== %d bills ===\n", len(bs))
		fmt.Printf("%-12s %-12s %8s %8s %9s %9s %9s %8s\n", "from", "to", "kWh", "elec$", "$/kWh", "deliv$", "supply$", "total$")
		for _, b := range bs {
			fmt.Printf("%-12s %-12s %8.0f %8.2f %9.4f %9.2f %9.2f %8.2f\n",
				b.BillingPeriodFrom, b.BillingPeriodTo, b.KWh(),
				float64(b.ElectricityAmount)/100, b.DollarsPerKWh(),
				float64(b.DeliveryCharge)/100, float64(b.SupplyCharge)/100,
				float64(b.TotalAmount)/100)
		}
	}

	if *intervals {
		iv, err := c.GetIntervals(ctx, *customer, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("=== intervals %s .. %s  granularities=%v ===\n",
			iv.FirstIntervalDiscovered.Format("2006-01-02"),
			iv.LastIntervalDiscovered.Format("2006-01-02"), iv.Granularities)
		for _, m := range iv.Meters {
			var total int64
			for _, in := range m.Intervals {
				if in.ElectricityConsumption != nil {
					total += *in.ElectricityConsumption
				}
			}
			fmt.Printf("meter %s: %d intervals, %.1f kWh total\n", m.ID, len(m.Intervals), float64(total)/1000)
		}
	}
}
