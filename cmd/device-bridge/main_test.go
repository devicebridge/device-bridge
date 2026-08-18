package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source/scanner"
	"github.com/gorilla/websocket"
)

func TestConfigureSources(t *testing.T) {
	b := bridge.New()
	cfg := &config.Config{Sources: []string{"scanner-main"}}

	adapters, err := configureSources(b, cfg)
	if err != nil {
		t.Fatalf("configureSources failed: %v", err)
	}
	defer adapters[0].Close()

	if src, err := b.Registry().Create("scanner-main"); err != nil || src == nil {
		t.Fatalf("scanner source was not registered: source=%v err=%v", src, err)
	}
}

func TestConfigureSourcesRejectsUnknownSource(t *testing.T) {
	b := bridge.New()
	cfg := &config.Config{Sources: []string{"unknown"}}

	if _, err := configureSources(b, cfg); err == nil {
		t.Fatal("expected unknown source error")
	}
}

func TestNewApplicationWithSerialPath(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "scanner-*")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	app, err := newApplication(&config.Config{
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		Sources:     []string{"scanner-main"},
		ScannerPath: file.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.closeAdapters()
	if len(app.serials) != 1 {
		t.Fatalf("expected one serial adapter, got %d", len(app.serials))
	}
	app.listener.Close()
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

func TestApplicationSourceToWebSocket(t *testing.T) {
	app, err := newApplication(&config.Config{
		HTTPHost: "127.0.0.1",
		HTTPPort: 0,
		Sources:  []string{"scanner-main"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.closeAdapters()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridgeDone := make(chan error, 1)
	go func() { bridgeDone <- app.bridge.Run(ctx) }()
	serveDone := make(chan error, 1)
	go func() { serveDone <- app.server.Serve(app.listener) }()
	defer func() {
		app.bridge.Hub().Shutdown()
		_ = app.server.Shutdown(context.Background())
		<-bridgeDone
		<-serveDone
	}()

	url := "ws" + strings.TrimPrefix("http://"+app.listener.Addr().String(), "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	if err := app.bridge.Hub().WaitForCount(waitCtx, 1); err != nil {
		t.Fatal(err)
	}

	if err := app.adapters[0].Publish(context.Background(), scanner.Input{Value: "from-adapter"}); err != nil {
		t.Fatal(err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var got message.Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "scanner-main" || got.Payload != "from-adapter" {
		t.Fatalf("unexpected message: %+v", got)
	}
	cancel()
}
