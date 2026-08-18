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
	listener, err := net.Listen("tcp", cfg.ListenAddr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr(), err)
	}

	b := bridge.New()
	if err := configureSources(b, cfg); err != nil {
		return err
	}
	srv := server.New()
	srv.Handle("/ws", websocket.NewHandler(b.Hub()))
	httpServer := &http.Server{Handler: srv.Handler()}

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	bridgeDone := make(chan error, 1)
	go func() { bridgeDone <- b.Run(runtimeCtx) }()

	serveDone := make(chan error, 1)
	go func() { serveDone <- httpServer.Serve(listener) }()

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
	if appErr == nil && bridgeErr != nil && !errors.Is(bridgeErr, context.Canceled) {
		appErr = bridgeErr
	}

	b.Hub().Shutdown()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && appErr == nil {
		appErr = fmt.Errorf("http graceful shutdown: %w", err)
	}

	return appErr
}

func configureSources(b *bridge.Bridge, cfg *config.Config) error {
	for _, name := range cfg.Sources {
		input := make(chan scanner.Input)
		switch name {
		case "scanner-main", "scanner-secondary":
			sourceID := name
			if err := b.Registry().Register(name, func() source.Source {
				return scanner.New(sourceID, input)
			}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown source %q", name)
		}
	}
	return nil
}
