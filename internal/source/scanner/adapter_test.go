package scanner

import (
	"context"
	"errors"
	"testing"
)

func TestChannelAdapterPublishesInput(t *testing.T) {
	a := NewChannelAdapter(1)
	defer a.Close()

	if err := a.Publish(context.Background(), Input{Value: "value"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if got := (<-a.Input()).Value; got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestChannelAdapterPublishAfterClose(t *testing.T) {
	a := NewChannelAdapter(1)
	a.Close()

	if err := a.Publish(context.Background(), Input{}); !errors.Is(err, ErrAdapterClosed) {
		t.Fatalf("expected ErrAdapterClosed, got %v", err)
	}

	a.Close()
}

func TestChannelAdapterPublishCancellation(t *testing.T) {
	a := NewChannelAdapter(0)
	defer a.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.Publish(ctx, Input{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
