# Getting your AWL controller back on the LAN

WaterFurnace's Symphony cloud (`awldeviceproxy.mywaterfurnace.com`) is dead. On a
stock **AWL (Aurora WebLink)** bridge that leaves you with two problems:

1. **It reboots every ~30 minutes.** The controller reboots itself whenever the
   cloud WebSocket times out — which is now always.
2. **Its web UI won't serve on your LAN.** Once the AWL is joined to your Wi-Fi
   and holds an IP (*infrastructure mode*), the web server answers only
   `runcmd.cgi` and a hardcoded `status.cgi` stub. The real interface —
   `index.htm`, `config.htm`, and the `request.cgi` Modbus passthrough that
   geopilot needs — is served **only in Soft-AP mode**. This is deliberate: the
   device was meant to be driven exclusively through the (now dead) cloud.

There are two ways to regain access. **They are not equivalent** — read the
comparison before picking one.

| | Config edit (`AWL.INI`) | Firmware patch |
|---|---|---|
| Stops the 30-min reboot | ✅ | ✅ |
| Serves full UI + `request.cgi` | ✅ *in Soft AP only* | ✅ *in infrastructure mode* |
| Reachable **on your home LAN** | ❌ device is its own AP (`192.168.1.3`) | ✅ (its normal DHCP address) |
| Modifies firmware / bricking risk | ❌ none, reversible | ⚠️ yes |
| Good for | one-time setup, manual diagnostics | **24/7 monitoring (geopilot)** |

---

## Route A — config edit (no flash, reversible)

Edit `bin/AWL.INI` on the SD card (**back the card up first** — the running device
rewrites this file):

```ini
[Setup]
LogOnlyMode=1        ; was 0 — stops the 30-min "WebSocket Timeout" reboot loop
BootLocalMode=1      ; was 0 — boot straight into Soft-AP with the full web UI
LocalModeTimeout=0   ; was 30 — never bounce back out of Soft-AP to Wi-Fi
```

`LogOnlyMode=1` and `LocalModeTimeout=0` are independent; either can be applied
alone. On boot the device comes up as **its own Wi-Fi access point**, serving the
full UI and `request.cgi` at **`192.168.1.3`** (also `172.20.10.1`), and
`LocalModeTimeout=0` keeps it from bouncing back out.

You can also trigger Soft-AP without editing the file — `GET
/runcmd.cgi?cmd=local` (that endpoint *is* served on the LAN), or holding the
device button for >5 s — but without `LocalModeTimeout=0` it reverts after 30 min.

> **The catch:** Soft-AP is *local* access, not *LAN* access. While in Soft-AP the
> device is not joined to your home network — you connect a client to the AWL's own
> access point to reach `192.168.1.3`. That's fine for setup or reading registers by
> hand, but a background collector can't poll a device that has left your network.
> **This route alone will not run geopilot.**

## Route B — firmware patch (permanent LAN access; required for geopilot)

To get the full UI and `request.cgi` served **in infrastructure mode — on your
actual LAN, alongside everything else, 24/7** — the file-serving restriction has to
be removed from the compiled HTTP handler itself; no config flag can do it. At a
high level:

1. Patch `bin/factory.ahf` (an Intel-HEX image) to neutralize the two gates in the
   HTTP request handler that restrict LAN serving to `runcmd.cgi` + the stub.
2. Recompute the MD5 in the image's `#` header — that hash is the image's **only**
   integrity check (there is no signature), so the bootloader accepts a correctly
   re-hashed image.
3. Set `ForceLoadFlag=1` in `bin/BOOT.INI` so the resident bootloader reflashes the
   patched image from the SD card on next boot.

Result: the device serves its full diagnostic UI and Modbus interface at its normal
DHCP address on your LAN, with no cloud and no Soft-AP hop. That LAN reachability is
exactly what geopilot's collector needs.

### What the two gates are

Both restrictions live in the firmware's HTTP request handler and are guarded by the
same condition — in effect, *"this client is on the home network in normal
(infrastructure) mode, and holds an IP — i.e. it is not the device's own Soft-AP."*
While that condition holds, two things happen:

1. **The file-serving gate.** Early in request dispatch, the handler lets only
   `runcmd.cgi` through and refuses every other path. That is precisely why
   `index.htm`, `config.htm`, and `request.cgi` come back rejected when the device is
   on your LAN. In Soft-AP mode the guard condition is false, so files serve normally
   — which is the whole reason the stock workaround is to force Soft-AP.

2. **The stub-redirect gate.** A little further along, when a request would otherwise
   be served, a second conditional keyed on the same "on-LAN, not Soft-AP" state
   substitutes the hardcoded `status.cgi` placeholder for the real page.

Both gates exist only to funnel a LAN client back toward cloud-era behavior; neither
is a safety interlock. Making the handler follow its normal (Soft-AP) serving path
*regardless* of network mode — i.e. neutralizing both conditions so they no longer
divert LAN requests — is what lets the full UI and `request.cgi` answer over
infrastructure Wi-Fi. Finding the two conditionals in your own image and doing that is
a disassembler exercise left to the reader; afterward, recompute the header MD5 and
set `ForceLoadFlag=1` as above.

> **Risks — you own them.** This modifies the controller's firmware. A bad flash
> can brick the board (mitigation: the MD5 check means a *mis-hashed* image is
> rejected rather than run, and keeping an untouched backup of the SD card lets you
> roll back). Separately, patching firmware and defeating an integrity check may
> implicate anti-circumvention law (e.g. DMCA §1201 in the US) depending on your
> jurisdiction and purpose; repairing a device you own after the manufacturer
> abandoned its cloud is the sympathetic case, but this is **not legal advice**.

> **Security — read this.** Patching also removes the barrier that kept the
> controller's control API off your LAN. The result is an **effectively
> unauthenticated read/write endpoint** on your network (see why in
> [`../SECURITY.md`](../SECURITY.md)). Keep it on a trusted, segmented network and
> **never expose it to the internet.**
> The exact byte-level patch is intentionally not published here.

---

## Which should I use?

- **Just want to read your furnace by hand, or do one-time setup?** Route A. It's
  reversible and can't brick anything.
- **Want continuous monitoring / geopilot?** Route B is required — the collector
  polls the controller over your LAN, which only the patch enables.

Once the controller is reachable and serving `request.cgi`, see
[`protocol/request-cgi.md`](protocol/request-cgi.md) for how to talk to it.
