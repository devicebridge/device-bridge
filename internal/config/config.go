package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultHTTPHost = "0.0.0.0"
	defaultHTTPPort = 8080
)

type Config struct {
	HTTPHost string
	HTTPPort int
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPHost: defaultHTTPHost,
		HTTPPort: defaultHTTPPort,
	}

	if host := os.Getenv("DEVICE_BRIDGE_HTTP_HOST"); host != "" {
		cfg.HTTPHost = host
	}

	if portStr := os.Getenv("DEVICE_BRIDGE_HTTP_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DEVICE_BRIDGE_HTTP_PORT: %w", err)
		}

		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("DEVICE_BRIDGE_HTTP_PORT out of range: %d", port)
		}

		cfg.HTTPPort = port
	}

	return cfg, nil
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}
