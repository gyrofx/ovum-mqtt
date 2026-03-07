// Package publisher handles the MQTT connection and message publishing.
package publisher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher wraps a paho MQTT client.
type Publisher struct {
	client      mqtt.Client
	topicPrefix string
	qos         byte
}

// New connects to the MQTT broker and returns a Publisher.
// brokerURL format: "tcp://host:port"
func New(brokerURL, clientID, username, password, topicPrefix string, qos byte) (*Publisher, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetOnConnectHandler(func(_ mqtt.Client) {
			slog.Info("MQTT connected", "broker", brokerURL)
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			slog.Warn("MQTT connection lost", "err", err)
		})

	if username != "" {
		opts.SetUsername(username).SetPassword(password)
	}

	slog.Info("connecting to MQTT broker", "broker", brokerURL, "clientID", clientID)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		slog.Error("MQTT connect timed out", "broker", brokerURL)
		return nil, fmt.Errorf("MQTT connect timeout to %s", brokerURL)
	}
	if err := token.Error(); err != nil {
		slog.Error("MQTT connect failed", "broker", brokerURL, "err", err)
		return nil, fmt.Errorf("MQTT connect %s: %w", brokerURL, err)
	}

	return &Publisher{client: client, topicPrefix: topicPrefix, qos: qos}, nil
}

// PublishFloat publishes a float64 value to "<prefix>/<name>".
// The payload is the shortest decimal representation of the value.
func (p *Publisher) PublishFloat(name string, value float64) error {
	topic := p.topicPrefix + "/" + name
	payload := strconv.FormatFloat(value, 'f', -1, 64)

	token := p.client.Publish(topic, p.qos, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		slog.Warn("MQTT publish timed out", "topic", topic)
		return fmt.Errorf("publish timeout: topic=%s", topic)
	}
	if err := token.Error(); err != nil {
		slog.Warn("MQTT publish failed", "topic", topic, "err", err)
		return err
	}
	slog.Debug("MQTT published", "topic", topic, "value", value)
	return nil
}

// PublishStatus publishes all fields as a single JSON object to "<prefix>/<deviceID>/status".
func (p *Publisher) PublishStatus(deviceID string, fields map[string]float64) error {
	topic := p.topicPrefix + "/" + deviceID + "/status"

	payload, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}

	token := p.client.Publish(topic, p.qos, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		slog.Warn("MQTT publish status timed out", "topic", topic)
		return fmt.Errorf("publish timeout: topic=%s", topic)
	}
	if err := token.Error(); err != nil {
		slog.Warn("MQTT publish status failed", "topic", topic, "err", err)
		return err
	}
	slog.Debug("MQTT published status", "topic", topic, "fields", len(fields))
	return nil
}

// Close disconnects from the broker.
func (p *Publisher) Close() {
	slog.Info("disconnecting from MQTT broker")
	p.client.Disconnect(500)
}
