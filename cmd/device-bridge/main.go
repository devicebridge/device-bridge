package main

import (
	"log"
	"net/http"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/config"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	b := bridge.New()

	srv := server.New()

	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	go b.Run()

	log.Fatal(http.ListenAndServe(cfg.ListenAddr(), srv.Handler()))
}
