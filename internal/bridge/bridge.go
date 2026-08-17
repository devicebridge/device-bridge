package bridge

import (
	"errors"
	"sync"

	"github.com/devicebridge/device-bridge/internal/bus"
	"github.com/devicebridge/device-bridge/internal/hub"
	"github.com/devicebridge/device-bridge/internal/source"
)

var ErrAlreadyRunning = errors.New("bridge runtime has already been started")

// Bridge coordinates application components.
type Bridge struct {
	registry *source.Registry
	bus      *bus.Bus
	hub      *hub.Hub
	runOnce  sync.Once
}

// New creates a new Bridge instance.
func New() *Bridge {
	return &Bridge{
		registry: source.NewRegistry(),
		bus:      bus.New(100),
		hub:      hub.New(),
	}
}

// Hub returns the message hub.
func (b *Bridge) Hub() *hub.Hub {
	return b.hub
}

// Registry returns the source registry.
func (b *Bridge) Registry() *source.Registry {
	return b.registry
}
