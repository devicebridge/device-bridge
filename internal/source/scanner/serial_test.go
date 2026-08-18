package scanner

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type serialTestPort struct {
	reader io.Reader
	closed chan struct{}
	once   sync.Once
}

func newSerialTestPort(reader io.Reader) *serialTestPort {
	return &serialTestPort{reader: reader, closed: make(chan struct{})}
}

func (p *serialTestPort) Read(buf []byte) (int, error) {
	select {
	case <-p.closed:
		return 0, io.ErrClosedPipe
	default:
		return p.reader.Read(buf)
	}
}

func (p *serialTestPort) Close() error {
	p.once.Do(func() { close(p.closed) })
	return nil
}

func TestSerialAdapterFramesLines(t *testing.T) {
	port := newSerialTestPort(strings.NewReader("first\r\nsecond\npartial"))
	a := NewSerialAdapter(port, 4)

	done := make(chan error, 1)
	go func() { done <- a.Run(context.Background()) }()

	var values []string
	for i := 0; i < 3; i++ {
		select {
		case input := <-a.Input():
			values = append(values, input.Value)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for serial input")
		}
	}

	if err := <-done; err != nil {
		t.Fatalf("unexpected adapter error: %v", err)
	}
	if strings.Join(values, ",") != "first,second,partial" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestSerialAdapterReturnsPortError(t *testing.T) {
	expected := errors.New("port failed")
	port := newSerialTestPort(errorReader{err: expected})
	a := NewSerialAdapter(port, 1)

	if err := a.Run(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("expected port error, got %v", err)
	}
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestSerialAdapterCancellationClosesPort(t *testing.T) {
	port := newInterruptiblePort()
	a := NewSerialAdapter(port, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("adapter did not stop after cancellation")
	}
}

type interruptiblePort struct {
	closed chan struct{}
	once   sync.Once
}

func newInterruptiblePort() *interruptiblePort {
	return &interruptiblePort{closed: make(chan struct{})}
}

func (p *interruptiblePort) Read([]byte) (int, error) {
	<-p.closed
	return 0, io.ErrClosedPipe
}

func (p *interruptiblePort) Close() error {
	p.once.Do(func() { close(p.closed) })
	return nil
}
