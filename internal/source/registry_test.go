package source

import (
	"context"
	"testing"

	"github.com/devicebridge/device-bridge/internal/message"
)

type registryMockSource struct{}

func (registryMockSource) Run(context.Context, chan<- message.Message) error {
	return nil
}

func registryMockFactory() Source {
	return registryMockSource{}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("registry is nil")
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("mock", registryMockFactory); err != nil {
		t.Fatalf("register failed: %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("mock", registryMockFactory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := r.Register("mock", registryMockFactory); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestCreate(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("mock", registryMockFactory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	src, err := r.Create("mock")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if src == nil {
		t.Fatal("source is nil")
	}
}

func TestCreateUnknown(t *testing.T) {
	r := NewRegistry()

	if _, err := r.Create("unknown"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("", registryMockFactory); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterNilFactory(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("mock", nil); err == nil {
		t.Fatal("expected error")
	}
}
