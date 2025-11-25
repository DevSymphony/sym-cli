package registry

import (
	"log"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// ToolRegistration contains all metadata for a linter tool.
type ToolRegistration struct {
	Adapter    adapter.Adapter          // Adapter instance
	Converter  adapter.LinterConverter  // LinterConverter instance (optional)
	ConfigFile string                   // Config filename (e.g., ".eslintrc.json")
}

// Registry manages tool registrations.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*ToolRegistration

	// Legacy: adapters map for backward compatibility
	adapters map[string]adapter.Adapter
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// Global returns the singleton registry instance.
func Global() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			tools:    make(map[string]*ToolRegistration),
			adapters: make(map[string]adapter.Adapter),
		}
	})
	return globalRegistry
}

// NewRegistry creates a new empty adapter registry.
// Deprecated: Use Global() instead for the singleton pattern.
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]*ToolRegistration),
		adapters: make(map[string]adapter.Adapter),
	}
}

// RegisterTool registers a tool with adapter, converter, and config file.
func (r *Registry) RegisterTool(
	adp adapter.Adapter,
	converter adapter.LinterConverter,
	configFile string,
) error {
	if adp == nil {
		return errNilAdapter
	}

	name := adp.Name()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Warn on duplicate registration (init order issues)
	if _, exists := r.tools[name]; exists {
		log.Printf("warning: adapter already registered: %s (ignoring duplicate)", name)
		return nil
	}

	r.tools[name] = &ToolRegistration{
		Adapter:    adp,
		Converter:  converter,
		ConfigFile: configFile,
	}

	// Also register in legacy adapters map for backward compatibility
	r.adapters[name] = adp

	return nil
}

// Register adds an adapter to the registry.
// Deprecated: Use RegisterTool() for new registrations.
func (r *Registry) Register(adp adapter.Adapter) error {
	if adp == nil {
		return errNilAdapter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := adp.Name()

	// Check if already registered via RegisterTool
	if _, exists := r.tools[name]; exists {
		return nil
	}

	r.adapters[name] = adp

	return nil
}

// GetAdapter finds an adapter by tool name (e.g., "eslint", "prettier", "tsc").
func (r *Registry) GetAdapter(toolName string) (adapter.Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First check tools map
	if reg, ok := r.tools[toolName]; ok {
		return reg.Adapter, nil
	}

	// Fallback to legacy adapters map
	if adp, ok := r.adapters[toolName]; ok {
		return adp, nil
	}

	return nil, &errAdapterNotFound{ToolName: toolName}
}

// GetConverter returns LinterConverter by tool name.
func (r *Registry) GetConverter(name string) (adapter.LinterConverter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.tools[name]; ok && reg.Converter != nil {
		return reg.Converter, true
	}
	return nil, false
}

// GetConfigFile returns config filename by tool name.
func (r *Registry) GetConfigFile(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.tools[name]; ok {
		return reg.ConfigFile
	}
	return ""
}

// BuildLanguageMapping dynamically builds language->tools mapping from adapter capabilities.
func (r *Registry) BuildLanguageMapping() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mapping := make(map[string][]string)
	for name, reg := range r.tools {
		caps := reg.Adapter.GetCapabilities()
		for _, lang := range caps.SupportedLanguages {
			mapping[lang] = append(mapping[lang], name)
		}
	}
	return mapping
}

// GetAllToolNames returns all registered tool names.
func (r *Registry) GetAllToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools)+len(r.adapters))
	seen := make(map[string]bool)

	for name := range r.tools {
		names = append(names, name)
		seen[name] = true
	}

	// Add any legacy adapters not in tools
	for name := range r.adapters {
		if !seen[name] {
			names = append(names, name)
		}
	}

	return names
}
