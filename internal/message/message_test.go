package message

import (
	"encoding/json"
	"testing"
)

func TestMarshalMessage(t *testing.T) {
	msg := Message{
		Source:    "scanner-main",
		Timestamp: 1785472345123,
		Payload:   "1234567890",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	expected := `{"source":"scanner-main","timestamp":1785472345123,"payload":"1234567890"}`

	if string(data) != expected {
		t.Fatalf("unexpected json:\nexpected: %s\nactual:   %s", expected, string(data))
	}
}

func TestUnmarshalMessage(t *testing.T) {
	input := `{"source":"scanner-main","timestamp":1785472345123,"payload":"1234567890"}`

	var msg Message

	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if msg.Source != "scanner-main" {
		t.Fatalf("unexpected source")
	}

	if msg.Timestamp != 1785472345123 {
		t.Fatalf("unexpected timestamp")
	}

	if msg.Payload != "1234567890" {
		t.Fatalf("unexpected payload")
	}
}
