package websocket

import (
	"net/http"

	"github.com/devicebridge/device-bridge/internal/hub"
	gorilla "github.com/gorilla/websocket"
)

var _ http.Handler = (*Handler)(nil)

// Handler upgrades HTTP connections to WebSocket.
type Handler struct {
	hub      *hub.Hub
	upgrader gorilla.Upgrader
}

// NewHandler creates a new WebSocket handler.
func NewHandler(h *hub.Hub) *Handler {
	return &Handler{
		hub: h,
		upgrader: gorilla.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ServeHTTP upgrades an HTTP connection to WebSocket.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(conn)

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	defer conn.Close()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
