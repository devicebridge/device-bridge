package websocket

import (
	"encoding/json"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	closeOnce sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func (c *Client) Send(msg message.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.mu.TryLock() {
			msg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "")
			c.conn.WriteMessage(websocket.CloseMessage, msg)
			c.mu.Unlock()
		}
		c.conn.Close()
	})
}
