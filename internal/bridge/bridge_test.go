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
}
