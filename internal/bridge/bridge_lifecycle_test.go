package bridge_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/source"
	"github.com/devicebridge/device-bridge/internal/source/scanner"
	"github.com/devicebridge/device-bridge/internal/websocket"
	gorilla "github.com/gorilla/websocket"
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

	recvCh := make(chan message.Message, len(messages))
	client := &chanClient{ch: recvCh}

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

	var received []message.Message
	for i := 0; i < len(messages); i++ {
		select {
		case msg := <-recvCh:
			received = append(received, msg)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds")
	}

	if len(received) != len(messages) {
		t.Fatalf("expected %d messages, got %d", len(messages), len(received))
	}
}

type chanClient struct {
	ch chan message.Message
}

func (c *chanClient) Send(msg message.Message) error {
	c.ch <- msg
	return nil
}

func (c *chanClient) Close() {}

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

func TestApplicationNormalShutdown(t *testing.T) {
	b := bridge.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Run(ctx)

	if err != nil {
		t.Fatalf("expected nil on already-cancelled context, got: %v", err)
	}
}

func TestApplicationErrorPropagation(t *testing.T) {
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

	if errors.Is(err, context.Canceled) {
		t.Fatal("source error should not be treated as Canceled")
	}

	if !errors.Is(err, srcErr) {
		t.Fatalf("expected source error, got: %v", err)
	}
}

func TestContextCancellationIsNotError(t *testing.T) {
	b := bridge.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Run(ctx)

	if err != nil {
		t.Fatalf("expected nil on normal cancellation, got: %v", err)
	}
}

func TestSourceCancellationDeterministic(t *testing.T) {
	b := bridge.New()

	blockSrc := &blockSource{unblock: make(chan struct{})}

	b.Registry().Register("block", func() source.Source {
		return blockSrc
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
			t.Fatalf("expected nil on cancellation, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds")
	}
}

type floodSource struct {
	n       int
	started chan struct{}
}

func (s *floodSource) Run(ctx context.Context, out chan<- message.Message) error {
	close(s.started)
	for i := 0; i < s.n; i++ {
		select {
		case out <- message.Message{Payload: "data"}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type blockClient struct {
	ch             chan struct{}
	closeCh        chan struct{}
	sendCalled     chan struct{}
	sendCalledOnce sync.Once
	closeOnce      sync.Once
}

func (c *blockClient) Send(message.Message) error {
	c.sendCalledOnce.Do(func() {
		close(c.sendCalled)
	})
	<-c.ch
	return nil
}

func (c *blockClient) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		close(c.ch)
	})
}

func TestBlockedDownstreamDoesNotHangShutdown(t *testing.T) {
	b := bridge.New()

	blocked := &blockClient{
		ch:         make(chan struct{}),
		closeCh:    make(chan struct{}),
		sendCalled: make(chan struct{}),
	}
	b.Hub().Register(blocked)

	fs := &floodSource{n: 200, started: make(chan struct{})}

	b.Registry().Register("flood", func() source.Source {
		return fs
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	<-fs.started

	select {
	case <-blocked.sendCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("Send was never called")
	}

	cancel()

	b.Hub().Shutdown()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil on cancellation with blocked downstream, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds — blocked downstream caused hang")
	}

	select {
	case <-blocked.closeCh:
	case <-time.After(5 * time.Second):
		t.Fatal("blocked client was not closed within 5 seconds after Shutdown")
	}

	select {
	case <-blocked.sendCalled:
	default:
		t.Fatal("Send was never called")
	}
}

func TestForwarderCompletesBeforeRunReturns(t *testing.T) {
	b := bridge.New()

	n := 50
	msgs := make([]message.Message, n)
	for i := 0; i < n; i++ {
		msgs[i] = message.Message{Source: "s", Payload: fmt.Sprintf("%d", i)}
	}

	client := &chanClient{ch: make(chan message.Message, n)}
	b.Hub().Register(client)

	b.Registry().Register("s", func() source.Source {
		return &publishSource{msgs: msgs}
	})

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	received := make(map[string]bool)
	for i := 0; i < n; i++ {
		select {
		case msg := <-client.ch:
			received[msg.Payload] = true
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out after %d messages", i)
		}
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5 seconds")
	}

	if len(received) != n {
		t.Fatalf("expected %d messages, got %d", n, len(received))
	}
}

func TestNilSourceSkipped(t *testing.T) {
	b := bridge.New()

	b.Registry().Register("nil", func() source.Source {
		return nil
	})

	b.Registry().Register("ok", func() source.Source {
		return &publishSource{msgs: []message.Message{{Payload: "data"}}}
	})

	client := &chanClient{ch: make(chan message.Message, 1)}
	b.Hub().Register(client)

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	select {
	case msg := <-client.ch:
		if msg.Payload != "data" {
			t.Fatalf("unexpected payload: %q", msg.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive message from ok source")
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

func TestMultipleCancellation(t *testing.T) {
	b := bridge.New()

	blockSrc := &blockSource{unblock: make(chan struct{})}
	b.Registry().Register("block", func() source.Source {
		return blockSrc
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	cancel()
	cancel()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestConcurrentCancelAndSourceError(t *testing.T) {
	b := bridge.New()

	srcErr := errors.New("source error")

	srcReady := make(chan struct{})

	b.Registry().Register("err", func() source.Source {
		go func() {
			<-srcReady
		}()
		return &concurrentErrSource{
			err:      srcErr,
			srcReady: srcReady,
		}
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	<-srcReady
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return")
	}
}

type concurrentErrSource struct {
	err      error
	srcReady chan struct{}
}

func (s *concurrentErrSource) Run(ctx context.Context, _ chan<- message.Message) error {
	close(s.srcReady)
	<-ctx.Done()
	return s.err
}

func TestHTTPServerGracefulShutdown(t *testing.T) {
	b := bridge.New()

	srv := server.New()
	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	httpServer := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: srv.Handler(),
	}

	go func() {
		httpServer.ListenAndServe()
	}()

	b.Registry().Register("s", func() source.Source {
		return &publishSource{msgs: []message.Message{{Payload: "hello"}}}
	})

	ctx, cancel := context.WithCancel(context.Background())

	bridgeDone := make(chan error, 1)
	go func() {
		bridgeDone <- b.Run(ctx)
	}()

	select {
	case err := <-bridgeDone:
		if err != nil {
			t.Fatalf("unexpected bridge error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not finish")
	}

	b.Hub().Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("http shutdown error: %v", err)
	}

	cancel()
}

func TestFullApplicationLifecycle(t *testing.T) {
	b := bridge.New()

	srv := server.New()
	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	ts := httptest.NewUnstartedServer(srv.Handler())
	ts.EnableHTTP2 = false
	ts.Start()
	defer ts.Close()

	input := make(chan scanner.Input, 2)
	b.Registry().Register("scanner", func() source.Source {
		return scanner.New("scanner-main", input)
	})

	ctx, cancel := context.WithCancel(context.Background())

	bridgeDone := make(chan error, 1)
	go func() {
		bridgeDone <- b.Run(ctx)
	}()

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)
	for b.Hub().Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	input <- scanner.Input{Value: "hello-world"}
	close(input)

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("did not receive message: %v", err)
	}

	var received message.Message
	json.Unmarshal(data, &received)

	if received.Payload != "hello-world" {
		t.Fatalf("unexpected payload: %q", received.Payload)
	}

	cancel()

	select {
	case err := <-bridgeDone:
		if err != nil {
			t.Fatalf("unexpected bridge error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not finish")
	}

	b.Hub().Shutdown()
}
