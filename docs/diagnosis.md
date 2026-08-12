# Furnace diagnosis — WaterFurnace 7 Series (NVV048)

Findings from live reads over the recovered `request.cgi` Modbus passthrough,
2026-08-11. Unit at `<controller-ip>`, Modbus addr 1. This is the furnace-health
record; the protocol reverse-engineering is in [`protocol/`](protocol/).

## Unit

| | |
|---|---|
| Model | `NVV048C111CTL0KN` — 7 Series, ~4 ton, **variable speed** |
| Serial | *(redacted)* |
| ABC board | `ABCVSP` v3.05 rev 2 (AXB present, v2.09) |
| Loop | **closed, antifreeze** (reg 402 = 485) |
| DHW / desuperheater | **enabled**, setpoint 130 °F (reg 400 = 1, reg 401 = 1300) |

## Fault history (regs 601–699, lifetime occurrence counts)

| Code | Count | Meaning | Read |
|---|---:|---|---|
| **E5** | **646** | **Frze Detect FP1** — source/loop-side freeze | **the standout; also the last hard lockout** |
| E15 | ≥999* | Hot Water Limit | **benign** — normal desuperheater high-limit (see below) |
| E55 | ≥999* | Out of Envelope | VS compressor driven past its pressure/temp envelope |
| E99 | 55 | System Reset | power cycles / resets |
| E19 | 6 | Crit Comm Error | comms dropouts |
| E59 | 1 | Int Drive Error | one-off drive event |

*999 = counter ceiling; true count is higher. Counts are **cumulative over the
unit's life, not dated** — the Aurora stores occurrences, not timestamps. Last
lockout = **E5** (reg 26 = `0x8005` → code 5 + lockout bit).

## The freeze story (E5 / FP1)

FP1 is the **source/loop-side** freeze sensor — it trips when refrigerant at the
water coil gets too cold. 646 trips + E5 being the last lockout = a **chronic,
loop-side, winter-heating problem** (the on-card syslog is all January 2026; the
FP1 sensor itself reads fine, 74 °F live, so it's not a dead sensor).

**E55 "Out of Envelope" (≥999) corroborates it** — low suction from a cold or
starved loop pushes the VS compressor outside its envelope. E5 + E55 hammering
together is one story, not two.

### The VS pump is healthy and is NOT the lever it looks like

| reg | value | meaning |
|---|---|---|
| 325 | 55% | VS pump output (current) |
| 321 / 322 | 50% / 100% | VS pump Min / Max |
| 323 | 32767 | control = auto/proportional (not forced Min/Max) |
| 403 / 1117 | present / 84 | flow meter present, 8.4 gpm |
| 419 | 0 | loop-pressure trip **feature off / not equipped** (not a real 0 psi; no E21 faults) |

The pump runs **proportional to compressor speed**, Min→Max. Confirmed by the
math: at compressor speed 2/12, `50 + (2−1)/(12−1)×(100−50) = 54.5% ≈ 55%` —
matches reg 325 exactly. So **55% now is because the compressor is near idle, not
because of any flow decision.** Earlier "8.4 gpm is low for 4 tons" was a bad
comparison — at speed 2 you don't need 4-ton flow.

Two consequences that matter:
1. **The pump does not boost itself to prevent a freeze.** It tracks compressor
   speed open-loop; FP1 protection faults/locks out rather than ramping the pump.
   A perfectly-working pump and repeated freeze trips are fully compatible.
2. **Max is already 100%.** At the winter freeze events the compressor was at high
   speed, so the pump was already flat out. "Call for more pump" is not available
   at the moment it matters — so if it froze at 100% pump, the fix is not more pump.

### Working hypothesis + ranked suspects

The loop runs **too cold at peak heating**, and/or **can't deliver enough flow
even at 100% pump**. Ranked:

1. **Loop too cold at peak** — marginal antifreeze freeze-point and/or an
   undersized/heat-depleted ground loop. Most likely, given the pump control is
   healthy and has no headroom left at peak. → verify antifreeze freeze point vs
   the loop's winter minimum EWT.
2. **Actual flow short of spec even at 100% pump** — air in the loop, a worn
   circulator, or a restriction. → the high-load test below.
3. **FP1 freeze threshold set too high** (30 °F vs the 15 °F antifreeze setting).
   The DIP register (reg 4 = 32767 "manual", reg 33 = 255) *decodes* to 30 °F, but
   `0xFF` also decodes to "Single Stage" — impossible on a VS unit — so it's almost
   certainly an unpopulated default, not a real reading. **Verify the physical DIP
   on the ABC board**, don't trust the register.

Not the problem: loop pressure (feature off, zero E21), the FP1 sensor (healthy),
the pump control (to spec).

### The one test that forks it

Catch the unit at **high compressor speed** (cold-morning heat call, or force a
high stage) and read these **together**: compressor speed (3001), VS pump output
(325), water flow (1117), EWT (1111), FP1 (19).

- Pump pegged at 100% **and** flow still short → pump / loop / air (can't deliver).
- Flow fine near spec **but** EWT/FP1 cold → loop-temperature / antifreeze problem;
  pump exonerated.

This is the winter condition we can't see at today's idle speed. A one-shot
capture script for this is the obvious next tool.

## E15 "Hot Water Limit" — benign

This is the **desuperheater (hot water generator)** high-limit, not a failure.
DHW is enabled (reg 400 = 1) with a 130 °F setpoint (reg 401 = 1300). The unit
makes domestic hot water from waste compressor heat via the DHW pump (K6); when
the tank water reaches the 130 °F limit, the desuperheater backs off and the ABC
logs "Hot Water Limit." That happens **every time the water heater is satisfied**,
so a household racks up hundreds of counts over the unit's life — which is exactly
why E15 sits at the ≥999 ceiling and why it's the current stored code (reg 25 = 15).

It signals the desuperheater is **working and regularly satisfying the tank** — a
good sign, not a fault to chase. It would only warrant attention if E15 were
climbing while you were getting *no* hot water assist (a stuck/miswired high-limit
aquastat, air in the HWG loop, or a dead DHW pump tripping the limit prematurely) —
not the case here.

### Installation note + efficiency opportunity (this site)

Two water heaters in series; the desuperheater return tees into the branch
**between** them. If the **first (preheat) tank's** own element is set high, its
electricity heats the buffer water before the desuperheater can — the HWG then
sees already-hot water, hits the 130 °F limit immediately (E15), and captures
little free heat. Lowering (or disabling) the **first** tank's thermostat so it
acts as the desuperheater's buffer, while the **second (finish) tank** holds the
safe delivery temperature, lets the geo do the heating — most valuable in **summer
cooling**, when waste heat (heat of rejection) is abundant. Caveats: keep the
finish tank hot enough for **Legionella** safety (≥130–140 °F delivery / periodic
disinfection — a lukewarm preheat tank is the risk zone), and keep scald-safe
delivery (thermostatic mixing valve). Worth measuring before/after — see the
desuperheater duty-cycle logging idea (DHW pump K6 = reg 1104 bit0; Hot Water
limit input = reg 1103).

> **Update (2026-08-12):** live logging refined this. This unit's refrigerant
> thermistors (discharge / suction / superheat — regs 3325 / 1113 / 1125) are
> **not populated**; they read flat or NA and don't move with compressor load. On
> low-load cooling days the compressor idles at 2/12 with no discharge superheat to
> harvest, so the E15 trips in that state are just the K6 pump circulating
> already-hot, element-heated tank water into the aquastat — the geo contributes
> ~nothing. So the "lower the first tank" idea above is **not** worth chasing in
> summer on this unit, and lowering a storage tank toward ~95 °F to make headroom
> would sit squarely in the Legionella growth band — don't. Keep both tanks at a
> safe temperature; real desuperheater harvest is a high-compressor-stage (winter
> heating / peak cooling) phenomenon. geopilot's dashboard now labels the
> desuperheater honestly by gating "Harvesting" on compressor stage, not the pump
> bit.

## Live power / operating point (idle-ish, cooling, this capture)

Cooling, Normal status. Total draw ~475–550 W (compressor ~326–400, blower ~98,
DHW/loop pump ~51, aux 0). Line 221–225 V. EWT 68 °F / LWT 73 °F, ΔT ~+4.7 °F,
8.4 gpm, compressor 2/12, blower 55%. All consistent with light summer cooling.

## Open items

- [ ] High-load capture during a real heat call (the fork test above).
- [ ] Physically verify the FP1 DIP setting on the ABC board (15 °F for antifreeze).
- [ ] Verify antifreeze freeze point vs the loop's winter minimum EWT.
- [ ] Inspect the Geo-Flo flow center: circulator condition, purge for air.
