package hub

import (
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
)

// Hub broadcasts messages to registered clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[Client]struct{}
	closed  chan struct{}
	once    sync.Once
}

// New creates a new Hub.
func New() *Hub {
	return &Hub{
		clients: make(map[Client]struct{}),
		closed:  make(chan struct{}),
	}
}

// Register adds a client.
func (h *Hub) Register(client Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	select {
	case <-h.closed:
		return
	default:
	}

	h.clients[client] = struct{}{}
}

// Unregister removes a client.
func (h *Hub) Unregister(client Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)
}

// Count returns the number of registered clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

// Broadcast sends a message to all registered clients.
func (h *Hub) Broadcast(msg message.Message) error {
	h.mu.RLock()

	select {
	case <-h.closed:
		h.mu.RUnlock()
		return nil
	default:
	}

	clients := make([]Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	h.mu.RUnlock()

	for _, client := range clients {
		if err := client.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown closes the hub and closes all registered clients.
func (h *Hub) Shutdown() {
	h.once.Do(func() {
		close(h.closed)

		h.mu.Lock()
		clients := make([]Client, 0, len(h.clients))
		for client := range h.clients {
			clients = append(clients, client)
		}
		h.mu.Unlock()

		for _, client := range clients {
			client.Close()
		}
	})
}

// Done returns a channel that is closed when the hub is shut down.
func (h *Hub) Done() <-chan struct{} {
	return h.closed
}
