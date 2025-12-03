package engine

import (
	"sort"
	"sync"
)

// EngineConfig holds common configuration for all LLM engines.
type EngineConfig struct {
	APIKey     string
	Model      string
	LargeModel string
	CLIPath    string
	Verbose    bool
	MCPSession interface{}
}

// EngineFactory creates an LLMEngine instance from configuration.
type EngineFactory func(cfg *EngineConfig) (LLMEngine, error)

// Registration represents a registered LLM provider.
type Registration struct {
	Name     string
	Priority int // Higher = preferred in auto mode
	Factory  EngineFactory
}

var (
	registry   = make(map[string]*Registration)
	registryMu sync.RWMutex
)

// Register adds an LLM provider to the registry.
func Register(r *Registration) {
	if r == nil || r.Factory == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[r.Name] = r
}

// GetRegistration returns a registration by name.
func GetRegistration(name string) (*Registration, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	r, ok := registry[name]
	return r, ok
}

// GetAllRegistrations returns all registrations sorted by priority (descending).
func GetAllRegistrations() []*Registration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]*Registration, 0, len(registry))
	for _, r := range registry {
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority > result[j].Priority
	})

	return result
}
