package websocket

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

const outboundQueueSize = 16

var (
	ErrClosed    = errors.New("websocket client is closed")
	ErrQueueFull = errors.New("websocket client outbound queue is full")
)

type Client struct {
	conn *websocket.Conn

	mu        sync.Mutex
	closed    bool
	writeErr  error
	done      chan struct{}
	outbound  chan []byte
	closeOnce sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	c := &Client{
		conn:     conn,
		done:     make(chan struct{}),
		outbound: make(chan []byte, outboundQueueSize),
	}
	go c.writeLoop()
	return c
}

// Send enqueues a message without blocking on the network write. A full
// outbound queue makes the client unhealthy so Hub can remove it.
func (c *Client) Send(msg message.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		if c.writeErr != nil {
			return c.writeErr
		}
		return ErrClosed
	}
	if c.writeErr != nil {
		return c.writeErr
	}

	select {
	case c.outbound <- data:
		return nil
	default:
		return ErrQueueFull
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.done)
		c.mu.Unlock()

		c.conn.Close()
	})
}

func (c *Client) writeLoop() {
	for {
		select {
		case data := <-c.outbound:
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.fail(err)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) fail(err error) {
	c.mu.Lock()
	if c.writeErr == nil {
		c.writeErr = err
	}
	c.closed = true
	c.mu.Unlock()
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}
