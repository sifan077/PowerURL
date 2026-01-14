package natsclient

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sifan077/PowerURL/config"
)

const defaultConnectTimeout = 5 * time.Second

// Connect creates a NATS connection (with JetStream available) using application config.
func Connect(cfg config.NATSConfig) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.Timeout(defaultConnectTimeout),
		nats.Name("powerurl"),
	}

	if cfg.User != "" {
		opts = append(opts, nats.UserInfo(cfg.User, cfg.Password))
	}

	url := buildURL(cfg)

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("nats: init jetstream: %w", err)
	}

	return conn, js, nil
}

func buildURL(cfg config.NATSConfig) string {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 4222
	}
	return fmt.Sprintf("nats://%s:%d", host, port)
}
