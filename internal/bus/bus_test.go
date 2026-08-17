package bus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/message"
)

func TestNew(t *testing.T) {
	b := New(1)

	if b == nil {
		t.Fatal("bus is nil")
	}
}

func TestPublishSubscribe(t *testing.T) {
	b := New(1)

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "1234567890",
	}

	b.Publish(expected)

	actual := <-b.Subscribe()

	if actual != expected {
		t.Fatalf("unexpected message")
	}
}

func TestBuffer(t *testing.T) {
	b := New(2)

	b.Publish(message.Message{})
	b.Publish(message.Message{})
}

func TestMessageOrder(t *testing.T) {
	b := New(3)

	b.Publish(message.Message{Payload: "1"})
	b.Publish(message.Message{Payload: "2"})
	b.Publish(message.Message{Payload: "3"})

	ch := b.Subscribe()

	if (<-ch).Payload != "1" {
		t.Fatal("unexpected first message")
	}

	if (<-ch).Payload != "2" {
		t.Fatal("unexpected second message")
	}

	if (<-ch).Payload != "3" {
		t.Fatal("unexpected third message")
	}
}

func TestPublishCtx(t *testing.T) {
	b := New(1)

	expected := message.Message{
		Source:    "test",
		Timestamp: 1000,
		Payload:   "data",
	}

	if err := b.PublishCtx(context.Background(), expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actual := <-b.Subscribe()

	if actual != expected {
		t.Fatalf("unexpected message")
	}
}

func TestPublishCtxCancelled(t *testing.T) {
	b := New(0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.PublishCtx(ctx, message.Message{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	b := New(1)

	b.Close()
	b.Close()

	select {
	case <-b.Done():
	default:
		t.Fatal("Done should be closed after Close")
	}
}

func TestPublishCtxAfterClose(t *testing.T) {
	b := New(0)
	b.Close()

	err := b.PublishCtx(context.Background(), message.Message{})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestBlockedPublishUnblocksOnClose(t *testing.T) {
	b := New(0)
	done := make(chan struct{})

	go func() {
		b.Publish(message.Message{})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Publish returned before the bus was closed")
	case <-time.After(10 * time.Millisecond):
	}

	b.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish did not unblock after Close")
	}
}

func TestConcurrentPublishAndClose(t *testing.T) {
	b := New(1)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.PublishCtx(context.Background(), message.Message{})
		}()
	}

	b.Close()
	wg.Wait()
}
