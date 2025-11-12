package linters

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	globalRegistry = &Registry{
		converters: make(map[string]LinterConverter),
	}
)

// Registry manages available linter converters
type Registry struct {
	mu         sync.RWMutex
	converters map[string]LinterConverter
}

// Register registers a linter converter
func Register(converter LinterConverter) {
	globalRegistry.Register(converter)
}

// Get retrieves a linter converter by name
func Get(name string) (LinterConverter, error) {
	return globalRegistry.Get(name)
}

// List returns all registered linter names
func List() []string {
	return globalRegistry.List()
}

// GetByLanguage returns converters that support a specific language
func GetByLanguage(language string) []LinterConverter {
	return globalRegistry.GetByLanguage(language)
}

// Register registers a linter converter
func (r *Registry) Register(converter LinterConverter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(converter.Name())
	r.converters[name] = converter
}

// Get retrieves a linter converter by name
func (r *Registry) Get(name string) (LinterConverter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name = strings.ToLower(name)
	converter, ok := r.converters[name]
	if !ok {
		return nil, fmt.Errorf("linter converter not found: %s", name)
	}

	return converter, nil
}

// List returns all registered linter names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.converters))
	for name := range r.converters {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetByLanguage returns converters that support a specific language
func (r *Registry) GetByLanguage(language string) []LinterConverter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	language = strings.ToLower(language)
	converters := make([]LinterConverter, 0)

	for _, converter := range r.converters {
		for _, lang := range converter.SupportedLanguages() {
			if strings.ToLower(lang) == language {
				converters = append(converters, converter)
				break
			}
		}
	}

	return converters
}

// GetAll returns all registered converters (package-level function)
func GetAll() []LinterConverter {
	return globalRegistry.GetAll()
}

// GetAll returns all registered converters
func (r *Registry) GetAll() []LinterConverter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	converters := make([]LinterConverter, 0, len(r.converters))
	for _, converter := range r.converters {
		converters = append(converters, converter)
	}

	return converters
}
