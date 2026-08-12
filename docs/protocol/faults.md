# Aurora fault reference (recovered from the controller UI)

## How Aurora faults are stored (verified live)

| Register(s) | Holds | Notes |
|---|---|---|
| **25** | current / last fault code | low 15 bits = E-code; bit 15 (`0x8000`) = hard lockout |
| **26** | **last lockout** code | low 15 bits = E-code; bit 15 set = valid |
| **27** | outputs active at last lockout | bitfield: 1=CC 2=CC2 4=O 8=G 16=EH1 32=EH2 512=AR 1024=LO 2048=RH/AL |
| **28** | inputs active at last lockout | bitfield: 1=Y1 2=Y2 4=W 8=O 16=G 32=DH 64=ES 512=LS |
| **601–699** | **per-code lifetime counts** | reg `600+n` = times E`n` has occurred; `−1` = never; `999` = counter ceiling |

The fault history is **occurrence counts, not timestamps** — the Aurora does not
date its faults. Only the IZ2 zone block (below) carries real date/time.

Reading gotcha: `request.cgi` returns stale/misaligned data if you reuse the
`id`/`set` values across calls — use a unique incrementing id per request and
confirm the echoed `id`/`set` match before trusting `values`.

### This unit's live counts (2026-08-11)

`E5`=646 (Frze Detect FP1, **last lockout**), `E15`=≥999 (Hot Water Limit, benign
desuperheater limit), `E55`=≥999 (Out of Envelope), `E99`=55 (System Reset),
`E19`=6 (Crit Comm Error), `E59`=1 (Int Drive Error). See [`../diagnosis.md`](../diagnosis.md).

**Update 2026-08-11/12:** all history counters cleared via `reg 47 = 21845`
(snapshot in `fault_history_pre_clear_2026-08-11_221432.txt`). E15 immediately
re-accumulated. Root cause confirmed live: on low-load cooling days the compressor
idles at 2/12 with **no discharge superheat** (regs 3325/1113/1125 unpopulated on
this unit — see `registers.md`), so the HWG pump only circulates ~130 °F
element-heated buffer-tank water into the aquastat and trips E15. It is a nuisance
non-lockout, **not** a sign the desuperheater is harvesting. Reg 25 shows E15 as the
last event but reg 26 (last *lockout*) stays E5, confirming E15 never locks out.

## Main fault / lockout codes (E-codes)

Shown by the controller as `E<n>`. Decoder: `a.Sa()` in indexc.js. The active/last code is register **25** (low 15 bits); fault history is the 601–699 count block above.

| Code | Meaning |
|---|---|
| E1 | Input Error |
| E2 | High Pressure |
| E3 | Low Pressure |
| E4 | Frze Detect FP2 |
| E5 | Frze Detect FP1 |
| E7 | Condensate |
| E8 | O/U Voltage |
| E9 | Airflow/RPM |
| E10 | Comp Monitor |
| E11 | FP1/2 Snr Err |
| E12 | RefrigPerformnc |
| E13 | NCritAxbSnrErr |
| E14 | CritAxbSnrErr |
| E15 | Hot Water Limit |
| E16 | VS Pump Error |
| E18 | NCritComm Error |
| E19 | Crit Comm Error |
| E22 | Comm ECM Error |
| E25 | AXB EEV Error |
| E26 | EntSrcLowLimAlm |
| E27 | EntSrcHiLimAlm |
| E28 | LvgSrcLowLimAlm |
| E29 | LvgSrcHiLimAlm |
| E31 | Src Flow Fault |
| E32 | Load Flow Fault |
| E41 | Hi Drive Temp |
| E42 | Hi Dischrg Temp |
| E43 | Lo Suct Press |
| E44 | Lo Cond Press |
| E45 | Hi Cond Press |
| E46 | Output Pwr Lmt |
| E47 | EEV ID Comm Err |
| E48 | EEV OD Comm Err |
| E49 | Cabinet Tmp Snr |
| E51 | Dischrg Tmp Snr |
| E52 | Suct Press Snr |
| E53 | Cond Press Snr |
| E54 | Lo Supply Volt |
| E55 | Out of Envelope |
| E56 | Drv Over Current |
| E57 | Drive O/U Volt |
| E58 | Hi Drive Temp |
| E59 | Int Drive Error |
| E61 | Multiple SafeMd |
| E62 | Low Temp |
| E63 | Fault Limit |
| E64 | EVC Intrnl Flt |
| E65 | Soft Start Fail |
| E66 | Power Fault |
| E67 | EVC Intrnl Flt |
| E69 | Invalid Comp Id |
| E71 | Loss of Charge |
| E72 | Suct Temp Snr |
| E73 | LvgAir Temp Snr |
| E74 | Max Op Pressure |
| E75 | Loss of Charge |
| E76 | Suct Temp Snr |
| E77 | LvgAir Temp Snr |
| E78 | Max Op Pressure |
| E99 | System Reset |

> Codes 23/24 (AXB sensor) expand to sub-codes via `d.Yc()`. Duplicate meanings at 71–78 are the two compressor circuits.

## VS-drive alarms / derates / safe-modes

Decoder `a.ke()` in indexc.js unpacks these from drive bitfield registers. Prefixes: `AL-` alarm, `DR-` derate, `SM-` safe-mode, `EEV2-` vapor-injection EEV.

`AOC Comm Lost`, `VSD Comm Lost`, `AL-Multi Safe Modes`, `AL-Out of Envelope`, `AL-Over Current`, `AL-Over Voltage`, `AL-Drive Over Temp`, `AL-Under Voltage`, `AL-High Discharge Tmp`, `AL-Inv Discharge Tmp`, `AL-OEM Comm Timeout`, `AL-MOC Safety`, `AL-DC Under Voltage`, `AL-Inv Suction Press`, `AL-Inv Disch Press`, `AL-Low Disch Press`, `AL-Internal Error`, `SM-EEV Indoor Failed`, `SM-EEV Outdoor Failed`, `SM-Inv Ambient Temp`, `DR-Drive Over Temp`, `DR-Low Suct Press`, `DR-Low Disch Press`, `DR-High Disch Press`, `DR-Output Power Limit`

## IZ2 zone-panel fault codes

Decoder `a.Gd()` in indexdmc.js. Stored in registers **31300–31328** (alt **12200–12228**): 10 entries, each = fault code + a 32-bit packed date/time (bits 5-10 sec, 11-15 hour, 16-20 day, 21-24 month).

| Code | Meaning |
|---|---|
| 1 | Humidity Mod Temp Sensor Open |
| 2 | Humidity Mod Temp Sensor Shorted |
| 3 | Outdoor Temp Sensor Error |
| 4 | Humidity Reading Too Low |
| 5 | Humidity Reading Too High |
| 6 | Humidity Sensor Failure |
| 7 | Comm Error Outdoor |
| 8 | Remote Room Sensor Error |
| 9 | Prim Temp Sensor Open |
| 10 | Prim Temp Sensor Shorted |
| 11 | Temp Reading Too Low |
| 12 | Temp Reading Too High |
| 13 | Comm Error Indoor Unit |
| 14 | Low Voltage Below 19VAC |
| 15 | Low Voltage Below 16VAC |
| 97 | Comm Error Zone Panel |
| 98 | Comm Error Zone 2 |
| 99 | Comm Error Zone 3 |
| 100 | Comm Error Zone 4 |
| 101 | Comm Error Zone 5 |
| 102 | Comm Error Zone 6 |
| 103 | Comm Error AUX Board |

