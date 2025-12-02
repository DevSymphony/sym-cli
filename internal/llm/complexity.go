package llm

import "github.com/DevSymphony/sym-cli/internal/llm/engine"

// Complexity re-exports engine.Complexity for backward compatibility.
type Complexity = engine.Complexity

const (
	// ComplexityMinimal is for trivial lookups.
	ComplexityMinimal Complexity = engine.ComplexityMinimal
	// ComplexityLow is for simple transformations.
	ComplexityLow Complexity = engine.ComplexityLow
	// ComplexityMedium is for moderate reasoning.
	ComplexityMedium Complexity = engine.ComplexityMedium
	// ComplexityHigh is for complex reasoning.
	ComplexityHigh Complexity = engine.ComplexityHigh
)
