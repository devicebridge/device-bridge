//go:build linux

package main

import (
	"fmt"
	"os"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
	"github.com/devicebridge/device-bridge/internal/source"
	"github.com/devicebridge/device-bridge/internal/source/hid"
	"github.com/devicebridge/device-bridge/internal/source/scanner"
)

func configureHID(b *bridge.Bridge, cfg *config.Config) ([]hidRuntime, error) {
	if cfg.HIDPath == "" {
		return nil, nil
	}
	device, err := os.Open(cfg.HIDPath)
	if err != nil {
		return nil, fmt.Errorf("open HID path %s: %w", cfg.HIDPath, err)
	}
	adapter := hid.New(device, 100)
	if err := b.Registry().Register("scanner-main", func() source.Source {
		return scanner.New("scanner-main", adapter.Input())
	}); err != nil {
		adapter.Close()
		return nil, err
	}
	return []hidRuntime{adapter}, nil
}
