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

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	err = <-bridgeDone

	b.Hub().Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("bridge error: %v", err)
		os.Exit(1)
	}
}
