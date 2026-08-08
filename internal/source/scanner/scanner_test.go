package scanner

import (
	"fmt"
	"testing"
	"time"

	"github.com/devicebridge/device-bridge/internal/bus"
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

func TestPublish(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 1)

	s := New("scanner-main", input, b)

	before := time.Now().UnixMilli()

	input <- Input{Value: "1234567890123"}
	close(input)

	if err := s.Run(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := <-b.Subscribe()

	if msg.Source != "scanner-main" {
		t.Fatalf("expected source scanner-main, got %q", msg.Source)
	}

	if msg.Payload != "1234567890123" {
		t.Fatalf("expected payload 1234567890123, got %q", msg.Payload)
	}

	if msg.Timestamp < before || msg.Timestamp > time.Now().UnixMilli() {
		t.Fatalf("timestamp out of range: %d", msg.Timestamp)
	}
}

func TestMultipleMessages(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 3)

	s := New("scanner-main", input, b)

	values := []string{
		"1234567890123",
		"1234567890456",
		"1234567890789",
	}

	for _, v := range values {
		input <- Input{Value: v}
	}
	close(input)

	if err := s.Run(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, expected := range values {
		msg := <-b.Subscribe()

		if msg.Payload != expected {
			t.Fatalf("message %d: expected payload %q, got %q", i, expected, msg.Payload)
		}
	}

	select {
	case msg := <-b.Subscribe():
		t.Fatalf("unexpected extra message: %+v", msg)
	default:
	}
}

func TestSourceID(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 2)

	s := New("scanner-main", input, b)

	input <- Input{Value: "AAA"}
	input <- Input{Value: "BBB"}
	close(input)

	if err := s.Run(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 2; i++ {
		msg := <-b.Subscribe()
		if msg.Source != "scanner-main" {
			t.Fatalf("message %d: expected source scanner-main, got %q", i, msg.Source)
		}
	}
}

func TestInputError(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 1)

	s := New("scanner-main", input, b)

	expectedErr := fmt.Errorf("input error")
	input <- Input{Err: expectedErr}
	close(input)

	err := s.Run(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCompletion(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 1)

	s := New("scanner-main", input, b)

	input <- Input{Value: "test"}
	close(input)

	if err := s.Run(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := <-b.Subscribe()
	if msg.Payload != "test" {
		t.Fatalf("unexpected payload: %q", msg.Payload)
	}

	select {
	case msg := <-b.Subscribe():
		t.Fatalf("unexpected extra message: %+v", msg)
	default:
	}
}

func TestEmptyValue(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 2)

	s := New("scanner-main", input, b)

	input <- Input{Value: ""}
	input <- Input{Value: "valid"}
	close(input)

	if err := s.Run(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := <-b.Subscribe()
	if msg.Payload != "valid" {
		t.Fatalf("expected payload valid, got %q", msg.Payload)
	}

	select {
	case msg := <-b.Subscribe():
		t.Fatalf("unexpected extra message: %+v", msg)
	default:
	}
}

func TestRegistryIntegration(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input, 1)

	registry := source.NewRegistry()

	factory := func() source.Source {
		return New("scanner-main", input, b)
	}

	if err := registry.Register("scanner-main", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	src, err := registry.Create("scanner-main")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if src == nil {
		t.Fatal("source is nil")
	}

	var _ source.Source = src

	input <- Input{Value: "12345"}
	close(input)

	if err := src.Run(nil); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	msg := <-b.Subscribe()
	if msg.Payload != "12345" {
		t.Fatalf("unexpected payload: %q", msg.Payload)
	}
}

func TestImplementsSourceInterface(t *testing.T) {
	b := bus.New(10)
	input := make(chan Input)

	var _ message.Message = message.Message{}

	s := New("scanner-main", input, b)

	var _ source.Source = s
}
