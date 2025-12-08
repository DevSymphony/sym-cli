package linter

import (
	"fmt"
	"log"
	"sync"
)

// ===== Errors =====

// errLinterNotFound is returned when no linter is found for the given tool name.
type errLinterNotFound struct {
	ToolName string
}

func (e *errLinterNotFound) Error() string {
	return fmt.Sprintf("linter not found: %s", e.ToolName)
}

// errNilLinter is returned when trying to register a nil linter.
var errNilLinter = fmt.Errorf("cannot register nil linter")

// ===== Registry =====

// ToolRegistration contains all metadata for a linter tool.
type ToolRegistration struct {
	Linter     Linter    // Linter instance
	Converter  Converter // Converter instance (optional)
	ConfigFile string    // Config filename (e.g., ".eslintrc.json")
}

// Registry manages linter registrations.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*ToolRegistration
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// Global returns the singleton registry instance.
func Global() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			tools: make(map[string]*ToolRegistration),
		}
	})
	return globalRegistry
}

// RegisterTool registers a tool with linter, converter, and config file.
func (r *Registry) RegisterTool(
	l Linter,
	converter Converter,
	configFile string,
) error {
	if l == nil {
		return errNilLinter
	}

	name := l.Name()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Warn on duplicate registration (init order issues)
	if _, exists := r.tools[name]; exists {
		log.Printf("warning: linter already registered: %s (ignoring duplicate)", name)
		return nil
	}

	r.tools[name] = &ToolRegistration{
		Linter:     l,
		Converter:  converter,
		ConfigFile: configFile,
	}

	return nil
}

// GetLinter finds a linter by tool name (e.g., "eslint", "prettier", "tsc").
func (r *Registry) GetLinter(toolName string) (Linter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.tools[toolName]; ok {
		return reg.Linter, nil
	}

	return nil, &errLinterNotFound{ToolName: toolName}
}

// GetConverter returns Converter by tool name.
func (r *Registry) GetConverter(name string) (Converter, bool) {
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

// BuildLanguageMapping dynamically builds language->tools mapping from linter capabilities.
func (r *Registry) BuildLanguageMapping() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mapping := make(map[string][]string)
	for name, reg := range r.tools {
		caps := reg.Linter.GetCapabilities()
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

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetAllConfigFiles returns all registered config file names.
func (r *Registry) GetAllConfigFiles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	files := make([]string, 0, len(r.tools))
	for _, reg := range r.tools {
		if reg.ConfigFile != "" {
			files = append(files, reg.ConfigFile)
		}
	}
	return files
}

// GetAllConverters returns all registered converters.
func (r *Registry) GetAllConverters() []Converter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	converters := make([]Converter, 0, len(r.tools))
	for _, reg := range r.tools {
		if reg.Converter != nil {
			converters = append(converters, reg.Converter)
		}
	}
	return converters
}
