package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
)

func TestConfigureSources(t *testing.T) {
	b := bridge.New()
	cfg := &config.Config{Sources: []string{"scanner-main"}}

	if err := configureSources(b, cfg); err != nil {
		t.Fatalf("configureSources failed: %v", err)
	}

	if src, err := b.Registry().Create("scanner-main"); err != nil || src == nil {
		t.Fatalf("scanner source was not registered: source=%v err=%v", src, err)
	}
}

func TestConfigureSourcesRejectsUnknownSource(t *testing.T) {
	b := bridge.New()
	cfg := &config.Config{Sources: []string{"unknown"}}

	if err := configureSources(b, cfg); err == nil {
		t.Fatal("expected unknown source error")
	}
}

func TestRunApplicationRejectsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	cfg := &config.Config{HTTPHost: "127.0.0.1", HTTPPort: addr.Port}

	err = runApplication(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected occupied port error")
	}
}

func TestRunApplicationStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	cfg := &config.Config{HTTPHost: "127.0.0.1", HTTPPort: 0}

	go func() { done <- runApplication(ctx, cfg) }()
	cancel()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected application error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runApplication did not stop after cancellation")
	}
}
