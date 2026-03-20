// Package modbusclient wraps github.com/goburrow/modbus to provide TCP and RTU
// connections for the Ovum heat-pump protocol.
package modbusclient

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/goburrow/modbus"
)

const (
	MethodTCP = "TCP"
	MethodRTU = "RTU"
)

// handler is a private interface that extends modbus.ClientHandler with the
// Connect/Close lifecycle methods that both TCPClientHandler and RTUClientHandler
// implement, but that are not part of the library's public interface.
type handler interface {
	modbus.ClientHandler
	Connect() error
	Close() error
}

// Client wraps a modbus handler together with a high-level Client.
// dialFn is stored so the connection can be re-established after a
// transaction-ID desync without restarting the whole process.
type Client struct {
	h      handler
	Client modbus.Client
	dialFn func() (handler, error)
}

// ConnectTCP opens a Modbus TCP connection to host:port.
func ConnectTCP(host string, port int) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	dialFn := func() (handler, error) {
		h := modbus.NewTCPClientHandler(addr)
		h.Timeout = 10 * time.Second
		h.IdleTimeout = 30 * time.Second
		if err := h.Connect(); err != nil {
			return nil, fmt.Errorf("modbus TCP connect %s: %w", addr, err)
		}
		return h, nil
	}
	h, err := dialFn()
	if err != nil {
		slog.Error("modbus TCP connect failed", "addr", addr, "err", err)
		return nil, err
	}
	slog.Info("modbus TCP connected", "addr", addr)
	return &Client{h: h, Client: modbus.NewClient(h), dialFn: dialFn}, nil
}

// ConnectRTU opens a Modbus RTU connection over a serial port.
func ConnectRTU(comPort string, baudRate int, parity string, stopBits int) (*Client, error) {
	dialFn := func() (handler, error) {
		h := modbus.NewRTUClientHandler(comPort)
		h.BaudRate = baudRate
		h.DataBits = 8
		h.StopBits = stopBits
		h.Timeout = 5 * time.Second
		switch parity {
		case "E":
			h.Parity = "E"
		case "O":
			h.Parity = "O"
		default:
			h.Parity = "N"
		}
		if err := h.Connect(); err != nil {
			return nil, fmt.Errorf("modbus RTU connect %s: %w", comPort, err)
		}
		return h, nil
	}
	h, err := dialFn()
	if err != nil {
		slog.Error("modbus RTU connect failed", "port", comPort, "err", err)
		return nil, err
	}
	slog.Info("modbus RTU connected", "port", comPort, "baud", baudRate, "parity", parity)
	return &Client{h: h, Client: modbus.NewClient(h), dialFn: dialFn}, nil
}

// isTransactionIDMismatch returns true when the goburrow library signals that
// the TCP response carried a different transaction ID than the request.  This
// indicates the connection state is desynced and must be reset.
func isTransactionIDMismatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "transaction id")
}

// reconnect tears down the current transport and dials a fresh one.
func (c *Client) reconnect() error {
	slog.Warn("modbus reconnecting after transaction ID desync")
	_ = c.h.Close()
	h, err := c.dialFn()
	if err != nil {
		slog.Error("modbus reconnect failed", "err", err)
		return err
	}
	c.h = h
	c.Client = modbus.NewClient(h)
	slog.Info("modbus reconnected successfully")
	return nil
}

// setSlaveID applies the slave/unit ID to whichever handler type is in use.
func (c *Client) setSlaveID(slave byte) {
	switch h := c.h.(type) {
	case *modbus.TCPClientHandler:
		h.SlaveId = slave
	case *modbus.RTUClientHandler:
		h.SlaveId = slave
	}
}

// ReadHoldingRegisters reads count holding registers starting at address for
// the given slave/unit ID. Returns raw bytes (2 bytes per register, big-endian).
//
// On a transaction-ID mismatch the connection is automatically re-established
// and the read is retried once on the fresh connection, breaking the desync
// cascade without requiring a full process restart.
func (c *Client) ReadHoldingRegisters(slave byte, address, count uint16) ([]byte, error) {
	// goburrow stores SlaveId on the handler; set it before every call so
	// RTU multi-slave setups work correctly.
	c.setSlaveID(slave)

	slog.Debug("reading holding registers", "slave", slave, "address", address, "count", count)
	data, err := c.Client.ReadHoldingRegisters(address, count)
	if err != nil {
		if isTransactionIDMismatch(err) {
			slog.Warn("transaction ID mismatch – reconnecting and retrying",
				"slave", slave, "address", address, "err", err)
			if rerr := c.reconnect(); rerr != nil {
				return nil, rerr
			}
			c.setSlaveID(slave)
			data, err = c.Client.ReadHoldingRegisters(address, count)
		}
		if err != nil {
			slog.Warn("read holding registers failed", "slave", slave, "address", address, "err", err)
			return nil, err
		}
	}
	return data, nil
}

// Close closes the underlying transport connection.
func (c *Client) Close() error {
	err := c.h.Close()
	if err != nil {
		slog.Warn("modbus close error", "err", err)
	} else {
		slog.Info("modbus connection closed")
	}
	return err
}
