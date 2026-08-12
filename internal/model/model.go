// Package model turns raw furnace registers into the decoded dashboard model.
// The decodes mirror the controller's own web UI (reverse-engineered) and are
// documented in regmap/REQUEST_CGI.md, FAULTS.md and DIAGNOSIS.md.
package model

import (
	"fmt"
	"math"

	"github.com/daniel-widrick/geopilot/internal/live"
)

type Zone struct {
	N       int     `json:"n"`
	Temp    float64 `json:"temp"`
	HeatSP  int     `json:"heat_sp"`
	CoolSP  int     `json:"cool_sp"`
	Calling bool    `json:"calling"`
}

type FaultCount struct {
	Code   int    `json:"code"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
	Capped bool   `json:"capped"`
}

type Dashboard struct {
	Time string `json:"time"`
	OK   bool   `json:"ok"` // did we have live data?

	Mode         string `json:"mode"`
	Status       string `json:"status"`
	FaultCode    string `json:"fault_code"`
	FaultName    string `json:"fault_name"`
	Lockout      bool   `json:"lockout"`
	LastLockCode string `json:"last_lock_code"`
	LastLockName string `json:"last_lock_name"`

	// water loop
	EWT      *float64 `json:"ewt"`
	LWT      *float64 `json:"lwt"`
	LoopDT   *float64 `json:"loop_dt"`
	FlowGPM  *float64 `json:"flow_gpm"`
	Fluid    string   `json:"fluid"`
	FP1      *float64 `json:"fp1"`
	FP2      *float64 `json:"fp2"`
	HeatRate *int64   `json:"heat_rate_btuh"` // rejection (cooling) or extraction (heating)
	HeatKind string   `json:"heat_kind"`

	// operation
	CompActual  int      `json:"comp_actual"`
	CompDesired int      `json:"comp_desired"`
	BlowerPct   int      `json:"blower_pct"`
	PumpPct     *int64   `json:"pump_pct"`
	PumpMin     *int64   `json:"pump_min"`
	PumpMax     *int64   `json:"pump_max"`
	LineV       *int64   `json:"line_v"`
	CabinetTemp *float64 `json:"cabinet_temp"` // VS-drive "Compressor Ambient" (reg 3326) — cabinet air, not outdoor

	// power
	CompW   *int64 `json:"comp_w"`
	BlowerW *int64 `json:"blower_w"`
	PumpW   *int64 `json:"pump_w"`
	AuxW    *int64 `json:"aux_w"`
	TotalW  *int64 `json:"total_w"`

	// domestic hot water / desuperheater
	DHWEnabled   bool     `json:"dhw_enabled"`    // feature configured on
	DHWPumpOn    bool     `json:"dhw_pump_on"`    // K6 circulating right now
	DHWWaterTemp *float64 `json:"dhw_water_temp"` // HWG water temp (reg 1114)
	DHWSetpt     *float64 `json:"dhw_setpoint"`
	DHWState     string   `json:"dhw_state"` // Off | Idle | Circulating | Harvesting
	DHWClass     string   `json:"dhw_class"` // standby | muted | good

	Zones  []Zone       `json:"zones"`
	Faults []FaultCount `json:"faults"`

	// outdoor (weather source)
	Outdoor   *float64 `json:"outdoor"`
	FeelsLike *float64 `json:"feels_like"`
	Humidity  *float64 `json:"humidity"`
	WindMph   *float64 `json:"wind_mph"`
	WxText    string   `json:"wx_text"`

	// cost (rate from Bayou bills)
	RateUSDPerKWh float64  `json:"rate_usd_per_kwh"`
	CostPerHour   *float64 `json:"cost_per_hour"` // geo power × rate, right now
	CostToday     *float64 `json:"cost_today"`    // set by the server (needs DB integral)

	// "should the windows be open?" advisor
	Dewpoint      *float64 `json:"dewpoint"`
	IndoorAvg     *float64 `json:"indoor_avg"`
	WindowsOpen   string   `json:"windows_open"` // "Open" / "Keep closed" / "—"
	WindowsReason string   `json:"windows_reason"`
	WindowsClass  string   `json:"windows_class"` // good | muted | warn
}

// dewpointF returns the dew point (°F) from air temp (°F) and relative humidity (%).
func dewpointF(tf, rh float64) float64 {
	if rh <= 0 {
		rh = 1
	}
	tc := (tf - 32) * 5.0 / 9.0
	const a, b = 17.625, 243.04
	alpha := math.Log(rh/100) + a*tc/(b+tc)
	tdC := b * alpha / (a - alpha)
	return tdC*9.0/5.0 + 32.0
}

// dhwHarvestStage is the compressor stage (of 12) at or above which the VS
// compressor produces enough discharge superheat for the desuperheater to add real
// heat. Below it (the 2–3/12 idle band this unit sits in on low-load cooling days)
// a running HWG pump merely circulates tank water. Set from the live finding that
// discharge ≈ water with zero superheat at 2/12.
const dhwHarvestStage = 4

func sgn(v int64) int64 {
	if v >= 32768 {
		return v - 65536
	}
	return v
}

func Build(s *live.Snapshot, now string, rate float64) Dashboard {
	d := Dashboard{Time: now, Fluid: "—", Mode: "—", Status: "—", RateUSDPerKWh: rate}

	get := func(reg int) (int64, bool) { return s.Furnace(reg) }
	// signed ÷10 helper returning a pointer (nil if missing / NA sentinel)
	temp := func(reg int) *float64 {
		v, ok := get(reg)
		if !ok {
			return nil
		}
		sv := sgn(v)
		if sv <= -9999 || sv >= 9999 {
			return nil
		}
		f := float64(sv) / 10.0
		return &f
	}
	i64 := func(reg int) *int64 {
		v, ok := get(reg)
		if !ok {
			return nil
		}
		return &v
	}
	pair := func(hi, lo int) *int64 {
		h, ok1 := get(hi)
		l, ok2 := get(lo)
		if !ok1 || !ok2 {
			return nil
		}
		v := 65536*h + l
		return &v
	}

	if r30, ok := get(30); ok {
		d.OK = true
		switch {
		case r30&4 != 0:
			d.Mode = "Cooling"
		case r30&16 != 0:
			d.Mode = "Heating"
		default:
			d.Mode = "Standby"
		}
	}

	// fault / status
	if r25, ok := get(25); ok {
		code := int(r25 & 0x7FFF)
		d.Lockout = r25&0x8000 != 0
		if code != 0 {
			d.FaultCode = fmt.Sprintf("E%d", code)
			d.FaultName = ECode(code)
		} else {
			d.FaultCode = "None"
		}
	}
	activeFault := false
	for r := 710; r <= 717; r++ {
		if v, ok := get(r); ok && v != 0 {
			activeFault = true
		}
	}
	if d.Lockout || activeFault {
		d.Status = "Lockout"
	} else if d.OK {
		d.Status = "Normal"
	}
	if r26, ok := get(26); ok {
		c := int(r26 & 0x7FFF)
		if c != 0 {
			d.LastLockCode = fmt.Sprintf("E%d", c)
			d.LastLockName = ECode(c)
		}
	}

	// water loop
	d.EWT = temp(1111)
	d.LWT = temp(1110)
	if d.EWT != nil && d.LWT != nil {
		dt := *d.LWT - *d.EWT
		d.LoopDT = &dt
	}
	if v, ok := get(1117); ok {
		f := float64(sgn(v)) / 10.0
		d.FlowGPM = &f
	}
	if v, ok := get(402); ok {
		if v == 485 {
			d.Fluid = "Antifreeze"
		} else {
			d.Fluid = "Water"
		}
	}
	d.FP1 = temp(19)
	d.FP2 = temp(20)
	if d.Mode == "Heating" {
		d.HeatRate = pair(1154, 1155)
		d.HeatKind = "extraction"
	} else {
		d.HeatRate = pair(1156, 1157)
		d.HeatKind = "rejection"
	}

	// operation
	if v, ok := get(3001); ok {
		d.CompActual = int(v)
	}
	if v, ok := get(3000); ok {
		d.CompDesired = int(v)
	}
	if v, ok := get(565); ok {
		d.BlowerPct = blowerPct(int(v))
	}
	d.PumpPct = i64(325)
	d.PumpMin = i64(321)
	d.PumpMax = i64(322)
	d.LineV = i64(16)
	d.CabinetTemp = temp(3326)

	// power (hi/lo pairs)
	d.CompW = pair(1146, 1147)
	d.BlowerW = pair(1148, 1149)
	d.AuxW = pair(1150, 1151)
	d.PumpW = pair(1164, 1165)
	d.TotalW = pair(1152, 1153)

	// DHW / desuperheater: enabled (400) vs actually heating (K6 pump = reg 1104 bit0)
	if v, ok := get(400); ok {
		d.DHWEnabled = v == 1
	}
	if v, ok := get(1104); ok {
		d.DHWPumpOn = v&1 != 0
	}
	d.DHWWaterTemp = temp(1114)
	d.DHWSetpt = temp(401)
	// Honest desuperheater state. The pump bit (1104) alone lies: on a low-load
	// cooling day the pump cycles and pulls already-hot, element-heated water out of
	// the buffer tank without the geo adding anything — it just trips E15 (Hot Water
	// Limit). A real harvest needs the compressor loaded enough to throw off discharge
	// superheat; at minimum speed there is none to give (verified live: at 2/12 the
	// discharge temp sat at/below the HWG water temp, superheat channel flat 0). We
	// gate on compressor stage rather than the discharge thermistor because this
	// unit's refrigerant thermistors (regs 3325/1113/1125) are not populated — see
	// regmap/REGISTERS.md.
	d.DHWState, d.DHWClass = "Off", "standby"
	if d.DHWEnabled {
		switch {
		case !d.DHWPumpOn:
			d.DHWState, d.DHWClass = "Idle", "standby"
		case d.CompActual >= dhwHarvestStage:
			d.DHWState, d.DHWClass = "Harvesting", "good"
		default:
			// pump running but compressor idling — circulating, not harvesting
			d.DHWState, d.DHWClass = "Circulating", "muted"
		}
	}

	// zones
	if zc, ok := get(31101); ok {
		nz := int((zc >> 8) & 7)
		for z := 0; z < nz && z < 6; z++ {
			base := 31007 + 3*z
			tv, ok1 := get(base)
			hi, ok2 := get(base + 1)
			lo, ok3 := get(base + 2)
			if !ok1 || !ok2 || !ok3 {
				continue
			}
			data := (hi << 16) | lo
			d.Zones = append(d.Zones, Zone{
				N:      z + 1,
				Temp:   float64(sgn(tv)) / 10.0,
				HeatSP: int((data>>11)&0x3F) + 36,
				CoolSP: int((data>>17)&0x3F) + 36,
			})
		}
	}

	// outdoor conditions (weather source, stored ×100 -> scale 0.01)
	wx := func(key string) *float64 {
		v, ok := s.Get(key)
		if !ok {
			return nil
		}
		f := float64(v.Raw) * 0.01
		return &f
	}
	d.Outdoor = wx("weather:temperature_2m@1")
	d.FeelsLike = wx("weather:apparent_temperature@1")
	d.Humidity = wx("weather:relative_humidity_2m@1")
	d.WindMph = wx("weather:wind_speed_10m@1")
	if code, ok := s.Get("weather:weather_code@1"); ok {
		d.WxText = wmoText(int(code.Raw / 100))
	}

	// fault history counts (regs 601..699 = E1..E99)
	for code := 1; code <= 99; code++ {
		v, ok := get(600 + code)
		if !ok {
			continue
		}
		if sgn(v) > 0 {
			d.Faults = append(d.Faults, FaultCount{
				Code: code, Name: ECode(code), Count: v, Capped: v >= 999,
			})
		}
	}

	// cost right now = geo power (kW) × effective rate ($/kWh)
	if d.TotalW != nil && rate > 0 {
		c := float64(*d.TotalW) / 1000.0 * rate
		d.CostPerHour = &c
	}

	// dewpoint + "should the windows be open?"
	if d.Outdoor != nil && d.Humidity != nil {
		dp := dewpointF(*d.Outdoor, *d.Humidity)
		d.Dewpoint = &dp
	}
	if len(d.Zones) > 0 {
		var sum float64
		for _, z := range d.Zones {
			sum += z.Temp
		}
		avg := sum / float64(len(d.Zones))
		d.IndoorAvg = &avg
	}
	d.WindowsOpen, d.WindowsReason, d.WindowsClass = windowsAdvice(&d)

	return d
}

// windowsAdvice decides whether opening the windows would help — free "night-flush"
// cooling when it's cooler AND drier outside, but not when it would trade heat for
// humidity (or when heating).
func windowsAdvice(d *Dashboard) (verdict, reason, class string) {
	if d.Outdoor == nil || d.IndoorAvg == nil {
		return "—", "waiting for indoor/outdoor temps", "muted"
	}
	out, in := *d.Outdoor, *d.IndoorAvg
	if d.Mode == "Heating" {
		return "Keep closed", fmt.Sprintf("heating — %.0f°F out would just bleed heat", out), "muted"
	}
	gap := in - out // positive = cooler outside
	if gap < 2 {
		return "Keep closed", fmt.Sprintf("outside (%.0f°F) isn't meaningfully cooler than inside (%.0f°F)", out, in), "muted"
	}
	if d.Dewpoint != nil && *d.Dewpoint > 62 {
		return "Keep closed", fmt.Sprintf("cooler out (%.0f°F) but muggy — dew point %.0f°F would bring humidity in", out, *d.Dewpoint), "warn"
	}
	dp := ""
	if d.Dewpoint != nil {
		dp = fmt.Sprintf(", dew point %.0f°F", *d.Dewpoint)
	}
	return "Open", fmt.Sprintf("%.0f°F and dry%s vs %.0f°F inside — free cooling available", out, dp, in), "good"
}

// wmoText maps a WMO weather code to a short label (Open-Meteo convention).
func wmoText(c int) string {
	switch {
	case c == 0:
		return "Clear"
	case c <= 2:
		return "Mostly clear"
	case c == 3:
		return "Overcast"
	case c >= 45 && c <= 48:
		return "Fog"
	case c >= 51 && c <= 57:
		return "Drizzle"
	case c >= 61 && c <= 67:
		return "Rain"
	case c >= 71 && c <= 77:
		return "Snow"
	case c >= 80 && c <= 82:
		return "Rain showers"
	case c >= 85 && c <= 86:
		return "Snow showers"
	case c >= 95:
		return "Thunderstorm"
	}
	return ""
}

func blowerPct(code int) int {
	switch code {
	case 1:
		return 25
	case 2:
		return 40
	case 3:
		return 55
	case 4:
		return 70
	case 5:
		return 85
	case 6:
		return 100
	}
	return code
}
