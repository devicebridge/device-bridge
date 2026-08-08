package websocket

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

func TestNewHandler(t *testing.T) {
	h := hub.New()
	handler := NewHandler(h)

	if handler == nil {
		t.Fatal("handler is nil")
	}
}

func TestHandlerE2E(t *testing.T) {
	h := hub.New()
	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "12345",
	}

	if err := h.Broadcast(expected); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	_, data, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var received message.Message
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if received.Source != expected.Source {
		t.Fatalf("source mismatch: expected %q, got %q", expected.Source, received.Source)
	}

	if received.Timestamp != expected.Timestamp {
		t.Fatalf("timestamp mismatch: expected %d, got %d", expected.Timestamp, received.Timestamp)
	}

	if received.Payload != expected.Payload {
		t.Fatalf("payload mismatch: expected %q, got %q", expected.Payload, received.Payload)
	}
}

func TestHandlerE2EMultipleMessages(t *testing.T) {
	h := hub.New()
	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	messages := []message.Message{
		{Source: "scanner-1", Timestamp: 1000, Payload: "AAA"},
		{Source: "scanner-2", Timestamp: 2000, Payload: "BBB"},
		{Source: "scanner-3", Timestamp: 3000, Payload: "CCC"},
	}

	for _, expected := range messages {
		if err := h.Broadcast(expected); err != nil {
			t.Fatalf("broadcast failed: %v", err)
		}

		_, data, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}

		var received message.Message
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if received != expected {
			t.Fatalf("message mismatch:\nexpected: %+v\nactual:   %+v", expected, received)
		}
	}
}
