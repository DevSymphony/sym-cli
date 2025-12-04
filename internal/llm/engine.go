package llm

import "context"

// Complexity represents task complexity hint (engine-agnostic).
// This allows callers to express intent without coupling to specific engine features.
type Complexity int

const (
	// ComplexityMinimal is for trivial lookups or boilerplate prompts.
	ComplexityMinimal Complexity = iota
	// ComplexityLow is for simple transformations, parsing, basic formatting.
	ComplexityLow
	// ComplexityMedium is for analysis, routing decisions, moderate reasoning.
	ComplexityMedium
	// ComplexityHigh is for complex reasoning, code generation, deep analysis.
	ComplexityHigh
)

// String returns human-readable complexity name.
func (c Complexity) String() string {
	switch c {
	case ComplexityMinimal:
		return "minimal"
	case ComplexityLow:
		return "low"
	case ComplexityMedium:
		return "medium"
	case ComplexityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// Request represents an engine-agnostic LLM request.
// All engines receive this unified request format and interpret it according to their capabilities.
type Request struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
	Complexity   Complexity
}

// CombinedPrompt returns system and user prompts combined.
func (r *Request) CombinedPrompt() string {
	if r.SystemPrompt == "" {
		return r.UserPrompt
	}
	return r.SystemPrompt + "\n\n" + r.UserPrompt
}

// LLMEngine is the interface for LLM execution engines.
type LLMEngine interface {
	// Execute sends request and returns response text.
	Execute(ctx context.Context, req *Request) (string, error)

	// Name returns engine identifier.
	Name() string

	// IsAvailable checks if this engine can currently be used.
	IsAvailable() bool

	// Capabilities returns what features this engine supports.
	Capabilities() Capabilities
}

// Capabilities describes what features an engine supports.
// This enables graceful degradation when features aren't available.
type Capabilities struct {
	// SupportsTemperature indicates if temperature parameter is respected.
	SupportsTemperature bool

	// SupportsMaxTokens indicates if max_tokens parameter is respected.
	SupportsMaxTokens bool

	// SupportsComplexity indicates if complexity hint affects model selection.
	SupportsComplexity bool

	// SupportsStreaming indicates if streaming responses are supported.
	SupportsStreaming bool

	// MaxContextLength is the maximum input context length (0 = unknown).
	MaxContextLength int

	// Models lists available models for this engine.
	Models []string
}

// Mode represents the preferred engine selection mode.
type Mode string

const (
	// ModeAuto automatically selects the best available engine.
	ModeAuto Mode = "auto"
	// ModeCLI forces CLI engine.
	ModeCLI Mode = "cli"
	// ModeAPI forces API engine.
	ModeAPI Mode = "api"
)

// IsValid checks if the engine mode is valid.
func (m Mode) IsValid() bool {
	switch m {
	case ModeAuto, ModeCLI, ModeAPI:
		return true
	default:
		return false
	}
}
