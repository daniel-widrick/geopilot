# Driving the furnace through `request.cgi`

Confirmed working against a live unit on 2026-08-11, after
the LAN patch. `request.cgi` is a Modbus-register passthrough to the Aurora ABC
board over the serial cable — the same channel the AID Tool and the Symphony
cloud used.

## Request shape

```
GET /request.cgi?cmd=<cmd>&id=<id>&set=<set>&addr=<addr>[&regs=<list>][&passcode=<pc>]
```

| Field | Meaning |
|---|---|
| `cmd` | `getregs`, `putregs`, `auth`, or a predefined set name (`abcinfo`, `devices2`, `devices3`, `devices`) |
| `id` | request id, 1–99, echoed back — the UI rotates it and checks the reply matches |
| `set` | a second rotating counter, 1–99, also echoed and matched |
| `addr` | Modbus slave address of the target. `1` = the ABC board itself |
| `regs` | for `getregs`/`putregs`: `;`-separated. `N` = one register, `N,C` = a run of C registers from N. putregs uses `reg,val;reg,val;` |
| `passcode` | for `cmd=auth` only |

The reply echoes the request and adds `err=` and `values=`:

```
cmd=getregs&id=2&set=2&addr=1&err=&values=16973,0,727,683,...
```

`err=` **empty means success.** Other values seen in firmware: `Unauthorized`,
`Invalid Request`, `Invalid Passcode`, `Timeout`, `Exception <n>`. Always confirm
the returned `id`/`set` match what you sent before trusting `values`.

## Authorization

- **Predefined sets** (`abcinfo`, `devices2`, `devices3`, `devices`) need **no auth** —
  they are hardcoded register lists in the firmware.
- **Arbitrary `getregs` and all `putregs`** require a session unlock, or they
  return `err=Unauthorized`.

Unlock:

```
GET /request.cgi?cmd=auth&id=1&set=1&addr=1&passcode=9999
```

The passcode **`9999`** is hardcoded in the firmware (`custom_http_app.c`) — the
same passcode the AID Tool uses. On success the
reply is `...&err=` (empty). The unlock is a **single global flag**, not
per-connection — once any client authenticates, arbitrary reads/writes are open
to everyone until the next reboot. It clears on reboot.

> `putregs` writes live furnace control registers. It exists here for
> completeness; treat it as capable of changing unit operation and don't write
> registers you haven't identified.

## Reliability gotcha — use unique request IDs

`request.cgi` returns **stale or misaligned values if you reuse the `id`/`set`
numbers** across calls in a session (the board/CGI keys its response on them). Use
a unique, incrementing `id` per request (1–99, wraps) and **confirm the echoed
`id`/`set` match** before trusting `values`. Symptom of the bug: a version/config
register that should be constant comes back different on each read. geopilot's
`internal/awl` client does this correctly (rotating id/set with reply matching).

## Key diagnostic registers (hand-verified)

Curated from decoding the UI handlers; more reliable than the auto-generated
`registers.md` for these specific rows. Temps/flows are signed, ÷10; `−9999`
(`55537`) = no-sensor NA.

| Reg | Meaning | Decode |
|---|---|---|
| 25 | current/last fault code | low 15 bits = E-code; bit15 = lockout |
| 26 / 27 / 28 | last lockout code / outputs / inputs at lockout | see `faults.md` |
| 30 | status bitfield | bit2 = cooling (Heat of Rejection), bit4 = heating |
| 19 / 20 | **FP1 / FP2** freeze sensor temp | °F ÷10 |
| 1111 / 1110 | Entering / Leaving Water temp | °F ÷10 |
| 1117 / 403 | Water flow / flow-meter-present | gpm ÷10 |
| 402 | loop fluid **and heat-calc factor** | 485 = antifreeze (also the ×gpm×ΔT constant), 500 = water |
| 1156/1157, 1154/1155 | **Heat of Rejection / Extraction** | 32-bit BTU/hr hi·lo pair; `Q = reg402 × flow(gpm) × loopΔT(°F)`. Rejection in cooling, Extraction in heating (same UI field, relabeled by mode). Verified: 485×8.5×4.4 = 18,139 = reg 1156/1157 |
| 400 / 401 | **DHW enabled / DHW setpoint** | 1=on ; °F ÷10 (130 °F) |
| 321 / 322 / 323 / 325 | VS pump Min / Max / command / **output %** | %, 32767 cmd = auto |
| 419 | loop-pressure trip | 0 = feature off / not equipped |
| 3000 / 3001 | compressor speed desired / actual | of 12 |
| 565 | blower speed code | 1–6 → 25/40/55/70/85/100 % |
| 16 | line voltage | V |
| 1146/1147, 1148/1149, 1164/1165, 1152/1153 | compressor / blower / pump / **total** power | W, hi·lo pair |
| 601–699 | fault-history counts | reg `600+n` = count of E`n` |
| 4 / 33 | DIP override / DIP switches | reg4=32767 → "manual"; see FP1 caveat in [`../diagnosis.md`](../diagnosis.md) |

## Thermostat / IntelliZone-2 zones (verified live — 2 zones)

Single-thermostat regs 501 (Room Setpoint) / 502 (Room Temp) read 0 when IZ2
zoning is active; the live data is per-zone in the IZ2 block (decoder `a.Xe`):

| Reg | Meaning |
|---|---|
| 31101 | zone count = bits 8–10 |
| `31007 + 3z` | zone *z* current temp, °F ÷10 |
| `31008/31009 + 3z` | zone *z* 32-bit data: `heatSP = bits11–16 + 36 °F`, `coolSP = bits17–22 + 36 °F`; also call/damper/priority bits |
| `31200 + 3z` | zone *z* config word (size, mode) |

Live: Zone 1 = 74.0 °F (heat 70 / cool 76), Zone 2 = 72.0 °F (heat 68 / cool 72).

## Writes & control (`putregs`, needs auth)

`putregs` writes are real — the UI itself uses them. Confirmed pattern (from
`k.Q(reg, val)` / `cmd=putregs&...&regs=<reg,val;>`):

| Action | Write |
|---|---|
| **Clear fault history** (reset all 601–699 counters + last-lockout) | `reg 47 = 21845` (0x5555) |
| Zone heat/cool setpoints, IZ2 settings, thermostat override, HA outputs | writable via the dealer/setup `m.U` pages |

> Writes change live furnace/zone behavior. `reg 47 = 21845` also wipes the E5
> freeze record — **snapshot the fault history first.** Validate any write target
> against the decoded UI before sending; there is no confirmation/undo at this layer.

## Derived field — Heat of Rejection / Heat of Extraction

The UI's "Heat of Rejection" (cooling) / "Heat of Extraction" (heating) field is a
single slot on the Performance Monitor page that relabels by mode. It is the rate
of heat crossing between the refrigerant and the ground loop, in **BTU/hr**,
computed by the ABC from the loop measurements:

```
Q (BTU/hr) = fluidFactor × loopFlow(gpm) × loopΔT(°F)
```

- **fluidFactor** = register **402**: `485` for antifreeze, `500` for water. (Reg
  402 is both the fluid label and the literal multiplier — antifreeze is a bit
  less dense / lower specific heat than water, hence 485 vs 500.)
- **loopΔT** = |leaving − entering water| = |reg 1110 − reg 1111| ÷ 10.
- **Cooling** (reg 30 bit 2): heat is *rejected* into the loop; value in regs
  **1156/1157**, labeled "Heat of Rejection".
- **Heating** (reg 30 bit 4): heat is *extracted* from the loop; value in regs
  **1154/1155**, labeled "Heat of Extraction".
- The register pair is a signed 32-bit value: `Q = 65536 × hi + lo` (`c.Jb()`),
  i.e. `hi` = reg 1156/1154, `lo` = reg 1157/1155.

**Verified live (cooling):** flow 8.5 gpm, ΔT 4.4 °F, antifreeze →
`485 × 8.5 × 4.4 = 18,139 BTU/hr`, matching reg 1156/1157 exactly.

Diagnostic use: rearranged, `loopΔT = Q / (fluidFactor × flow)`. In heating, for a
given extraction rate `Q`, lower flow forces a larger ΔT → colder water leaving the
coax → toward the FP1 freeze trip (E5). So "Heat of Extraction" on a high-load
winter run quantifies loop stress directly. See [`../diagnosis.md`](../diagnosis.md).

## Predefined sets (firmware literals)

| Set | Registers |
|---|---|
| `abcinfo` | `2;8;88;89;90;91;92;93;42` |
| `devices2` | `800;803;806;807;808;809;812;815;818;821;824` |
| `devices3` | `800;803;806;807;808;809;812;815;818;821;824;830;833` |
| `devices` | same as `devices2` |

`abcinfo` registers 88–93 are the ABC model string packed two ASCII chars per
register (`16706` = `0x4142` = "AB", …).

## Value encoding

- 16-bit registers. Temperatures and pressures are **signed** — value ≥ 32768
  means `value − 65536`.
- **`−9999` (returns as `55537`) is the "no sensor / NA" sentinel.** The UI
  prints `NA`; don't treat it as a real reading.
- Most temperatures and flows are scaled **÷10** for display (683 → 68.3 °F,
  84 → 8.4 gpm). Energy/power values are often **hi/lo register pairs** combined
  into one number (see the `c.za`/`c.Jb` helpers in `indexc.js`).
- Some registers are **bitfields** (e.g. reg 30 status bits) or **enum codes**
  (e.g. reg 402 == 485 → "Antifreeze", else "Water"). See `registers.md` for
  which label a register feeds, and `indexc.js` for the exact per-row scaling.

## Live-validated sample (cooling, this unit)

`getregs` of the Performance Monitor list `30;502;1110,5;1117;1154,4;402,2;567;1119`:

| Reg | Raw | Decoded | Label |
|---|---|---|---|
| 1111 | 683 | 68.3 °F | Entering Water Temp |
| 1110 | 727 | 72.7 °F | Leaving Water Temp |
| 1117 | 84 | 8.4 gpm | Water Flow |
| 402 | 485 | Antifreeze | loop fluid type |
| 30 | 0x424D | bit2 set → Heat of Rejection | status bitfield (cooling) |
| 1119 | −9999 | NA | Loop Pressure (no sensor) |

## Reusable snapshot

geopilot's collector authenticates with the passcode and polls these registers on
tiered intervals. See [`registers.md`](registers.md) for the full recovered map
and [`dealer-registers.md`](dealer-registers.md) for the dealer / IZ2 pages.

## Fault codes

The controller's own fault dictionaries are recovered in **`faults.md`**:

- **60 Aurora E-codes** (E1–E99) — `a.Sa()`. The active/last code is register **25**,
  low 15 bits; bit 15 (`0x8000`) set = hard lockout. Live flags are regs **710–717**.
- **VS-drive alarms / derates / safe-modes** (`AL-`/`DR-`/`SM-`) — `a.ke()`, unpacked
  from drive bitfield registers.
- **IZ2 zone-panel faults** + the fault-history format (regs **31300–31328**:
  10 entries of code + packed date/time).

## Caveats on the map

- Recovered by static extraction from the minified UI. `registers.md` now merges
  **both** UIs — `src=std` (homeowner `indexc.js`) and `src=dealer`
  (`indexdmc.js`). Labels marked `inferred` were reached through one resolved local
  variable — good evidence, worth a glance at the live value before relying on them;
  `direct` labels are certain.
- **Dealer / IZ2 registers** are catalogued by page in **`dealer-registers.md`**
  (33 pages, 289 registers): Equipment Status, Zone Status/Config, Fault Status,
  Accessories run-hours, Temp Sensors & Offsets, Dealer Info, etc. The dealer UI
  packs many values as register bitfields, so the reliable unit there is the
  page→register-range grouping, not always a 1:1 per-row register.
- `31xxx` = IntelliZone-2 primary registers; `12xxx` = the AXB/legacy variant of
  the same pages (selected by the `b.d.g` flag in the UI).
