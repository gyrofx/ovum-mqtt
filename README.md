# ovum-mqtt

A Go rewrite of the Ovum heat-pump exporter. Instead of exposing a Prometheus
HTTP endpoint it publishes every value to an MQTT broker so that downstream tools
(Telegraf, Home Assistant, Node-RED, etc.) can consume them.

## Topic structure

```
<prefix>/<device-id>/status
```

All fields are published as a **single JSON object** once per poll cycle. The default topic is `ovum/heatpump/status`:

```json
{
  "inverter_rps": 50,
  "outdoor_temp": 8.0,
  "cop_weekly": 5.3,
  "heating_power": 6.03
}
```

Use `--mqtt-device-id` (or `OVUM_MQTT_DEVICE_ID`) to distinguish multiple heat pumps on the same broker.

## All published metrics

| Topic | Register | Description |
|---|---|---|
| `inverter_rps` | 12308 | Inverter RPS set |
| `inverter_power` | 13498 | Inverter power (kW) |
| `drive_temperature` | 13488 | Drive temperature (°C) |
| `running_time` | 12348 | Running time (min) |
| `working_hours` | 12358 | Working hours (h) |
| `heating_power` | 12368 | Heating power (kW) |
| `weekly_heating_energy` | 12528 | Weekly heating energy (kWh) |
| `monthly_heating_energy` | 12538 | Monthly heating energy (kWh) |
| `yearly_heating_energy` | 12548 | Yearly heating energy (kWh) |
| `total_heating_energy` | 12558 | Total heating energy (kWh) |
| `cop_weekly` | 12578 | COP weekly |
| `cop_monthly` | 12588 | COP monthly |
| `cop_yearly` | 12598 | COP yearly |
| `cop_total` | 12608 | COP total |
| `ambient_temp_avg` | 12388 | Ambient temperature average (°C) |
| `outdoor_temp` | 14608 | Outdoor temperature (°C) |
| `controller_temp` | 12398 | Controller / condenser temperature (°C) |
| `dhw_tank_upper_temp` | 12408 | DHW tank upper temperature (°C) |
| `dhw_tank_middle_temp` | 12418 | DHW tank middle temperature (°C) |
| `dhw_tank_lower_temp` | 12428 | DHW tank lower temperature (°C) |
| `ground_source_in` | 12988 | Brine inlet temperature (°C) |
| `ground_source_out` | 12998 | Brine outlet temperature (°C) |
| `ground_source_pump` | 13008 | Brine pump (%) |
| `heat_circle_1_flow_temp` | 13898 | HC1 flow temperature (°C) |
| `heat_circle_1_return_temp` | 13918 | HC1 return temperature (°C) |
| `heat_circle_1_return_set_temp` | 14848 | HC1 return set-point temperature (°C) |
| `heat_circle_1_pump` | 14898 | HC1 pump (%) |
| `tap_water_temp` | 15538 | Tap water set temperature (°C) |
| `tap_act_temp` | 15548 | Tap actual temperature (°C) |
| `tap_pump_min` | 15578 | Tap pump minimum (%) |
| `tap_pump_percent` | 15448 | Tap pump percent (%) |

## Usage

```
ovum-mqtt [flags]

Flags:
  --method              TCP or RTU (default: TCP)           [OVUM_METHOD]
  --host                Modbus TCP host IP (default: 127.0.0.1) [OVUM_HOST]
  --port                Modbus TCP port (default: 502)      [OVUM_PORT]
  --slave               Modbus slave / unit ID (default: 247) [OVUM_SLAVE]
  --comport             Serial port for RTU (default: /dev/ttyUSB0) [OVUM_COMPORT]
  --baudrate            Baud rate for RTU (default: 19200)  [OVUM_BAUDRATE]
  --parity              Parity for RTU: E, O, N (default: E) [OVUM_PARITY]
  --stopbits            Stop bits for RTU (default: 1)      [OVUM_STOPBITS]
  --mqtt-broker         MQTT broker URL (default: tcp://localhost:1883) [OVUM_MQTT_BROKER]
  --mqtt-client-id      MQTT client ID (default: ovum-mqtt) [OVUM_MQTT_CLIENT_ID]
  --mqtt-username       MQTT username (optional)            [OVUM_MQTT_USERNAME]
  --mqtt-password       MQTT password (optional)            [OVUM_MQTT_PASSWORD]
  --mqtt-topic-prefix   Topic prefix (default: ovum)        [OVUM_MQTT_TOPIC_PREFIX]
  --mqtt-device-id      Device ID in topic (default: heatpump) [OVUM_MQTT_DEVICE_ID]
  --interval            Poll interval (default: 20s)        [OVUM_INTERVAL]
```

### Example — Modbus TCP

```sh
ovum-mqtt \
  --host 192.168.1.100 \
  --slave 247 \
  --mqtt-broker tcp://mosquitto:1883 \
  --interval 30s
```

### Example — Modbus RTU

```sh
ovum-mqtt \
  --method RTU \
  --comport /dev/ttyUSB0 \
  --slave 247 \
  --mqtt-broker tcp://mosquitto:1883
```

### Docker

```sh
docker build -t ovum-mqtt .
docker run --rm ovum-mqtt \
  --host 192.168.1.100 \
  --mqtt-broker tcp://192.168.1.10:1883
```

## Grafana integration

The recommended path is **MQTT → Telegraf → InfluxDB → Grafana**.

Example Telegraf MQTT consumer (parses the JSON status payload):

```toml
[[inputs.mqtt_consumer]]
  servers      = ["tcp://localhost:1883"]
  topics       = ["ovum/+/status"]
  data_format  = "json_v2"

  [[inputs.mqtt_consumer.json_v2]]
    [[inputs.mqtt_consumer.json_v2.field]]
      path = "inverter_rps"
    [[inputs.mqtt_consumer.json_v2.field]]
      path = "outdoor_temp"
    # ... add remaining fields or use a wildcard measurement
```

## Docker Compose

```yaml
services:
  ovum-mqtt:
    build: .
    environment:
      OVUM_HOST: 192.168.1.100
      OVUM_SLAVE: 247
      OVUM_MQTT_BROKER: tcp://mosquitto:1883
      OVUM_MQTT_DEVICE_ID: heatpump
      OVUM_INTERVAL: 20s
```

## Building

```sh
go build -o ovum-mqtt ./cmd/ovum-mqtt
```
