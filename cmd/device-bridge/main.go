package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/source"
	"github.com/devicebridge/device-bridge/internal/source/scanner"
	"github.com/devicebridge/device-bridge/internal/websocket"
)

const shutdownTimeout = 5 * time.Second

type application struct {
	bridge   *bridge.Bridge
	adapters []*scanner.ChannelAdapter
	serials  []*scanner.SerialAdapter
	listener net.Listener
	server   *http.Server
	router   *server.Server
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runApplication(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("application exited with error: %v", err)
		os.Exit(1)
	}
}

func runApplication(ctx context.Context, cfg *config.Config) error {
	app, err := newApplication(cfg)
	if err != nil {
		return err
	}
	defer app.closeAdapters()

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	bridgeDone := make(chan error, 1)
	serialDone := make(chan error, len(app.serials))
	for _, serial := range app.serials {
		go func(serial *scanner.SerialAdapter) {
			serialDone <- serial.Run(runtimeCtx)
		}(serial)
	}
	go func() { bridgeDone <- app.bridge.Run(runtimeCtx) }()

	serveDone := make(chan error, 1)
	go func() { serveDone <- app.server.Serve(app.listener) }()
	app.router.SetReady(true)

	var (
		appErr         error
		bridgeErr      error
		bridgeDoneSeen bool
	)
	select {
	case <-ctx.Done():
		cancel()
		appErr = ctx.Err()
	case err := <-bridgeDone:
		bridgeErr = err
		bridgeDoneSeen = true
		appErr = err
		cancel()
	case err := <-serveDone:
		if !errors.Is(err, http.ErrServerClosed) {
			appErr = err
			cancel()
		}
	}

	if !bridgeDoneSeen {
		bridgeErr = <-bridgeDone
	}
	for range app.serials {
		<-serialDone
	}
	if appErr == nil && bridgeErr != nil && !errors.Is(bridgeErr, context.Canceled) {
		appErr = bridgeErr
	}

	app.bridge.Hub().Shutdown()
	app.router.SetReady(false)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := app.server.Shutdown(shutdownCtx); err != nil && appErr == nil {
		appErr = fmt.Errorf("http graceful shutdown: %w", err)
	}

	return appErr
}

func newApplication(cfg *config.Config) (*application, error) {
	listener, err := net.Listen("tcp", cfg.ListenAddr())
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", cfg.ListenAddr(), err)
	}

	b := bridge.New()
	sourceCfg := *cfg
	if cfg.ScannerPath != "" {
		sourceCfg.Sources = make([]string, 0, len(cfg.Sources))
		for _, name := range cfg.Sources {
			if name != "scanner-main" {
				sourceCfg.Sources = append(sourceCfg.Sources, name)
			}
		}
	}
	adapters, err := configureSources(b, &sourceCfg)
	if err != nil {
		listener.Close()
		return nil, err
	}
	srv := server.New()
	srv.Handle("/ws", websocket.NewHandler(b.Hub()))
	httpServer := &http.Server{Handler: srv.Handler()}
	serials := make([]*scanner.SerialAdapter, 0)
	if cfg.ScannerPath != "" {
		port, err := os.Open(cfg.ScannerPath)
		if err != nil {
			listener.Close()
			return nil, fmt.Errorf("open scanner path %s: %w", cfg.ScannerPath, err)
		}
		serial := scanner.NewSerialAdapter(port, 100)
		serials = append(serials, serial)
		if err := b.Registry().Register("scanner-main", func() source.Source {
			return scanner.New("scanner-main", serial.Input())
		}); err != nil {
			serial.Close()
			listener.Close()
			return nil, err
		}
	}
	return &application{bridge: b, adapters: adapters, serials: serials, listener: listener, server: httpServer, router: srv}, nil
}

func (a *application) closeAdapters() {
	for _, adapter := range a.adapters {
		adapter.Close()
	}
	for _, serial := range a.serials {
		serial.Close()
	}
}

func configureSources(b *bridge.Bridge, cfg *config.Config) ([]*scanner.ChannelAdapter, error) {
	adapters := make([]*scanner.ChannelAdapter, 0, len(cfg.Sources))
	for _, name := range cfg.Sources {
		switch name {
		case "scanner-main", "scanner-secondary":
			adapter := scanner.NewChannelAdapter(100)
			sourceID := name
			if err := b.Registry().Register(name, func() source.Source {
				return scanner.New(sourceID, adapter.Input())
			}); err != nil {
				return nil, err
			}
			adapters = append(adapters, adapter)
		default:
			return nil, fmt.Errorf("unknown source %q", name)
		}
	}
	return adapters, nil
}
