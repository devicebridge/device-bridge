package websocket

import (
	"log"
	"net/http"

	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub      *hub.Hub
	upgrader websocket.Upgrader
}

func NewHandler(h *hub.Hub) *Handler {
	return &Handler{
		hub:      h,
		upgrader: websocket.Upgrader{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	client := NewClient(conn)

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
