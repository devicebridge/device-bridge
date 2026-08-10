package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/websocket"
)

const shutdownTimeout = 5 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	b := bridge.New()

	srv := server.New()

	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr(),
		Handler: srv.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bridgeDone := make(chan error, 1)
	go func() {
		bridgeDone <- b.Run(ctx)
	}()

	httpFatal := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			httpFatal <- err
		}
	}()

	var appErr error

	select {
	case err := <-bridgeDone:
		appErr = err
	case err := <-httpFatal:
		log.Printf("http server startup error: %v", err)
		appErr = err
		stop()
		bridgeErr := <-bridgeDone
		if bridgeErr != nil && !errors.Is(bridgeErr, context.Canceled) {
			log.Printf("runtime error during shutdown: %v", bridgeErr)
		}
	}

	b.Hub().Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http graceful shutdown error: %v", err)
	}

	if appErr != nil && !errors.Is(appErr, context.Canceled) {
		log.Printf("application exited with error: %v", appErr)
		os.Exit(1)
	}
}
