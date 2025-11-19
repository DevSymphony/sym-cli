package registry

import (
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Registry manages adapter instances - simple tool name based lookup.
type Registry struct {
	mu sync.RWMutex

	// adapters maps tool name to adapter (e.g., "eslint" -> ESLintAdapter)
	adapters map[string]adapter.Adapter
}

// NewRegistry creates a new empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]adapter.Adapter),
	}
}

// Register adds an adapter to the registry.
// Returns ErrNilAdapter if the adapter is nil.
func (r *Registry) Register(adp adapter.Adapter) error {
	if adp == nil {
		return ErrNilAdapter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := adp.Name()
	r.adapters[name] = adp

	return nil
}

// GetAdapter finds an adapter by tool name (e.g., "eslint", "prettier", "tsc").
// Returns the adapter, or ErrAdapterNotFound if not registered.
func (r *Registry) GetAdapter(toolName string) (adapter.Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adp, ok := r.adapters[toolName]
	if !ok {
		return nil, &ErrAdapterNotFound{ToolName: toolName}
	}

	return adp, nil
}

// GetAll returns all registered adapters.
func (r *Registry) GetAll() []adapter.Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]adapter.Adapter, 0, len(r.adapters))
	for _, adp := range r.adapters {
		result = append(result, adp)
	}
	return result
}

// ListTools returns all registered tool names.
func (r *Registry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		tools = append(tools, name)
	}
	return tools
}
