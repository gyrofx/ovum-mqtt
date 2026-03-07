package config

import "time"

// Config holds all runtime configuration for ovum-mqtt.
type Config struct {
	// Modbus connection
	Method   string // "TCP" or "RTU"
	Host     string
	Port     int
	ComPort  string
	BaudRate int
	Parity   string // "E", "O", "N"
	StopBits int
	Slave    int

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
		Method:          "TCP",
		Host:            "127.0.0.1",
		Port:            502,
		ComPort:         "/dev/ttyUSB0",
		BaudRate:        19200,
		Parity:          "E",
		StopBits:        1,
		Slave:           247,
		MQTTBroker:      "tcp://localhost:1883",
		MQTTClientID:    "ovum-mqtt",
		MQTTTopicPrefix: "ovum",
		MQTTDeviceID:    "heatpump",
		MQTTQOS:         0,
		PollInterval:    20 * time.Second,
	}
}
