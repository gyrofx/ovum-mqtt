package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gyrofx/ovum-exporter/internal/config"
	"github.com/gyrofx/ovum-exporter/internal/modbusclient"
	"github.com/gyrofx/ovum-exporter/internal/ovum"
	"github.com/gyrofx/ovum-exporter/internal/publisher"
)

func main() {
	defaults := config.Defaults()

	v := viper.New()
	v.SetEnvPrefix("OVUM")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	root := &cobra.Command{
		Use:   "ovum-mqtt",
		Short: "Reads Ovum heat-pump registers via Modbus and publishes to MQTT",
		// Bind pflags to viper after cobra has parsed CLI args, so that flag
		// values take precedence over env vars (viper precedence: flag > env > default).
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return v.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{
				Host:            v.GetString("host"),
				Port:            v.GetInt("port"),
				Slave:           v.GetInt("slave"),
				MQTTBroker:      v.GetString("mqtt-broker"),
				MQTTClientID:    v.GetString("mqtt-client-id"),
				MQTTUsername:    v.GetString("mqtt-username"),
				MQTTPassword:    v.GetString("mqtt-password"),
				MQTTTopicPrefix: v.GetString("mqtt-topic-prefix"),
				MQTTDeviceID:    v.GetString("mqtt-device-id"),
				PollInterval:    v.GetDuration("interval"),
			}
			return run(cfg)
		},
	}

	f := root.Flags()

	// Modbus TCP
	f.String("host", defaults.Host, "Modbus TCP host IP  [OVUM_HOST]")
	f.Int("port", defaults.Port, "Modbus TCP port  [OVUM_PORT]")
	f.Int("slave", defaults.Slave, "Modbus slave / unit ID  [OVUM_SLAVE]")

	// MQTT
	f.String("mqtt-broker", defaults.MQTTBroker, "MQTT broker URL  [OVUM_MQTT_BROKER]")
	f.String("mqtt-client-id", defaults.MQTTClientID, "MQTT client ID  [OVUM_MQTT_CLIENT_ID]")
	f.String("mqtt-username", defaults.MQTTUsername, "MQTT username  [OVUM_MQTT_USERNAME]")
	f.String("mqtt-password", defaults.MQTTPassword, "MQTT password  [OVUM_MQTT_PASSWORD]")
	f.String("mqtt-topic-prefix", defaults.MQTTTopicPrefix, "MQTT topic prefix  [OVUM_MQTT_TOPIC_PREFIX]")
	f.String("mqtt-device-id", defaults.MQTTDeviceID, "Device ID in topic: <prefix>/<id>/status  [OVUM_MQTT_DEVICE_ID]")

	// Poll interval
	f.Duration("interval", defaults.PollInterval, "How often to poll the heat pump  [OVUM_INTERVAL]")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	slog.Info("starting ovum-mqtt",
		"host", cfg.Host,
		"port", cfg.Port,
		"slave", cfg.Slave,
		"mqtt_broker", cfg.MQTTBroker,
		"topic_prefix", cfg.MQTTTopicPrefix,
		"interval", cfg.PollInterval,
	)

	// ---------- Modbus connection ----------
	mbClient, err := modbusclient.Connect(cfg.Host, cfg.Port)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := mbClient.Close(); cerr != nil {
			slog.Warn("modbus close error", "err", cerr)
		}
	}()

	// ---------- MQTT connection ----------
	slog.Info("Connecting to MQTT broker", "broker", cfg.MQTTBroker)
	pub, err := publisher.New(
		cfg.MQTTBroker, cfg.MQTTClientID,
		cfg.MQTTUsername, cfg.MQTTPassword,
		cfg.MQTTTopicPrefix, cfg.MQTTQOS,
	)
	if err != nil {
		return err
	}
	defer pub.Close()

	// ---------- Signal handling ----------
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("Starting poll loop", "interval", cfg.PollInterval, "metrics", len(ovum.Metrics))

	// Poll immediately, then on each tick.
	if err := poll(mbClient, pub, byte(cfg.Slave), cfg.MQTTDeviceID); err != nil {
		slog.Error("first poll failed", "err", err)
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	// consecutiveErrors counts back-to-back poll failures.  Once the threshold
	// is exceeded the process exits so Docker restarts it with a clean state.
	const maxConsecutiveErrors = 3
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutting down")
			return nil
		case <-ticker.C:
			slog.Debug("poll tick")
			if err := poll(mbClient, pub, byte(cfg.Slave), cfg.MQTTDeviceID); err != nil {
				consecutiveErrors++
				slog.Error("poll error",
					"consecutive", consecutiveErrors,
					"max", maxConsecutiveErrors,
					"err", err)
				if consecutiveErrors >= maxConsecutiveErrors {
					slog.Error("too many consecutive poll errors – exiting for container restart",
						"count", consecutiveErrors)
					os.Exit(1)
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

// poll reads every metric register, collects all values, and publishes them
// as a single JSON message to "<prefix>/<deviceID>/status".
func poll(mb *modbusclient.Client, pub *publisher.Publisher, slave byte, deviceID string) error {
	start := time.Now()
	var lastErr error
	fields := make(map[string]float64, len(ovum.Metrics))
	skipped := 0

	for _, m := range ovum.Metrics {
		data, err := mb.ReadHoldingRegisters(slave, m.Address, ovum.RegisterBlockSize)
		if err != nil {
			slog.Warn("modbus read error", "metric", m.TopicName, "address", m.Address, "err", err)
			lastErr = err
			skipped++
			continue
		}

		rv, err := ovum.Decode(m.Address, data)
		if err != nil {
			slog.Warn("decode error", "metric", m.TopicName, "address", m.Address, "err", err)
			skipped++
			continue
		}

		slog.Debug("read", "metric", m.TopicName, "value", rv.Value)
		fields[m.TopicName] = rv.Value
	}

	if err := pub.PublishStatus(deviceID, fields); err != nil {
		slog.Error("publish status failed", "err", err)
		lastErr = err
	}

	slog.Info("poll complete", "published", len(fields), "skipped", skipped, "duration", time.Since(start))
	return lastErr
}
