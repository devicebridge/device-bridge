package scanner

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReconnectingAdapterRetriesFactory(t *testing.T) {
	var attempts int32
	adapter := NewReconnectingAdapter(func() (io.ReadCloser, error) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			return nil, errors.New("not ready")
		}
		return io.NopCloser(strings.NewReader("value\n")), nil
	}, 0, 1)

	done := make(chan error, 1)
	go func() { done <- adapter.Run(context.Background()) }()

	select {
	case input := <-adapter.Input():
		if input.Value != "value" {
			t.Fatalf("unexpected value: %q", input.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconnect input")
	}
	adapter.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("adapter did not stop after Close")
	}
}
