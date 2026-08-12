# Aurora register map (recovered from indexc.js + indexdmc.js)

125 registers labelled across both UIs. **direct** = register literal sits in the display call; **inferred** = reached via one resolved local. `std` = homeowner UI, `dealer` = dealer/IZ2 UI.

**Cross-checked 2026-08-12** against the independent
[`ccutrer/waterfurnace_aurora`](https://github.com/ccutrer/waterfurnace_aurora) Modbus
map (derived from the AID/Modbus side, vs. ours from the web UI). Decode conventions
and every diagnostic-critical register agree. Corrections applied from that pass:
reg **20** = FP2 / Air Coil (not EWT), reg **1111** = Entering Water (not Entering
Air), reg **402** = Brine Type (the "Off Time Amount" label was a bad auto-extraction),
reg **3808** = EEV2 % Open (not SuperHeat), and reg **3906** SuperHeat unit is °F (not
%). A live idle read (3808 = 75, 3906 = 11.3) supports 3808 = EEV valve position and
3906 = superheat in °F; both are worth re-confirming under compressor load.

> ⚠️ **Refrigerant thermistors are NOT populated on this unit (NVV048).** Verified
> live 2026-08-12 against stored history: **reg 3325 "Discharge Temperature"** sat
> flat at 93–99 °F and barely moved (96→98 °F) across compressor stages 2→6 — a real
> hot-gas discharge line climbs hard with load, so the sensor isn't tracking real
> discharge temperature on this unit (the label is correct — cross-confirmed by
> ccutrer/waterfurnace_aurora); **reg 1125 "SuperHeat" reads 0** at all times; **reg 1113 "Suction
> Temperature"** returns the NA sentinel (55537). Treat 3325 / 1113 / 1125 as
> **unsensored / defaulting — do not use them** for superheat or desuperheater
> analysis. The trustworthy load signal is compressor stage (reg 3001) + compressor
> power (1146/1147) + HWG water temp (1114).
>
> **Desuperheater / E15 finding:** on low-load cooling days the compressor idles at
> 2/12, producing no discharge superheat, so the HWG cannot add heat. The K6 pump
> (reg 1104 bit0) still cycles and pulls ~130 °F element-heated water out of the
> buffer tank into the aquastat → logs **E15 "Hot Water Limit"** (a benign
> non-lockout nuisance trip, not a real fault, and *not* evidence of harvesting).
> geopilot's dashboard therefore gates the "Harvesting" label on compressor stage
> (`dhwHarvestStage`, model.go), not on the pump bit. Real harvest is expected only
> at elevated compressor stages (high-load cooling hours, or winter high-stage
> heating). See [`../diagnosis.md`](../diagnosis.md) and [`faults.md`](faults.md).

| Reg | Label(s) | Unit | Conf | Src | Pages |
|----:|----------|------|------|-----|-------|
| 1 | Fault / Mode / Random Start Delay |  | direct | std | Demo Overview, Timers |
| 6 | Compressor ASC Delay / Mode |  | direct | std | Demo Overview, Timers |
| 9 | Minimum Run Time |  | direct | std | Timers |
| 15 | Blower Off Delay |  | direct | std | Timers |
| 16 | Line Voltage | V | direct | std | Energy Monitor |
| 17 | Aux/E Heat Stage |  | inferred | std | Timers |
| 19 | * Cooling LL Temperature / Cooling LL Temperature / FP1 | °C, °F | direct | std | Refrig Monitor, Sensor Inputs |
| 20 | FP2 / Air Coil Temperature | °C, °F | inferred | std | Sensor Inputs |
| 25 | Fault / Fault: / Mode |  | inferred | std | Demo Overview, Sensor Inputs |
| 26 | Last Lockout |  | inferred | std | Lockout Details |
| 30 | Actual Comp Speed / Aux Heat / Compressor |  | inferred | dealer | Equipment Status, Status |
| 33 | Active Outputs |  | inferred | std | Lockout Details |
| 35 | Actual Mode / Mode |  | inferred | dealer | Equipment Status, Status |
| 36 | ABC Board Rev |  | inferred | std | About Unit |
| 50 | Continuous Spd |  | inferred | std | Demo Overview |
| 51 | Low Capacity Spd |  | inferred | std | Demo Overview |
| 52 | High Capacity Spd |  | inferred | std | Demo Overview |
| 54 | Actual Fan Speed / Blower Spd Act / Fan / IZ2 Blower % Des |  | direct | dealer+std | Equipment Status, Status |
| 84 | SO Valve Delay |  | direct | std | Timers |
| 85 | Test Mode Timer |  | direct | std | Timers |
| 92 | Serial Number |  | inferred | dealer | Equipment |
| 105 | Model Number |  | inferred | dealer | Equipment |
| 110 |  / Reheat Delay |  | direct | std | Timers |
| 112 | Line Voltage Setting |  | direct | std | Line Voltage Setup, Sensor Kit Setup |
| 321 | VS Pump Min | % | direct | std | AXB Setup |
| 322 | VS Pump Max | % | direct | std | AXB Setup |
| 325 | VS Pump Output | % | direct | std | AXB Outputs |
| 326 |  / Fault Reason |  | inferred | std | System Faults |
| 330 | EEV Offset Position | % | direct | std | VS Config |
| 331 | EEV Start Position | % | direct | std | VS Config |
| 340 | Blower Only |  | direct | std | ECM Setup, VS Overview |
| 341 | Lo Compressor / Low Setting |  | direct | std | ECM Setup, VS Overview |
| 342 | Hi Compressor / High Setting |  | direct | std | ECM Setup, VS Overview |
| 346 | Adjustment / Clg % | % | inferred | std | Cooling Airflow Setup, VS Overview |
| 347 | Aux Heat / Aux/E Heat Spd / EH Setting |  | direct | std | Demo Overview, ECM Setup |
| 401 | DHW Setpoint | °C, °F | direct | std | AXB Setup |
| 402 | Brine Type / loop fluid (485=antifreeze, 500=water) |  | direct | std | AXB Setup |
| 404 | Blower Type |  | inferred | dealer | Equipment |
| 406 | Status |  | direct | std | Demo Overview, System Faults |
| 409 | Fault / Fault: |  | inferred | std | Demo Overview, Sensor Inputs |
| 411 | Fault / Fault: |  | inferred | std | Demo Overview, Sensor Inputs |
| 417 | Factor L |  | direct | std | Power Adjustment Factor Setup |
| 418 | Factor H |  | direct | std | Power Adjustment Factor Setup |
| 419 | Loop Pressure Trip | kPA, psi | direct | std | AXB Setup |
| 480 | Comp Type / Equipment Type / Max Zones |  | inferred | dealer | Equipment, Equipment Status |
| 501 | Room Setpoint | °C, °F | direct | std | VS Overview |
| 502 | Room Temp | °C, °F | direct | std | VS Overview |
| 564 | Comp. Spd Des |  | direct | std | VS Overview |
| 1105 | Blower | A | direct | std | Energy Monitor |
| 1106 | Aux | A | direct | std | Energy Monitor |
| 1107 | Compressor 1 | A | direct | std | Energy Monitor |
| 1108 | Compressor 2 | A | direct | std | Energy Monitor |
| 1109 | * Heating LL Temperature / Heating LL Temperature | °C, °F | direct | std | Refrig Monitor, VS Sensors and Ref |
| 1110 | Leaving Water Temp | °C, °F | inferred | std | Performance Monitor |
| 1111 | Entering Water Temp | °C, °F | inferred | std | Performance Monitor |
| 1112 | Leaving Air Temp | °C, °F | inferred | std | Performance Monitor |
| 1113 |   / Suction Temperature | °C, °F | inferred | std | Refrig Monitor |
| 1114 | Hot Water | °C, °F | inferred | std | AXB Inputs |
| 1115 | Discharge Pressure | kPA, psi | inferred | std | Refrig Monitor |
| 1116 | Suction Pressure / Vapor Inj. | kPA, psi | inferred | std | Refrig Monitor |
| 1117 | WaterFlow | gpm, lps | direct | std | AXB Setup, Performance Monitor |
| 1119 | Loop Pressure | kPA, psi | inferred | std | AXB Inputs, Performance Monitor |
| 1124 | Saturated Evaporator / Saturated Vapor Inj. | °C, °F | inferred | std | Refrig Monitor |
| 1125 | SuperHeat / Vapor Inj. SuperHeat | °C, °F | direct | std | Refrig Monitor |
| 1126 | Vapor Inj. Open % | % | inferred | std | Refrig Monitor |
| 1134 | Saturated Condenser | °C, °F | direct | std | Refrig Monitor, VS Sensors and Ref |
| 1135 | SubCooling | °C, °F | direct | std | Refrig Monitor, VS Sensors and Ref |
| 1136 | SubCooling | °C, °F | direct | std | Refrig Monitor, VS Sensors and Ref |
| 1146 | Compressor | W | direct | std | Energy Monitor |
| 1147 | Compressor | W | direct | std | Energy Monitor |
| 1148 | Blower | W | direct | std | Energy Monitor |
| 1149 | Blower | W | direct | std | Energy Monitor |
| 1150 | Aux | W | direct | std | Energy Monitor |
| 1151 | Aux | W | direct | std | Energy Monitor |
| 1152 | Total | W | direct | std | Energy Monitor |
| 1153 | Total | W | direct | std | Energy Monitor |
| 1154 | Heat of Extraction / Heat of Rejection | Btuh, Wh | direct | std | Performance Monitor |
| 1155 | Heat of Extraction / Heat of Rejection | Btuh, Wh | direct | std | Performance Monitor |
| 1156 | Heat of Extraction / Heat of Rejection | Btuh, Wh | direct | std | Performance Monitor |
| 1157 | Heat of Extraction / Heat of Rejection | Btuh, Wh | direct | std | Performance Monitor |
| 1164 | FC1 / FC1_GLNP / FC2 / FC2_GLNP / Open Loop / Other / Pump / VS / VS + 26-99 / VS + UPS26-99 | W | direct | std | Energy Monitor |
| 1165 | FC1 / FC1_GLNP / FC2 / FC2_GLNP / Open Loop / Other / Pump / VS / VS + 26-99 / VS + UPS26-99 | W | direct | std | Energy Monitor |
| 3001 | Comp. Spd Act |  | inferred | std | VS Overview |
| 3027 | Actual Comp Speed / Compressor |  | inferred | dealer | Equipment Status, Status |
| 3220 | General |  | direct | std | VS Drive Details |
| 3221 | General |  | direct | std | VS Drive Details |
| 3222 | Derate |  | direct | std | VS Drive Details |
| 3223 | Derate |  | direct | std | VS Drive Details |
| 3224 | Safemode |  | direct | std | VS Drive Details |
| 3225 | Safemode |  | direct | std | VS Drive Details |
| 3226 | Alarm |  | direct | std | VS Drive Details |
| 3227 | Alarm |  | direct | std | VS Drive Details |
| 3322 | Discharge Pressure | kPA, psi | inferred | std | VS Sensors and Ref |
| 3323 | Suction Pressure | kPA, psi | inferred | std | VS Sensors and Ref |
| 3325 | Discharge Temperature | °C, °F | inferred | std | VS Sensors and Ref |
| 3326 | Compressor Ambient | °C, °F | inferred | std | VS Sensors and Ref |
| 3327 | Drive Temperature | °C, °F | direct | std | VS Service |
| 3330 | VS Drive EWT | °C, °F | inferred | std | VS Sensors and Ref |
| 3331 | Line Voltage | V | direct | std | VS Service |
| 3332 | Thermo Power | % | direct | std | VS Service |
| 3422 | Compressor Power | W | direct | std | VS Service |
| 3423 | Compressor Power | W | direct | std | VS Service |
| 3424 | Drive Supply Voltage | V | direct | std | VS Service |
| 3425 | Drive Supply Voltage | V | direct | std | VS Service |
| 3522 | Inverter Temperature | °C, °F | direct | std | VS Service |
| 3523 | UDC Voltage | V | direct | std | VS Service |
| 3524 | Drive Fan Speed | % | direct | std | VS Service |
| 3804 | EEV2 Ctl |  | direct | std | VS Drive Details |
| 3808 | EEV2 % Open | % | direct | std | VS Sensors and Ref |
| 3903 | Suction Temperature | °C, °F | inferred | std | VS Sensors and Ref |
| 3904 | Leaving Air Temp | °C, °F | inferred | std | Performance Monitor |
| 3905 | Saturated Evaporator | °C, °F | direct | std | VS Sensors and Ref |
| 3906 | SuperHeat | °C, °F | direct | std | VS Sensors and Ref |
| 12002 | Room Temp |  | inferred | dealer | Status |
| 12004 | Outdoor Temp / Remote Temp |  | inferred | dealer | Status |
| 12301 | Auto / Manual / Auto Changeover Time / Aux Heat Lockout / Compressor Satisfy / Cooling Lockout / Cycles per Hour / Fan With Heat / Smart Recovery / TStat Type |  | inferred | dealer | TStat Configuration |
| 12307 | 1st Stage / 2nd Stage / Aux Heat |  | inferred | dealer | Select Differentials |
| 31002 | Dehumid |  | inferred | dealer | Equipment Status |
| 31004 | Zone Call Comp Speed / Zone Call Fan % / Zone Call Mode |  | inferred | dealer | Equipment Status |
| 31005 | Fan Demand / Unit Demand |  | inferred | dealer | Equipment Status |
| 31008 | Zone Status |  | inferred | dealer | Xe |
| 31009 | Zone Status |  | inferred | dealer | Xe |
| 31025 | Dehumid / Zone Call Comp Speed |  | inferred | dealer | Equipment Status |
| 31101 | Aux Heat Lockout / Cool Staging / Damper / Fan With Heat / Heat Staging / Number Of Zones / Number of Zones / Thermostat Type |  | inferred | dealer | Equipment Status, IZ2 Configuration |
| 31110 | Dehumid |  | inferred | dealer | Equipment Status |
