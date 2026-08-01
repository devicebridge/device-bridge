package websocket

import (
	"encoding/json"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

// Client implements hub.Client using WebSocket.
type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// NewClient creates a new WebSocket client.
func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

// Send sends a message over WebSocket.
func (c *Client) Send(msg message.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteMessage(websocket.TextMessage, data)
}
