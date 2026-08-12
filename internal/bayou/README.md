# bayou — Bayou Energy utility-data client

A small, dependency-free Go client for the [Bayou Energy](https://docs.bayou.energy)
utility-data API. It pulls **bills** (structured line items — no PDF parsing) and
**interval usage** for a utility account the customer has authorized. Nothing here
is geopilot-specific; drop the package into any Go project.

In geopilot it's the "billing/rate" source: bills give the effective $/kWh for the
cost model, intervals give whole-house usage for load-shape analysis. Same
`series`/`readings` model as every other source.

## API shape (v2)

- Base `https://bayou.energy/api/v2`, HTTP Basic auth (**API key as username**, blank password).
- `POST /customers` → create; `GET /customers/{id}` → onboarding link + status.
- `GET /customers/{id}/bills` → `[]Bill` (consumption Wh, amounts in cents, delivery/supply split, PDF url).
- `GET /customers/{id}/intervals` → 15/30-min usage per meter (Wh; net/generated/demand).

## Get a key and connect an account

1. Sign up at **bayou.energy** — the first meter is free.
2. Copy your **API key** into `geopilot/.env` as `BAYOU_API_KEY=…` (git-ignored).
3. **Test on the sandbox first** (no real utility needed — Bayou's fake utility
   "Speculoos Power", api_code `speculoos`, accepts test logins like
   `iamvalid@bayou.energy` with any password):

   ```bash
   cd geopilot
   go run ./cmd/bayou --create --utility speculoos --email iamvalid@bayou.energy
   # open the printed onboarding link, "log in", then:
   go run ./cmd/bayou --customer <id> --status
   go run ./cmd/bayou --customer <id> --bills
   go run ./cmd/bayou --customer <id> --intervals
   ```

4. For the **real account**: confirm Bayou covers National Grid Upstate NY (ask
   support if it's not listed — they add utilities on request), create a customer
   on that utility, open the onboarding link, and authorize your National Grid
   login **at Bayou** (credentials go to Bayou, not into this repo). Put the
   resulting customer id in `.env` as `BAYOU_CUSTOMER_ID`.

## Reuse

The client (`internal/bayou`) is standalone. The CLI (`cmd/bayou`) is a thin
wrapper for testing/onboarding. To wire it into a data pipeline, call `GetBills`
/ `GetIntervals` on a schedule (bills refresh ~monthly, intervals ~daily) and map
the results into your store — see the planned geopilot ingester.
