package bridge_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/source"
	"github.com/devicebridge/device-bridge/internal/source/scanner"
	"github.com/devicebridge/device-bridge/internal/websocket"
	gorilla "github.com/gorilla/websocket"
)

func TestE2ESingleMessage(t *testing.T) {
	b := bridge.New()

	srv := server.New()

	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	input := make(chan scanner.Input, 1)

	b.Registry().Register("scanner-main", func() source.Source {
		return scanner.New("scanner-main", input)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Run(ctx)

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)

	for b.Hub().Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered within 5 seconds")
		}
		time.Sleep(10 * time.Millisecond)
	}

	input <- scanner.Input{Value: "1234567890123"}
	close(input)

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("did not receive expected WebSocket message within 5 seconds: %v", err)
	}

	var received message.Message
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if received.Source != "scanner-main" {
		t.Fatalf("source mismatch: expected scanner-main, got %q", received.Source)
	}

	if received.Payload != "1234567890123" {
		t.Fatalf("payload mismatch: expected 1234567890123, got %q", received.Payload)
	}

	if received.Timestamp == 0 {
		t.Fatal("timestamp not set")
	}
}

func TestE2EMultipleMessages(t *testing.T) {
	b := bridge.New()

	srv := server.New()

	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	values := []string{"1234567890123", "1234567890456", "1234567890789"}

	input := make(chan scanner.Input, len(values))

	b.Registry().Register("scanner-main", func() source.Source {
		return scanner.New("scanner-main", input)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Run(ctx)

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)

	for b.Hub().Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered within 5 seconds")
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, v := range values {
		input <- scanner.Input{Value: v}
	}
	close(input)

	for i, expected := range values {
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("message %d: did not receive message: %v", i, err)
		}

		var received message.Message
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("message %d: unmarshal failed: %v", i, err)
		}

		if received.Source != "scanner-main" {
			t.Fatalf("message %d: source mismatch: expected scanner-main, got %q", i, received.Source)
		}

		if received.Payload != expected {
			t.Fatalf("message %d: payload mismatch: expected %q, got %q", i, expected, received.Payload)
		}

		if received.Timestamp == 0 {
			t.Fatalf("message %d: timestamp not set", i)
		}
	}
}
