package bus

import (
	"context"
	"errors"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
)

var ErrClosed = errors.New("bus is closed")

// Bus transports messages between producers and consumers.
type Bus struct {
	ch   chan message.Message
	done chan struct{}
	once sync.Once
}

// New creates a new message bus.
func New(size int) *Bus {
	if size < 0 {
		panic("bus size cannot be negative")
	}

	return &Bus{
		ch:   make(chan message.Message, size),
		done: make(chan struct{}),
	}
}

// Publish sends a message to the bus.
func (b *Bus) Publish(msg message.Message) {
	select {
	case b.ch <- msg:
	case <-b.done:
	}
}

// PublishCtx sends a message to the bus or returns an error if
// the context is cancelled before the message can be sent.
func (b *Bus) PublishCtx(ctx context.Context, msg message.Message) error {
	select {
	case b.ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-b.done:
		return ErrClosed
	}
}

// Subscribe returns a receive-only message channel.
func (b *Bus) Subscribe() <-chan message.Message {
	return b.ch
}

// Close closes the bus, preventing further publishes.
func (b *Bus) Close() {
	b.once.Do(func() {
		close(b.done)
	})
}

// Done returns a channel that is closed when the bus is shut down.
func (b *Bus) Done() <-chan struct{} {
	return b.done
}
