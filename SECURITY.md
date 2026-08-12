# Security

**Read this before you patch the controller or expose geopilot.**

## The short version

Following this repo's local-access instructions puts your furnace controller on your
LAN with an **effectively unauthenticated read/write control API**, and stands up a
couple of unauthenticated services next to it. On a trusted home network that's a
reasonable tradeoff. Exposed to the internet — even by accident — it is exactly the
kind of industrial-control endpoint that automated scanners and ICS-targeting
attackers hunt for. **Do not expose any of this to the internet.**

## What the patch exposes

The firmware patch ([`docs/local-access.md`](docs/local-access.md), Route B) removes
the gates that hid the controller's web interface on the LAN. Once patched, anything
on your network can reach `request.cgi`, the Modbus passthrough to the Aurora control
board:

- **Reads** — every register: temperatures, power, faults, configuration.
- **Writes** — `putregs` can change live control registers: setpoints, fault resets,
  and other operating parameters.

The controller's only access control is a passcode (`9999`) that is **hardcoded and
universal** — it's the AID Tool passcode, and it's documented right here — and the
unlock is a **single global device flag**: once any client authenticates, arbitrary
read/write is open to *every* client until the device reboots. Because geopilot's
collector authenticates on startup and stays running, in practice the device sits
**unlocked** — any host on the LAN can read and write furnace registers **without even
knowing the passcode**. Treat it as an open, unauthenticated control surface.

Why it matters: this is HVAC / geothermal control. Write access means someone on the
segment can change how your heat pump runs — comfort, equipment stress, wasted energy,
forced fault/lockout, or masking a real fault. And a Modbus-style endpoint answering on
a network is a recognizable, attractive target; if it is ever reachable from the
internet, expect automated discovery (Shodan / Censys) within days.

## geopilot's own surfaces

- **Dashboard + WebSocket** (default `:8080`) — **no authentication**. Anyone who can
  reach the port sees everything, and it's a potential foothold.
- **TimescaleDB** — **not** published to the host; it's reachable only by the
  collector over the internal compose network. Keep it that way, and still change
  `DB_PASSWORD` from the default.
- **`.env`** — holds `DB_PASSWORD` and your `BAYOU_API_KEY`. Keep it off the repo
  (it's git-ignored) and off shared hosts.

## What to do

1. **Never port-forward, DMZ, or UPnP-expose** the controller, the dashboard, or the
   database. For remote access use a VPN (WireGuard / Tailscale), never a public port.
2. **Segment the network.** Put the AWL (and ideally geopilot) on an isolated IoT VLAN
   that guest/untrusted devices can't reach, with firewall rules limiting who can talk
   to the controller.
3. **Bind services narrowly.** The database ships with no published port (internal to
   the compose network) — keep it that way. Only expose the dashboard on an interface
   you control, and put an authenticating reverse proxy in front of it if others share
   the LAN.
4. **Change defaults.** Set a real `DB_PASSWORD`. The furnace passcode *can't* be
   changed (it's in firmware) — which is precisely why network isolation, not the
   passcode, is the real control.
5. **Prefer the reversible route for casual use.** The Soft-AP route
   ([`docs/local-access.md`](docs/local-access.md), Route A) never puts the controller
   on your LAN — lower exposure — though it can't run geopilot.

## The honest bottom line

The passcode is not a security boundary; **your network is.** geopilot assumes a
trusted LAN. If that assumption doesn't hold for your network, isolate these services
until it does.

## Reporting

Found a security issue in geopilot itself? Open a GitHub issue or contact the
maintainer. No warranty is provided; see [`LICENSE`](LICENSE).
