# Utility bills & rates via Bayou Energy

geopilot can show what your heat pump actually **costs** — dollars per hour right
now, dollars today — but only if it knows your electricity rate. That rate isn't on
the furnace; it's on your utility bill. [Bayou Energy](https://bayou.energy) is how
geopilot gets it.

## What Bayou is

Bayou is a service that connects to your electric utility account and re-exposes your
data through one clean API — structured **bills** (kWh used, the dollar amounts,
delivery vs. supply split) and, where the utility supports it, **interval usage**
(15/30-minute consumption). You authorize it once against your utility login; Bayou
handles the scraping/Green-Button/1st-party integration behind the scenes so you
don't have to parse bill PDFs or automate a utility website that fights automation.

## Why geopilot uses it

Your true cost per kWh isn't a single published number — it's the bill total
(supply + delivery + fees) divided by kWh, and it drifts every month. geopilot pulls
your recent bills, computes that **effective $/kWh**, and multiplies it by the geo's
live power draw to show real cost. Bills are the source; the dashboard's "Cost Now"
and "today" figures come straight from them.

It's optional. With no Bayou key configured, geopilot runs fine — you just don't get
the cost readouts.

## The honest tradeoff

Bayou is free for a single meter, which is why geopilot uses it. The cost isn't
money — it's trust: to read your account, Bayou needs authorized access to your
utility login, so you're handing a third party access to your energy account. That's
a deliberate choice. If you'd rather not, leave Bayou unconfigured and geopilot skips
cost (everything else still works). Your utility credentials go to **Bayou's**
onboarding page, never into this repo or geopilot's config — geopilot only ever holds
a Bayou **API key**.

## Setup

1. **Sign up** at [bayou.energy](https://bayou.energy) — the first meter is free —
   and copy your **API key**.
2. **Put it in `.env`** (git-ignored) in the `geopilot/` directory:
   ```ini
   BAYOU_API_KEY=your-key-here
   ```
3. **Try the sandbox first** — Bayou's fake utility "Speculoos Power" needs no real
   account:
   ```bash
   cd geopilot
   go run ./cmd/bayou --create --utility speculoos --email iamvalid@bayou.energy
   # open the printed onboarding link, "log in", then:
   go run ./cmd/bayou --customer <id> --status
   go run ./cmd/bayou --customer <id> --bills
   ```
4. **Connect your real account:** create a customer on your actual utility, open the
   onboarding link Bayou prints, and authorize your utility login **at Bayou**. Once
   the customer's `bills_are_ready`, put its id in `.env`:
   ```ini
   BAYOU_CUSTOMER_ID=your-customer-id
   ```
5. Restart the collector (`docker compose up -d --build`). It refreshes bills on the
   `POLL_BAYOU` cadence (default 12h) and the cost readouts light up.

## A note on National Grid (and coverage)

If your utility isn't in Bayou's list, ask their support — they add utilities on
request. One gotcha worth knowing: coverage isn't all-or-nothing. National Grid
Upstate NY, for example, exposes **bills** through Bayou but **not** interval usage,
so cost/rate works while load-shape analysis doesn't. Check `--status`
(`bills_are_ready` / `intervals_are_ready`) to see what your account actually offers.

## More detail

For the API shape, data types, and reusing the client outside geopilot, see
[`../internal/bayou/README.md`](../internal/bayou/README.md).
