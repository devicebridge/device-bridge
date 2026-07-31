package hub

import "github.com/devicebridge/device-bridge/internal/message"

// Hub broadcasts messages to registered clients.
type Hub struct {
	clients map[Client]struct{}
}

// New creates a new Hub.
func New() *Hub {
	return &Hub{
		clients: make(map[Client]struct{}),
	}
}

// Register adds a client.
func (h *Hub) Register(client Client) {
	if client == nil {
		return
	}
	h.clients[client] = struct{}{}
}

// Unregister removes a client.
func (h *Hub) Unregister(client Client) {
        if client == nil {
                return
        }
	delete(h.clients, client)
}

// Broadcast sends a message to all registered clients.
func (h *Hub) Broadcast(msg message.Message) error {
	for client := range h.clients {
		if err := client.Send(msg); err != nil {
			return err
		}
	}

	return nil
}
