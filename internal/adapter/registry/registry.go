package registry

import (
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Registry manages adapter instances and provides capability-based lookup.
type Registry struct {
	mu sync.RWMutex

	// adapters stores all registered adapters.
	adapters []adapter.Adapter

	// languageCache maps language to adapters for faster lookup.
	// Key: language (e.g., "javascript"), Value: adapters that support it.
	languageCache map[string][]adapter.Adapter
}

// NewRegistry creates a new empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters:      make([]adapter.Adapter, 0),
		languageCache: make(map[string][]adapter.Adapter),
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

	r.adapters = append(r.adapters, adp)

	// Update language cache
	caps := adp.GetCapabilities()
	for _, lang := range caps.SupportedLanguages {
		r.languageCache[lang] = append(r.languageCache[lang], adp)
	}

	return nil
}

// GetAdapter finds an adapter that supports the given language and category.
// Returns the first matching adapter, or ErrAdapterNotFound if none match.
func (r *Registry) GetAdapter(language, category string) (adapter.Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First, filter by language using cache
	candidates, ok := r.languageCache[language]
	if !ok || len(candidates) == 0 {
		return nil, &ErrLanguageNotSupported{Language: language}
	}

	// Then, filter by category
	for _, adp := range candidates {
		caps := adp.GetCapabilities()
		if contains(caps.SupportedCategories, category) {
			return adp, nil
		}
	}

	return nil, &ErrAdapterNotFound{Language: language, Category: category}
}

// GetAll returns all registered adapters.
func (r *Registry) GetAll() []adapter.Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]adapter.Adapter, len(r.adapters))
	copy(result, r.adapters)
	return result
}

// GetSupportedLanguages returns all languages supported for the given category.
// If category is empty, returns all supported languages across all categories.
func (r *Registry) GetSupportedLanguages(category string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	languageSet := make(map[string]bool)

	for _, adp := range r.adapters {
		caps := adp.GetCapabilities()

		// If category is specified, filter by it
		if category != "" && !contains(caps.SupportedCategories, category) {
			continue
		}

		// Add all supported languages
		for _, lang := range caps.SupportedLanguages {
			languageSet[lang] = true
		}
	}

	// Convert set to slice
	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}

	return languages
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
