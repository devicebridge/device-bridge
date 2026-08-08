package hub

import (
	"errors"
	"testing"

	"github.com/devicebridge/device-bridge/internal/message"
)

// mockClient is a test implementation of Client.
type mockClient struct {
	last message.Message
	err  error
}

func (m *mockClient) Send(msg message.Message) error {
	m.last = msg
	return m.err
}

func TestNew(t *testing.T) {
	h := New()

	if h == nil {
		t.Fatal("hub is nil")
	}

	if h.clients == nil {
		t.Fatal("clients map is nil")
	}
}

func TestRegister(t *testing.T) {
	h := New()

	c := &mockClient{}

	h.Register(c)

	if len(h.clients) != 1 {
		t.Fatal("client was not registered")
	}
}

func TestUnregister(t *testing.T) {
	h := New()

	c := &mockClient{}

	h.Register(c)
	h.Unregister(c)

	if len(h.clients) != 0 {
		t.Fatal("client was not removed")
	}
}

func TestBroadcast(t *testing.T) {
	h := New()

	c := &mockClient{}
	h.Register(c)

	msg := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "12345",
	}

	if err := h.Broadcast(msg); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	if c.last != msg {
		t.Fatal("message mismatch")
	}
}

func TestBroadcastMultipleClients(t *testing.T) {
	h := New()

	c1 := &mockClient{}
	c2 := &mockClient{}

	h.Register(c1)
	h.Register(c2)

	msg := message.Message{Payload: "ABC"}

	if err := h.Broadcast(msg); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	if c1.last != msg {
		t.Fatal("first client did not receive message")
	}

	if c2.last != msg {
		t.Fatal("second client did not receive message")
	}
}

func TestBroadcastError(t *testing.T) {
	h := New()

	expected := errors.New("send failed")

	c := &mockClient{
		err: expected,
	}

	h.Register(c)

	err := h.Broadcast(message.Message{})

	if !errors.Is(err, expected) {
		t.Fatal("unexpected error")
	}
}

func TestConcurrentAccess(t *testing.T) {
	h := New()

	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			c := &mockClient{}
			h.Register(c)
			h.Unregister(c)
		}
		close(done)
	}()

	for i := 0; i < 1000; i++ {
		_ = h.Broadcast(message.Message{})
	}

	<-done
}
