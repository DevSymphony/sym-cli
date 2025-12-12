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

// the execution strategy for LLM-based validation.
type ProviderMode string

const (
	// ModeParallelAPI is for traditional API providers (OpenAI, Gemini API).
	ModeParallelAPI ProviderMode = "parallel_api"

	// ModeAgenticSingle is for agentic CLI tools (Claude Code, Gemini CLI).
	ModeAgenticSingle ProviderMode = "agentic_single"
)

// ProviderProfile contains mode-specific configuration for execution strategy.
type ProviderProfile struct {
	// MaxPromptChars is the maximum prompt length before truncation.
	MaxPromptChars int

	// DefaultTimeoutSec is the default timeout per request in seconds.
	DefaultTimeoutSec int

	// MaxRetries is the maximum retry attempts on transient failures.
	MaxRetries int

	// ResponseFormatHint suggests the expected response structure to the LLM.
	ResponseFormatHint string
}

// String returns the string representation of the format.
func (f ResponseFormat) String() string {
	return string(f)
}

// Config holds LLM provider configuration.
type Config struct {
	Provider string // "claudecode", "geminicli", "openaiapi"
	Model    string // Model name (optional, uses provider default)
	Verbose  bool   // Enable verbose logging
}

// ModelInfo describes a model available for a provider.
type ModelInfo struct {
	ID          string // Internal model identifier (e.g., "sonnet", "gpt-4o-mini")
	DisplayName string // Human-readable name for UI
	Description string // Short description
	Recommended bool   // Default/recommended model flag
}

// APIKeyConfig describes API key requirements for a provider.
type APIKeyConfig struct {
	Required   bool   // Whether this provider requires an API key
	EnvVarName string // Environment variable name (e.g., "OPENAI_API_KEY")
	Prefix     string // Expected prefix for validation (e.g., "sk-")
}

// ProviderInfo contains provider metadata.
type ProviderInfo struct {
	Name         string
	DisplayName  string
	DefaultModel string
	Available    bool
	Path         string          // CLI path or empty for API providers
	Models       []ModelInfo     // Available models for this provider
	APIKey       APIKeyConfig    // API key configuration
	Mode         ProviderMode    // Execution mode (agentic_single or parallel_api)
	Profile      ProviderProfile // Mode-specific execution profile
}
