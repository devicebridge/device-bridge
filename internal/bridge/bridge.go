package bridge

import "github.com/devicebridge/device-bridge/internal/source"

// Bridge coordinates application components.
type Bridge struct {
	registry *source.Registry
}

// New creates a new Bridge instance.
func New() *Bridge {
	return &Bridge{
		registry: source.NewRegistry(),
	}
}
