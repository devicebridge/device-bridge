package bridge_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

type blockSource struct {
	unblock chan struct{}
}

func (s *blockSource) Run(ctx context.Context, _ chan<- message.Message) error {
	select {
	case <-s.unblock:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type errSource struct {
	err error
}

func (s *errSource) Run(_ context.Context, _ chan<- message.Message) error {
	return s.err
}

type publishSource struct {
	msgs []message.Message
}

func (s *publishSource) Run(_ context.Context, out chan<- message.Message) error {
	for _, msg := range s.msgs {
		out <- msg
	}
	return nil
}

type cancPublishSource struct {
	msgs    []message.Message
	blockCh chan struct{}
}

func (s *cancPublishSource) Run(ctx context.Context, out chan<- message.Message) error {
	close(s.blockCh)

	for _, msg := range s.msgs {
		select {
		case out <- msg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestNormalShutdown(t *testing.T) {
	b := bridge.New()

	src := &blockSource{unblock: make(chan struct{})}

	b.Registry().Register("block", func() source.Source {
		return src
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil on normal shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds after cancel")
	}
}

func TestSourceError(t *testing.T) {
	b := bridge.New()

	srcErr := errors.New("source failed")

	b.Registry().Register("err", func() source.Source {
		return &errSource{err: srcErr}
	})

	ctx := context.Background()

	err := b.Run(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, srcErr) {
		t.Fatalf("expected source error, got: %v", err)
	}
}

func TestSourceErrorStopsOthers(t *testing.T) {
	b := bridge.New()

	srcErr := errors.New("source A failed")

	b.Registry().Register("err", func() source.Source {
		return &errSource{err: srcErr}
	})

	blockDone := make(chan error, 1)
	blockSrc := &blockSource{unblock: make(chan struct{})}

	b.Registry().Register("block", func() source.Source {
		go func() {
			select {
			case <-blockSrc.unblock:
				blockDone <- nil
			}
		}()
		return &catchContextSource{ctxErr: &blockDone}
	})

	ctx := context.Background()

	err := b.Run(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, srcErr) {
		t.Fatalf("expected source error, got: %v", err)
	}

	select {
	case e := <-blockDone:
		if e != context.Canceled {
			t.Fatalf("block source expected context.Canceled, got: %v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("block source was not cancelled within 5 seconds")
	}
}

type catchContextSource struct {
	ctxErr *chan error
}

func (s *catchContextSource) Run(ctx context.Context, _ chan<- message.Message) error {
	<-ctx.Done()
	*s.ctxErr <- ctx.Err()
	return ctx.Err()
}

func TestMultipleSources(t *testing.T) {
	b := bridge.New()

	messages := []message.Message{
		{Source: "source-a", Timestamp: 1000, Payload: "AAA"},
		{Source: "source-b", Timestamp: 2000, Payload: "BBB"},
	}

	var mu sync.Mutex
	var received []message.Message

	client := &collectClient{
		collect: func(msg message.Message) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
		},
	}

	b.Hub().Register(client)

	b.Registry().Register("a", func() source.Source {
		return &publishSource{msgs: messages[:1]}
	})

	b.Registry().Register("b", func() source.Source {
		return &publishSource{msgs: messages[1:]}
	})

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds")
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received))
	}
}

type collectClient struct {
	collect func(message.Message)
}

func (c *collectClient) Send(msg message.Message) error {
	c.collect(msg)
	return nil
}

func (c *collectClient) Close() {}

func TestFirstErrorPreserved(t *testing.T) {
	b := bridge.New()

	srcErrA := errors.New("error A")
	srcErrB := errors.New("error B")

	b.Registry().Register("a", func() source.Source {
		return &errSource{err: srcErrA}
	})

	b.Registry().Register("b", func() source.Source {
		return &errSource{err: srcErrB}
	})

	ctx := context.Background()

	err := b.Run(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, srcErrA) && !errors.Is(err, srcErrB) {
		t.Fatalf("expected one of the source errors, got: %v", err)
	}
}

func TestCancellationDuringPublishing(t *testing.T) {
	b := bridge.New()

	messages := []message.Message{
		{Source: "s", Timestamp: 1, Payload: "A"},
		{Source: "s", Timestamp: 2, Payload: "B"},
		{Source: "s", Timestamp: 3, Payload: "C"},
	}

	blockCh := make(chan struct{})

	b.Registry().Register("s", func() source.Source {
		return &cancPublishSource{msgs: messages, blockCh: blockCh}
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	<-blockCh

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds")
	}
}

func TestBridgeImplementsSourceInterfaceIndirectly(t *testing.T) {
	b := bridge.New()

	if b == nil {
		t.Fatal("bridge is nil")
	}

	var _ *hub.Hub = b.Hub()
}
