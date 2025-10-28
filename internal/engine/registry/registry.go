package registry

import (
	"fmt"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Registry manages available engines.
// Thread-safe for concurrent access.
type Registry struct {
	mu        sync.RWMutex
	engines   map[string]core.Engine
	factories map[string]EngineFactory
}

// EngineFactory creates engine instances.
type EngineFactory func() (core.Engine, error)

var globalRegistry = &Registry{
	engines:   make(map[string]core.Engine),
	factories: make(map[string]EngineFactory),
}

// Global returns the global engine registry.
func Global() *Registry {
	return globalRegistry
}

// Register registers an engine factory.
// The factory will be called lazily when Get() is first called.
func (r *Registry) Register(name string, factory EngineFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("engine %q already registered", name)
	}

	r.factories[name] = factory
	return nil
}

// Get retrieves an engine by name.
// Creates the engine on first access (lazy initialization).
func (r *Registry) Get(name string) (core.Engine, error) {
	// Fast path: check if already created
	r.mu.RLock()
	if engine, ok := r.engines[name]; ok {
		r.mu.RUnlock()
		return engine, nil
	}
	r.mu.RUnlock()

	// Slow path: create engine
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if engine, ok := r.engines[name]; ok {
		return engine, nil
	}

	// Look up factory
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("engine %q not registered", name)
	}

	// Create engine
	engine, err := factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create engine %q: %w", name, err)
	}

	r.engines[name] = engine
	return engine, nil
}

// List returns all registered engine names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// MustRegister registers an engine factory and panics on error.
// Useful for init() functions.
func MustRegister(name string, factory EngineFactory) {
	if err := Global().Register(name, factory); err != nil {
		panic(err)
	}
}
