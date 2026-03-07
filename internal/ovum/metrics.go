package ovum

// Metric maps a Modbus register address to a short MQTT topic name.
// The MQTT topic will be:  <prefix>/<TopicName>
// The full Grafana/Prometheus-style name is "ovum_" + TopicName.
type Metric struct {
	Address   uint16
	TopicName string // snake_case without the "ovum_" prefix
}

// Metrics is the canonical list of registers to read and publish.
// Register addresses and parameter codes match the Python implementation.
//
//nolint:gochecknoglobals
var Metrics = []Metric{
	// Compressor / inverter
	{12308, "inverter_rps"},      // Rps   – Inverter RPS set
	{13498, "inverter_power"},    // DrKw  – Inverter power  (kW)
	{13488, "drive_temperature"}, // DrTm  – Drive temperature (°C)

	// Run-time counters
	{12348, "running_time"},  // CoMi  – Running time (min)
	{12358, "working_hours"}, // CoHo  – Working hours (h)

	// Thermal output
	{12368, "heating_power"}, // HePw  – Heating power (kW)

	// Energy counters
	{12528, "weekly_heating_energy"},  // HeaW  – Weekly heating energy (kWh)
	{12538, "monthly_heating_energy"}, // HeaM  – Monthly heating energy (kWh)
	{12548, "yearly_heating_energy"},  // HeaY  – Yearly heating energy (kWh)
	{12558, "total_heating_energy"},   // HeaT  – Total heating energy (kWh)

	// COP
	{12578, "cop_weekly"},  // COPW
	{12588, "cop_monthly"}, // COPM
	{12598, "cop_yearly"},  // COPY
	{12608, "cop_total"},   // TOTA

	// Temperatures – ambient / outdoor
	{12388, "ambient_temp_avg"}, // ATvz  – Ambient temp avg (°C)
	{14608, "outdoor_temp"},     // AI9s  – Outdoor temp (°C)

	// Temperatures – system
	{12398, "controller_temp"},      // ReTe  – Controller / condenser temp (°C)
	{12408, "dhw_tank_upper_temp"},  // Spo   – DHW tank upper (°C)
	{12418, "dhw_tank_middle_temp"}, // Spm   – DHW tank middle (°C)
	{12428, "dhw_tank_lower_temp"},  // Spu   – DHW tank lower / heating tank lower (°C)

	// Ground source (brine) circuit
	{12988, "ground_source_in"},   // EQin  – Brine inlet temp (°C)
	{12998, "ground_source_out"},  // EQou  – Brine outlet temp (°C)
	{13008, "ground_source_pump"}, // AO02  – Brine pump (%)

	// Heating circuit 1
	{13898, "heat_circle_1_flow_temp"},       // AI07  – HC1 flow temp (°C)
	{13918, "heat_circle_1_return_temp"},     // AI08  – HC1 return temp (°C)
	{14848, "heat_circle_1_return_set_temp"}, // HReS  – HC1 return set-point (°C)
	{14898, "heat_circle_1_pump"},            // AO11  – HC1 pump (%)

	// Domestic hot water tap
	{15538, "tap_water_temp"},   // DoTs  – Tap water set temp (°C)
	{15548, "tap_act_temp"},     // DoT   – Tap actual temp (°C)
	{15578, "tap_pump_min"},     // FpMi  – Tap pump minimum (%)
	{15448, "tap_pump_percent"}, // SpTo  – Tap pump percent (%)
}
