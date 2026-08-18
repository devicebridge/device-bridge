package hub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/message"
)

// mockClient is a test implementation of Client.
type mockClient struct {
	last    message.Message
	err     error
	onClose func()
}

func (m *mockClient) Send(msg message.Message) error {
	m.last = msg
	return m.err
}

func (m *mockClient) Close() {
	if m.onClose != nil {
		m.onClose()
	}
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

func TestBroadcastRemovesFailedClient(t *testing.T) {
	h := New()
	expected := errors.New("send failed")
	failed := &mockClient{err: expected}
	healthy := &mockClient{}
	h.Register(failed)
	h.Register(healthy)

	err := h.Broadcast(message.Message{Payload: "first"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected send error, got %v", err)
	}
	if h.Count() != 1 {
		t.Fatalf("expected failed client to be removed, got %d clients", h.Count())
	}

	if err := h.Broadcast(message.Message{Payload: "second"}); err != nil {
		t.Fatalf("healthy client should receive later broadcasts, got %v", err)
	}
	if healthy.last.Payload != "second" {
		t.Fatalf("healthy client received %q, want %q", healthy.last.Payload, "second")
	}
}

func TestBroadcastFailureAndShutdownCloseClientOnce(t *testing.T) {
	h := New()
	client := &closeCountingClient{err: errors.New("send failed")}
	h.Register(client)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = h.Broadcast(message.Message{})
	}()
	go func() {
		defer wg.Done()
		h.Shutdown()
	}()
	wg.Wait()

	if got := atomic.LoadInt32(&client.closed); got != 1 {
		t.Fatalf("expected one Close call, got %d", got)
	}
}

type closeCountingClient struct {
	err    error
	closed int32
}

func (c *closeCountingClient) Send(message.Message) error {
	return c.err
}

func (c *closeCountingClient) Close() {
	atomic.AddInt32(&c.closed, 1)
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

func TestShutdown(t *testing.T) {
	h := New()

	closed := make(chan struct{})
	c := &mockClient{
		onClose: func() {
			close(closed)
		},
	}

	h.Register(c)
	h.Shutdown()

	<-closed

	if h.Count() != 0 {
		t.Fatal("shutdown should remove clients from the registry")
	}

	select {
	case <-h.Done():
	default:
		t.Fatal("Done channel should be closed after Shutdown")
	}
}

func TestShutdownIdempotent(t *testing.T) {
	h := New()

	c := &mockClient{}
	h.Register(c)

	h.Shutdown()
	h.Shutdown()
	h.Shutdown()

	// verify no panic
}

func TestRegisterAfterShutdown(t *testing.T) {
	h := New()

	h.Shutdown()

	c := &mockClient{}
	h.Register(c)

	if h.Count() != 0 {
		t.Fatal("client should not be registered after shutdown")
	}
}

func TestBroadcastAfterShutdown(t *testing.T) {
	h := New()

	c := &mockClient{}
	h.Register(c)

	h.Shutdown()

	msg := message.Message{Payload: "test"}

	if err := h.Broadcast(msg); err != nil {
		t.Fatalf("broadcast should return nil after shutdown, got: %v", err)
	}
}

func TestWaitForCount(t *testing.T) {
	h := New()

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done <- h.WaitForCount(ctx, 1)
	}()

	c := &mockClient{}
	h.Register(c)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForCount did not return after Register")
	}
}

func TestWaitForCountAlreadySatisfied(t *testing.T) {
	h := New()

	c := &mockClient{}
	h.Register(c)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.WaitForCount(ctx, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForCountShutdown(t *testing.T) {
	h := New()

	done := make(chan error, 1)
	go func() {
		ctx := context.Background()
		done <- h.WaitForCount(ctx, 1)
	}()

	h.Shutdown()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after shutdown, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForCount did not return after Shutdown")
	}
}

func TestWaitForCountStress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		h := New()

		done := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			done <- h.WaitForCount(ctx, 1)
		}()

		c := &mockClient{}
		h.Register(c)

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("iteration %d: unexpected error: %v", i, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("iteration %d: WaitForCount did not return", i)
		}
	}
}
