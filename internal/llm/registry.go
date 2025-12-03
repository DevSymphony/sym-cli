package llm

import (
	"sort"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
)

// EngineFactory creates an LLMEngine instance from configuration.
type EngineFactory func(cfg *EngineConfig) (engine.LLMEngine, error)

// EngineConfig holds common configuration for all LLM engines.
type EngineConfig struct {
	APIKey     string
	Model      string
	LargeModel string
	CLIPath    string
	Verbose    bool
	MCPSession interface{}
}

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
// Should be called from init() in each provider package.
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

// GetRegisteredNames returns all registered provider names.
func GetRegisteredNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
