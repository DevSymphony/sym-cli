// Package llm provides a unified interface for LLM providers.
package llm

import (
	"fmt"
	"sort"
	"strings"
)

// rawProviderFactory creates a RawProvider instance.
type rawProviderFactory func(cfg Config) (RawProvider, error)

var providers = make(map[string]rawProviderFactory)
var providerMeta = make(map[string]ProviderInfo)

// RegisterProvider registers a provider factory.
// Called by provider packages in their init() functions.
func RegisterProvider(name string, factory rawProviderFactory, info ProviderInfo) {
	providers[name] = factory
	providerMeta[name] = info
}

// New creates a new LLM provider based on the configuration.
// Returns an error if the provider is not available (CLI not installed, API key missing, etc.)
// The returned Provider automatically handles response parsing.
func New(cfg Config) (Provider, error) {
	factory, ok := providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %s)", cfg.Provider, availableProviders())
	}
	rawProvider, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	return wrapWithParser(rawProvider), nil
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

// GetProviderOptions returns a list of display names for all registered providers.
// Results are sorted alphabetically. If includeSkip is true, "Skip" is appended.
func GetProviderOptions(includeSkip bool) []string {
	result := make([]string, 0, len(providerMeta)+1)
	for _, info := range providerMeta {
		result = append(result, info.DisplayName)
	}
	sort.Strings(result)
	if includeSkip {
		result = append(result, "Skip")
	}
	return result
}

// GetProviderByDisplayName returns provider info by display name.
func GetProviderByDisplayName(displayName string) *ProviderInfo {
	for _, info := range providerMeta {
		if info.DisplayName == displayName {
			infoCopy := info
			return &infoCopy
		}
	}
	return nil
}

// GetModelOptions returns model display options for a provider.
// Format: "DisplayName - Description (recommended)" for recommended models.
func GetModelOptions(providerName string) []string {
	info := GetProviderInfo(providerName)
	if info == nil || len(info.Models) == 0 {
		return nil
	}

	result := make([]string, 0, len(info.Models))
	for _, model := range info.Models {
		option := model.DisplayName
		if model.Description != "" {
			option += " - " + model.Description
		}
		if model.Recommended {
			option += " (recommended)"
		}
		result = append(result, option)
	}
	return result
}

// GetModelIDFromOption extracts the model ID from a display option.
func GetModelIDFromOption(providerName, option string) string {
	info := GetProviderInfo(providerName)
	if info == nil {
		return ""
	}

	for _, model := range info.Models {
		displayOption := model.DisplayName
		if model.Description != "" {
			displayOption += " - " + model.Description
		}
		if model.Recommended {
			displayOption += " (recommended)"
		}
		if displayOption == option {
			return model.ID
		}
	}
	return ""
}

// GetDefaultModelOption returns the recommended model display option for a provider.
func GetDefaultModelOption(providerName string) string {
	info := GetProviderInfo(providerName)
	if info == nil {
		return ""
	}

	for _, model := range info.Models {
		if model.Recommended {
			option := model.DisplayName
			if model.Description != "" {
				option += " - " + model.Description
			}
			option += " (recommended)"
			return option
		}
	}

	// Fall back to first model if no recommended
	if len(info.Models) > 0 {
		model := info.Models[0]
		option := model.DisplayName
		if model.Description != "" {
			option += " - " + model.Description
		}
		return option
	}
	return ""
}

// RequiresAPIKey returns true if the provider requires an API key.
func RequiresAPIKey(providerName string) bool {
	info := GetProviderInfo(providerName)
	if info == nil {
		return false
	}
	return info.APIKey.Required
}

// ValidateAPIKey validates an API key for a provider.
// Returns nil if valid, error with message if invalid.
func ValidateAPIKey(providerName, apiKey string) error {
	info := GetProviderInfo(providerName)
	if info == nil {
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	if !info.APIKey.Required {
		return nil // No validation needed
	}

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if info.APIKey.Prefix != "" && !strings.HasPrefix(apiKey, info.APIKey.Prefix) {
		return fmt.Errorf("API key should start with '%s'", info.APIKey.Prefix)
	}

	return nil
}

// GetAPIKeyEnvVar returns the environment variable name for the provider's API key.
func GetAPIKeyEnvVar(providerName string) string {
	info := GetProviderInfo(providerName)
	if info == nil {
		return ""
	}
	return info.APIKey.EnvVarName
}
