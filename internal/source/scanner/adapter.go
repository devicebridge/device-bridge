package scanner

import (
	"context"
	"errors"
	"sync"
)

var ErrAdapterClosed = errors.New("scanner input adapter is closed")

// ChannelAdapter is the transport-neutral input boundary for a Scanner.
// Physical device adapters can publish into the same channel contract.
type ChannelAdapter struct {
	input chan Input
	done  chan struct{}
	once  sync.Once
}

func NewChannelAdapter(buffer int) *ChannelAdapter {
	if buffer < 0 {
		panic("scanner adapter buffer cannot be negative")
	}
	return &ChannelAdapter{
		input: make(chan Input, buffer),
		done:  make(chan struct{}),
	}
}

func (a *ChannelAdapter) Input() <-chan Input {
	return a.input
}

func (a *ChannelAdapter) Publish(ctx context.Context, input Input) error {
	select {
	case <-a.done:
		return ErrAdapterClosed
	default:
	}

	select {
	case a.input <- input:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-a.done:
		return ErrAdapterClosed
	}
}

func (a *ChannelAdapter) Close() {
	a.once.Do(func() { close(a.done) })
}
