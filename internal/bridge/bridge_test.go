package bridge

import "testing"

func TestNew(t *testing.T) {
	b := New()

	if b == nil {
		t.Fatal("bridge is nil")
	}

	if b.registry == nil {
		t.Fatal("registry is nil")
	}

	if b.bus == nil {
		t.Fatal("bus is nil")
	}

	if b.hub == nil {
		t.Fatal("hub is nil")
	}
}
