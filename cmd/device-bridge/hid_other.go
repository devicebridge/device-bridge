//go:build !linux

package main

import (
	"fmt"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
)

func configureHID(_ *bridge.Bridge, cfg *config.Config) ([]hidRuntime, error) {
	if cfg.HIDPath != "" {
		return nil, fmt.Errorf("HID path is only supported on Linux: %q", cfg.HIDPath)
	}
	return nil, nil
}
