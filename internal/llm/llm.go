// Package llm provides a unified interface for LLM providers.
package llm

import "context"

// Provider is the interface for LLM providers.
type Provider interface {
	// Execute sends a prompt and returns the parsed response.
	Execute(ctx context.Context, prompt string, format ResponseFormat) (string, error)
	// Name returns the provider name.
	Name() string
	// Close releases any resources held by the provider.
	Close() error
}

// RawProvider is the interface for provider implementations.
// Provider implementations should implement this interface.
// The registry will automatically wrap RawProvider with parsing logic.
type RawProvider interface {
	// ExecuteRaw sends a prompt and returns the raw (unparsed) response.
	ExecuteRaw(ctx context.Context, prompt string, format ResponseFormat) (string, error)
	// Name returns the provider name.
	Name() string
	// Close releases any resources held by the provider.
	Close() error
}

// ResponseFormat specifies the expected response format.
type ResponseFormat string

const (
	Text ResponseFormat = "text"
	JSON ResponseFormat = "json"
	XML  ResponseFormat = "xml"
)

// String returns the string representation of the format.
func (f ResponseFormat) String() string {
	return string(f)
}

// Config holds LLM provider configuration.
type Config struct {
	Provider string // "claudecode", "geminicli", "openaiapi"
	Model    string // Model name (optional, uses provider default)
	APIKey   string // API key for OpenAI
	Verbose  bool   // Enable verbose logging
}

// ProviderInfo contains provider metadata.
type ProviderInfo struct {
	Name         string
	DisplayName  string
	DefaultModel string
	Available    bool
	Path         string // CLI path or empty for API providers
}
