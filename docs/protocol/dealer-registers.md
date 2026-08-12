# Dealer / IZ2 register catalog (indexdmc.js)

The dealer UI groups registers by page. `read` pages monitor; `write` pages are setup forms. Where a page offers a `31xxx` and a `12xxx` list, `31xxx` is the IntelliZone-2 primary and `12xxx` the AXB/legacy variant. `base,count` means a run of `count` consecutive registers. Bit-packed values mean an exact per-row register split is not always 1:1 — the page grouping is the reliable unit.

> **Cross-checked 2026-08-12** with [`ccutrer/waterfurnace_aurora`](https://github.com/ccutrer/waterfurnace_aurora).
> The zone-block structure agrees exactly: zone read data starts at **base 31007,
> stride 3** per zone, with `31200 + 3·(zone−1)` holding zone size/priority. His map
> adds the per-zone **setpoint write** block and reg **31003 = IZ2 Outdoor Temperature**
> (see "Zone control" at the bottom); ours is the more complete for the fault-history
> (31300–31328) and IZ2-config (31100/31101) blocks, which his map does not cover. Also
> confirmed: reg **54** = ECM/blower actual speed, **3027** = compressor speed, **31005**
> = demand, **31400** = dealer name, and reg **30** = live *System Outputs* (the same
> bitfield reg 27 snapshots at lockout).

### Accessories  _(read)_

`31102;31103;31104;31105;`

Rows: Air Filter Mode · Air Filter Run Hours · Air Filter Calendar · Humidifier Mode · Humidifier Run Hours · Humidifier Calendar · UV Lamp Mode · UV Lamp Run Hours · UV Lamp Calendar · Air Cleaner Mode · Air Cleaner Run Hours · Air Cleaner Calendar

### Dealer Information  _(read)_

`31400,73;`

Rows: Dealer Name · Dealer Phone · Dealer Address 1 · Dealer Address 2 · Dealer Email · Dealer Website

### Dealer Information  _(read)_

`12400,73;`

Rows: Dealer Name · Dealer Phone · Dealer Address 1 · Dealer Address 2 · Dealer Email · Dealer Website

### Equipment  _(read)_

`404;480;92,13;105,5`

Rows: Equipment Type · Blower Type · Serial Number · Model Number

### Equipment Status  _(read)_

`30;31;35;54;480;3027;31002;31025;31101;31004;31005;31109;31110`

Rows: Zone Call Mode · Zone Call Comp Speed · Zone Call Fan % · Actual Mode · Actual Comp Speed · Actual Fan Speed · Unit Demand · Fan Demand · Dehumid · Dehumid · Heat Staging · Cool Staging · Fan Staging · Aux Heat · Damper · Comp Type

### Fault Status  _(read)_

`31300,29;`

Rows: Fault Status

### Fault Status  _(read)_

`12200,29;`

Rows: Fault Status

### Fe  _(write)_

`12301;`

### Humidity Control & Offset  _(read)_

`31109;`

Rows: Humidity Control · Humidity Offset

### Humidity Control & Offset  _(read)_

`12309;`

Rows: Humidity Control · Humidity Offset

### IZ2 Configuration  _(read)_

`480;31101;`

Rows: Equipment Type · Max Zones · Number of Zones · Damper · Heat Staging · Heat Staging % · Cool Staging · Cool Staging % · Thermostat Type · Fan With Heat · Aux Heat Lockout

### IZ2 Settings  _(read)_

`31100;`

Rows: Fahrenheit/Celsius · Time Format · Daylight Saving Time · AWL Time Synch · Monitor AWL Status · Data Logging

### IZ2 Settings  _(read)_

`12300;`

Rows: Fahrenheit/Celsius · Time Format · Daylight Saving Time · AWL Time Synch · Monitor AWL Status · Data Logging

### Oc  _(write)_

`21107;21108;21109;21110;`

### Oc  _(write)_

`12302;12303;12304;12305;`

### Select Differentials  _(read)_

`31107`

Rows: 1st Stage · 2nd Stage · Aux Heat

### Select Differentials  _(read)_

`12307`

Rows: 1st Stage · 2nd Stage · Aux Heat

### Status  _(read)_

`30;35;54;480;12002;12004;12308;3027`

Rows: Mode · Compressor · Fan · Aux Heat · Room Temp · Remote Temp · Outdoor Temp

### TStat Configuration  _(read)_

`12301;`

Rows: TStat Type · Fan With Heat · Cycles per Hour · Smart Recovery · Compressor Satisfy · Cooling Lockout · Aux Heat Lockout · Auto / Manual · Auto Changeover Time

### Temp Sensors & Offsets  _(read)_

`31108;`

Rows: Remote Sensor ·  · Indoor Temp Offset · Remote Temp Offset · Outdoor Temp Offset

### Temp Sensors & Offsets  _(read)_

`12308;`

Rows: Remote Sensor ·  · Indoor Temp Offset · Remote Temp Offset · Outdoor Temp Offset

### Wc  _(write)_

`21125;`

### Wc  _(write)_

`12617;`

### Yd  _(write)_

`21114;`

### Zone Configuration  _(read)_

`31101;31200;31203;31206;31209;31212;31215`

Rows: Number Of Zones

### Zone Status  _(read)_

`31101;31007,18;31200;31203;31206;31209;31212;31215`

### (ce)  _(write)_

`21106;`

### (he)  _(write)_

`21105;`

### (he)  _(write)_

`12300;`

### (pe)  _(write)_

`21113;`

### (pe)  _(write)_

`12308;`

### (td)  _(write)_

`21112;`

### (td)  _(write)_

`12307;`

---

## Zone control — cross-checked with ccutrer/waterfurnace_aurora

**Read (per zone, stride 3 from base 31007):** `31007 + 3·(z−1)` = zone ambient temp;
`+1` / `+2` (i.e. `31008`/`31009` for zone 1) are packed config words carrying call,
mode, damper, and the setpoints. geopilot decodes the 32-bit pair as `heatSP =
bits 11–16 + 36 °F`, `coolSP = bits 17–22 + 36 °F` — **live-verified** (Z1 70/76,
Z2 68/72). `31200 + 3·(z−1)` = zone size / priority. ccutrer's map confirms this
base/stride independently (he labels the two config words "Configuration 1/2" with a
setpoint carry bit spanning them).

**Write (per zone, stride 9) — VERIFIED live 2026-08-12.** Setpoints are set with a
single `putregs`, value encoded as **°F × 10** (72.0 °F → `720`):

| register (zone z) | field |
|---|---|
| `21203 + 9·(z−1)` | **Heating Setpoint** (°F × 10) |
| `21204 + 9·(z−1)` | **Cooling Setpoint** (°F × 10) |
| `21202 + 9·(z−1)` | Heating Mode |
| `21205 + 9·(z−1)` | Fan Mode |

So Z1 cool = `21204`, Z2 cool = `21213`; Z1 heat = `21203`, Z2 heat = `21212`. Confirmed
by `putregs 21204,720`, then watching the read-mirror `31008` change to coolSP 72.

Gotchas that make a correct write look dead:

- **It applies on the IZ2 sync cycle (~10–20 s), not instantly.** A readback within a few
  seconds still shows the old value — this repeatedly looked like "no effect."
- **Out-of-range / mis-scaled values are silently accepted, then ignored** — e.g. `0`, or a
  bare `72` (= 7.2 °F, below the 54–99 °F cooling range). Always send °F × 10.
- The read-mirror (`31007+3z` / `31008/9`) is **read-only** — writing it directly returns
  `Exception 2`. Read setpoints there; write them at the `21xxx` block.
- **Revert** by writing the prior value ×10 (e.g. `putregs 21204,760` restores 76 °F).

This is (almost certainly) the path the dead Symphony cloud used for remote thermostat
control — now recovered. It's the write target for per-zone setpoint automation
(roadmap item 4), and matches ccutrer/waterfurnace_aurora (`iz2_zone.rb`: cool/heat at
`21204`/`21203 + 9·(zone−1)`, value ×10). Note this contradicts an earlier working
assumption that setpoints were unwritable — they are, just at the `21xxx` block with the
×10 scaling and the sync delay, not the read-only `31xxx` mirror.

**Also:** reg **31003 = IZ2 Outdoor Temperature** (the `31xxx` primary; `12004` is the
`12xxx` variant our catalog already lists).

