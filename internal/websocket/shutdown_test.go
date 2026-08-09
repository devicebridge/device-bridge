package websocket

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/gorilla/websocket"
)

func TestWebSocketShutdown(t *testing.T) {
	h := hub.New()

	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for h.Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered within 5 seconds")
		}
		time.Sleep(10 * time.Millisecond)
	}

	h.Shutdown()

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected read error after shutdown, got nil")
	}

	deadline = time.Now().Add(5 * time.Second)
	for h.Count() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not unregistered within 5 seconds after shutdown")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWebSocketShutdownMultipleClients(t *testing.T) {
	h := hub.New()

	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn1, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()

	deadline := time.Now().Add(5 * time.Second)
	for h.Count() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("expected 2 clients, got %d", h.Count())
		}
		time.Sleep(10 * time.Millisecond)
	}

	h.Shutdown()

	conn1.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn1.ReadMessage()
	if err == nil {
		t.Fatal("expected read error on conn1 after shutdown")
	}

	conn2.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn2.ReadMessage()
	if err == nil {
		t.Fatal("expected read error on conn2 after shutdown")
	}

	deadline = time.Now().Add(5 * time.Second)
	for h.Count() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("clients were not unregistered, remaining: %d", h.Count())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	h := hub.New()

	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)
	for h.Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	h.Shutdown()
	h.Shutdown()
	h.Shutdown()
	h.Shutdown()

	// no panic
}

func TestBroadcastDuringShutdown(t *testing.T) {
	h := hub.New()

	handler := NewHandler(h)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)
	for h.Count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			h.Broadcast(message.Message{Payload: "test"})
		}
	}()

	time.Sleep(time.Millisecond)
	h.Shutdown()

	<-done
	// no panic, no race
}
