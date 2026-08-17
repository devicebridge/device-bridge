package source

import (
	"fmt"
	"sync"
)

// Factory creates a new Source instance.
type Factory func() Source

// Registry stores registered source factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry creates an empty source registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register registers a source factory by name.
func (r *Registry) Register(name string, factory Factory) error {
	if name == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	if factory == nil {
		return fmt.Errorf("factory cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("source %q already registered", name)
	}

	r.factories[name] = factory
	return nil
}

// Create creates a source by its registered name.
func (r *Registry) Create(name string) (Source, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown source %q", name)
	}

	return factory(), nil
}

// Names returns all registered source names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))

	for name := range r.factories {
		names = append(names, name)
	}

	return names
}
