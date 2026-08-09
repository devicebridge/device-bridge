package bus

import "github.com/devicebridge/device-bridge/internal/message"

// Bus transports messages between producers and consumers.
type Bus struct {
	ch chan message.Message
}

// New creates a new message bus.
func New(size int) *Bus {
	if size < 0 {
		panic("bus size cannot be negative")
	}

	return &Bus{
		ch: make(chan message.Message, size),
	}
}

// Publish sends a message to the bus.
func (b *Bus) Publish(msg message.Message) {
	b.ch <- msg
}

// Subscribe returns a receive-only message channel.
func (b *Bus) Subscribe() <-chan message.Message {
	return b.ch
}

// Close closes the bus, preventing further publishes.
func (b *Bus) Close() {
	close(b.ch)
}
