package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestCloseDoesNotWaitForMutex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		conn.ReadMessage()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	c := NewClient(clientConn)

	c.mu.Lock()

	closeDone := make(chan struct{})
	go func() {
		c.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Close() blocked on mutex held by Send")
	}

	c.mu.Unlock()
}

func TestCloseIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		conn.ReadMessage()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	c := NewClient(clientConn)

	c.Close()
	c.Close()
	c.Close()
}

func TestSendAfterClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		conn.ReadMessage()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	c := NewClient(clientConn)

	c.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Send(message.Message{Payload: "test"})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after close, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() did not return after Close()")
	}
}

func TestConcurrentSendAndClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		conn.ReadMessage()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	c := NewClient(clientConn)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		c.Send(message.Message{Payload: "1"})
	}()

	go func() {
		defer wg.Done()
		c.Send(message.Message{Payload: "2"})
	}()

	go func() {
		defer wg.Done()
		c.Close()
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Send+Close did not complete")
	}
}
