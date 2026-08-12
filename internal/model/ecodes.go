// Package model — Aurora E-code fault names, recovered from the controller UI
// (a.Sa in indexc.js). Generated from regmap/merged.json; see FAULTS.md.
package model

var ecodes = map[int]string{
	1:  "Input Error",
	2:  "High Pressure",
	3:  "Low Pressure",
	4:  "Frze Detect FP2",
	5:  "Frze Detect FP1",
	7:  "Condensate",
	8:  "O/U Voltage",
	9:  "Airflow/RPM",
	10: "Comp Monitor",
	11: "FP1/2 Snr Err",
	12: "RefrigPerformnc",
	13: "NCritAxbSnrErr",
	14: "CritAxbSnrErr",
	15: "Hot Water Limit",
	16: "VS Pump Error",
	18: "NCritComm Error",
	19: "Crit Comm Error",
	22: "Comm ECM Error",
	25: "AXB EEV Error",
	26: "EntSrcLowLimAlm",
	27: "EntSrcHiLimAlm",
	28: "LvgSrcLowLimAlm",
	29: "LvgSrcHiLimAlm",
	31: "Src Flow Fault",
	32: "Load Flow Fault",
	41: "Hi Drive Temp",
	42: "Hi Dischrg Temp",
	43: "Lo Suct Press",
	44: "Lo Cond Press",
	45: "Hi Cond Press",
	46: "Output Pwr Lmt",
	47: "EEV ID Comm Err",
	48: "EEV OD Comm Err",
	49: "Cabinet Tmp Snr",
	51: "Dischrg Tmp Snr",
	52: "Suct Press Snr",
	53: "Cond Press Snr",
	54: "Lo Supply Volt",
	55: "Out of Envelope",
	56: "Drv Over Current",
	57: "Drive O/U Volt",
	58: "Hi Drive Temp",
	59: "Int Drive Error",
	61: "Multiple SafeMd",
	62: "Low Temp",
	63: "Fault Limit",
	64: "EVC Intrnl Flt",
	65: "Soft Start Fail",
	66: "Power Fault",
	67: "EVC Intrnl Flt",
	69: "Invalid Comp Id",
	71: "Loss of Charge",
	72: "Suct Temp Snr",
	73: "LvgAir Temp Snr",
	74: "Max Op Pressure",
	75: "Loss of Charge",
	76: "Suct Temp Snr",
	77: "LvgAir Temp Snr",
	78: "Max Op Pressure",
	99: "System Reset",
}

// ECode returns the fault name for an Aurora E-code, or "Unknown".
func ECode(code int) string {
	if n, ok := ecodes[code]; ok {
		return n
	}
	return "Unknown"
}
