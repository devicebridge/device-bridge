package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

func TestNewClient(t *testing.T) {
	var serverConn *websocket.Conn
	ready := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		serverConn = conn
		close(ready)

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		conn.ReadMessage()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	<-ready

	c := NewClient(clientConn)

	if c == nil {
		t.Fatal("client is nil")
	}

	serverConn.Close()
}

func TestSend(t *testing.T) {
	msgCh := make(chan message.Message, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Error(err)
			return
		}

		var msg message.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Error(err)
			return
		}

		msgCh <- msg
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	c := NewClient(clientConn)

	expected := message.Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "12345",
	}

	if err := c.Send(expected); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	received := <-msgCh

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
