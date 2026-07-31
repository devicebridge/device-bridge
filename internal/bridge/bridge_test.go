package bridge

import (
	"testing"

	"github.com/devicebridge/device-bridge/internal/message"
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

// runtimeMockClient is a test implementation of hub.Client.
type runtimeMockClient struct {
	received chan message.Message
}

func (c *runtimeMockClient) Send(msg message.Message) error {
	c.received <- msg
	return nil
}

func TestRun(t *testing.T) {
	b := New()

	client := &runtimeMockClient{
		received: make(chan message.Message, 1),
	}

	b.hub.Register(client)

	go b.Run()

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "1234567890",
	}

	b.bus.Publish(expected)

	actual := <-client.received

	if actual != expected {
		t.Fatal("unexpected message")
	}
}
