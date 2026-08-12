# geopilot

Polls a WaterFurnace 7 Series (Aurora / AWL controller) over the recovered
`request.cgi` Modbus passthrough, stores everything in TimescaleDB, and serves a
live dashboard — homeowner summary above the fold, freeze/desuperheater/pump
diagnostics below. Built to grow into whole-home energy automation (water-heater
probes, power prices, weather, and control).

> **Prerequisite:** geopilot polls the controller over your LAN, which a stock
> cloud-era AWL won't serve. See [`docs/local-access.md`](docs/local-access.md) to
> get yours answering `request.cgi` on the network first.

## Run

```bash
cd geopilot
cp .env.example .env          # set DB_PASSWORD, confirm AWL_URL
go mod tidy                   # resolve go.sum (needs internet once)
docker compose up -d --build
```

- Dashboard: http://localhost:8080
- TimescaleDB (user/db `geopilot`) runs internal to the compose network — it is not
  published to the host. Reach it for debugging with `docker compose exec timescaledb
  psql -U geopilot`.

The collector auto-authenticates (AID passcode) and **re-auths automatically after
a controller reboot/outage** — the unlock is a device-global flag that clears on
power loss, so no manual intervention is needed.

## How it's shaped

| Piece | What it does |
|---|---|
| `series` table | the poll plan *and* the catalog — one row per signal, with `unit`, `scale`, `signed`, and a `tier` (fast/medium/slow). Add a row → it gets polled. Seeded from the reverse-engineered register map (`migrations/002_seed_furnace.sql`, 558 furnace signals). |
| `readings` hypertable | narrow `(time, series_id, value)`; raw integer values, decoded at read time. Compresses columnar after 30 days (segment by series), 5-year retention, 1-min/1-hour continuous aggregates. |
| collector | tiered pollers (fast 5s / medium 60s / slow 1h), batched `getregs`, writes to DB + an in-memory live snapshot. |
| web | serves the dashboard and pushes the decoded model over a WebSocket (no polling, no refresh). `/api/history?series=<key>&hours=N` for trends. |

`series.source` is generic (`furnace` today) so future inputs — `waterheater`,
`power`, `weather` — land in the same table and timeline for automation.

## Data model notes

- Values are stored **raw** (16-bit register). `display = raw * scale`, with
  signed 16-bit interpretation when `signed` is set. `-9999` (`55537`) is the
  no-sensor sentinel. Derived metrics (power hi/lo pairs, heat of extraction) are
  computed from raw registers in `internal/model`.
- The register meanings, fault codes, and equations are documented in
  [`docs/protocol/`](docs/protocol/); [`docs/diagnosis.md`](docs/diagnosis.md) is a
  worked furnace-diagnosis example.

## Roadmap

1. **(this) data backbone + live dashboard.**
2. Additional sources: water-heater temp probes, real-time power price, weather.
3. Historical charts on the dashboard (the `/api/history` endpoint + continuous aggregates are ready).
4. Control: setpoint automation, desuperheater optimization, load-shifting on price/weather (writes via `putregs`).

## Documentation

- [`docs/local-access.md`](docs/local-access.md) — get your AWL controller on the LAN
- [`docs/protocol/`](docs/protocol/) — the `request.cgi` interface, register map, dealer/IZ2 catalog, and fault codes
- [`docs/diagnosis.md`](docs/diagnosis.md) — a worked furnace-diagnosis example
- [`docs/billing-bayou.md`](docs/billing-bayou.md) — utility bills & rates via Bayou Energy (for the cost readouts)

## Security

Patching the controller for LAN access (and running geopilot against it) leaves an
**effectively unauthenticated read/write control API** on your network, plus an
unauthenticated dashboard and database. This is safe only on a trusted, segmented LAN
— **never expose it to the internet.** Read [`SECURITY.md`](SECURITY.md) before you
patch or deploy.

## Disclaimer

Independent, unofficial interoperability and repair software for hardware you own —
**not affiliated with or endorsed by WaterFurnace International**. See
[`NOTICE`](NOTICE). Licensed under [MIT](LICENSE).
