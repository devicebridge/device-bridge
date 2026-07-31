package hub

import "github.com/devicebridge/device-bridge/internal/message"

// Client receives transport messages.
type Client interface {
	Send(message.Message) error
}
