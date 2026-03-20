package config

import "time"

// Config holds all runtime configuration for ovum-mqtt.
type Config struct {
	// Modbus TCP connection
	Host  string
	Port  int
	Slave int

	// MQTT broker
	MQTTBroker      string // e.g. "tcp://localhost:1883"
	MQTTClientID    string
	MQTTUsername    string
	MQTTPassword    string
	MQTTTopicPrefix string // e.g. "ovum" → publishes to "ovum/<DeviceID>/status"
	MQTTDeviceID    string // e.g. "heatpump" → topic becomes "ovum/heatpump/status"
	MQTTQOS         byte

	// Polling
	PollInterval time.Duration
}

// Defaults returns a Config with sensible default values.
func Defaults() Config {
	return Config{
		Host:            "127.0.0.1",
		Port:            502,
		Slave:           247,
		MQTTBroker:      "tcp://localhost:1883",
		MQTTClientID:    "ovum-mqtt",
		MQTTTopicPrefix: "ovum",
		MQTTDeviceID:    "heatpump",
		MQTTQOS:         0,
		PollInterval:    20 * time.Second,
	}
}
