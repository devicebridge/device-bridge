package bridge_test

import (
	"encoding/json"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bridge"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/server"
	"github.com/devicebridge/device-bridge/internal/websocket"
	gorilla "github.com/gorilla/websocket"
)

func TestBridgeHTTPIntegration(t *testing.T) {
	b := bridge.New()

	srv := server.New()

	wsHandler := websocket.NewHandler(b.Hub())
	srv.Handle("/ws", wsHandler)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	go b.Run()

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if b.Hub().Count() > 0 {
			break
		}
		runtime.Gosched()
	}

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "12345",
	}

	if err := b.Hub().Broadcast(expected); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	_, data, err := conn.ReadMessage()
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

	conn.Close()

	for i := 0; i < 50; i++ {
		if b.Hub().Count() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if b.Hub().Count() != 0 {
		t.Fatal("client was not unregistered")
	}
}
