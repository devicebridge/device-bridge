package bridge

import (
	"context"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

func TestNew(t *testing.T) {
	b := New()

	if b == nil {
		t.Fatal("bridge is nil")
	}

	if b.registry == nil {
		t.Fatal("registry is nil")
	}

	if b.bus == nil {
		t.Fatal("bus is nil")
	}

	if b.hub == nil {
		t.Fatal("hub is nil")
	}
}

type runtimeMockClient struct {
	received chan message.Message
}

func (c *runtimeMockClient) Send(msg message.Message) error {
	c.received <- msg
	return nil
}

func (c *runtimeMockClient) Close() {}

type runTestSource struct {
	msg message.Message
}

func (s *runTestSource) Run(_ context.Context, out chan<- message.Message) error {
	out <- s.msg
	return nil
}

func TestRun(t *testing.T) {
	b := New()

	client := &runtimeMockClient{
		received: make(chan message.Message, 1),
	}

	b.hub.Register(client)

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "1234567890",
	}

	b.Registry().Register("test", func() source.Source {
		return &runTestSource{msg: expected}
	})

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	select {
	case actual := <-client.received:
		if actual != expected {
			t.Fatal("unexpected message")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive message")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return")
	}
}
