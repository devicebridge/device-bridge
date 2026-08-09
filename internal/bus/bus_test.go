package bus

import (
	"context"
	"errors"
	"testing"

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
