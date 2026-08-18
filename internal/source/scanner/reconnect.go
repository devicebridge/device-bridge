package scanner

import (
	"context"
	"io"
	"sync"
	"time"
)

// PortFactory opens a serial-compatible port. Implementations may use OS
// serial libraries; tests can provide in-memory or PTY-backed factories.
type PortFactory func() (io.ReadCloser, error)

// ReconnectingAdapter retries opening and reading a port until cancellation.
// It preserves the line framing behavior of SerialAdapter.
type ReconnectingAdapter struct {
	factory PortFactory
	delay   time.Duration
	input   *ChannelAdapter
	once    sync.Once
}

func NewReconnectingAdapter(factory PortFactory, delay time.Duration, buffer int) *ReconnectingAdapter {
	return &ReconnectingAdapter{factory: factory, delay: delay, input: NewChannelAdapter(buffer)}
}

func (a *ReconnectingAdapter) Input() <-chan Input { return a.input.Input() }

func (a *ReconnectingAdapter) Run(ctx context.Context) error {
	defer a.input.Close()
	for {
		port, err := a.factory()
		if err != nil {
			if err := waitReconnect(ctx, a.delay); err != nil {
				return err
			}
			continue
		}

		serial := NewSerialAdapter(port, 100)
		serialDone := make(chan error, 1)
		go func() { serialDone <- serial.Run(ctx) }()
		for {
			select {
			case input := <-serial.Input():
				if publishErr := a.input.Publish(ctx, input); publishErr != nil {
					serial.Close()
					return publishErr
				}
			case err = <-serialDone:
				goto serialFinished
			case <-a.input.Done():
				serial.Close()
				return ErrAdapterClosed
			}
		}
	serialFinished:
		serial.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := waitReconnect(ctx, a.delay); err != nil {
			return err
		}
	}
}

func (a *ReconnectingAdapter) Close() {
	a.once.Do(func() { a.input.Close() })
}

func waitReconnect(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
