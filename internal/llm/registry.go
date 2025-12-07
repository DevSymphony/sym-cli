// Package llm provides a unified interface for LLM providers.
package llm

import (
	"fmt"
	"strings"
)

// providerFactory creates a provider instance.
type providerFactory func(cfg Config) (Provider, error)

var providers = make(map[string]providerFactory)
var providerMeta = make(map[string]ProviderInfo)

// RegisterProvider registers a provider factory.
// Called by provider packages in their init() functions.
func RegisterProvider(name string, factory providerFactory, info ProviderInfo) {
	providers[name] = factory
	providerMeta[name] = info
}

// New creates a new LLM provider based on the configuration.
// Returns an error if the provider is not available (CLI not installed, API key missing, etc.)
func New(cfg Config) (Provider, error) {
	factory, ok := providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %s)", cfg.Provider, availableProviders())
	}
	return factory(cfg)
}

// GetProviderInfo returns metadata for a provider.
func GetProviderInfo(name string) *ProviderInfo {
	info, ok := providerMeta[name]
	if !ok {
		return nil
	}
	return &info
}

// ListProviders returns info for all registered providers.
func ListProviders() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(providerMeta))
	for _, info := range providerMeta {
		result = append(result, info)
	}
	return result
}

func availableProviders() string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
