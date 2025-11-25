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
// Returns errNilAdapter if the adapter is nil.
func (r *Registry) Register(adp adapter.Adapter) error {
	if adp == nil {
		return errNilAdapter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := adp.Name()
	r.adapters[name] = adp

	return nil
}

// GetAdapter finds an adapter by tool name (e.g., "eslint", "prettier", "tsc").
// Returns the adapter, or errAdapterNotFound if not registered.
func (r *Registry) GetAdapter(toolName string) (adapter.Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adp, ok := r.adapters[toolName]
	if !ok {
		return nil, &errAdapterNotFound{ToolName: toolName}
	}

	return adp, nil
}
