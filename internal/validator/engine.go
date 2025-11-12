package validator

import (
	"context"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Engine represents a validation engine that can check code against rules
type Engine interface {
	// Name returns the engine name (e.g., "eslint", "llm-validator", "checkstyle")
	Name() string

	// CanHandle checks if this engine can handle the given rule
	CanHandle(rule schema.PolicyRule) bool

	// Returns violations found
	Execute(ctx context.Context, files []string, rules []schema.PolicyRule) ([]Violation, error)
}

// EngineRegistry manages available validation engines
type EngineRegistry struct {
	engines map[string]Engine
}

// NewEngineRegistry creates a new engine registry
func NewEngineRegistry() *EngineRegistry {
	return &EngineRegistry{
		engines: make(map[string]Engine),
	}
}

// Register registers a validation engine
func (r *EngineRegistry) Register(engine Engine) {
	r.engines[engine.Name()] = engine
}

// Get retrieves an engine by name
func (r *EngineRegistry) Get(name string) (Engine, bool) {
	engine, ok := r.engines[name]
	return engine, ok
}

// GetEngineForRule finds the appropriate engine for a rule
func (r *EngineRegistry) GetEngineForRule(rule schema.PolicyRule) Engine {
	// Check if rule specifies an engine
	if engineName, ok := rule.Check["engine"].(string); ok {
		if engine, exists := r.engines[engineName]; exists {
			if engine.CanHandle(rule) {
				return engine
			}
		}
	}

	// Fallback: find any engine that can handle this rule
	for _, engine := range r.engines {
		if engine.CanHandle(rule) {
			return engine
		}
	}

	return nil
}

// ListEngines returns all registered engines
func (r *EngineRegistry) ListEngines() []string {
	names := make([]string, 0, len(r.engines))
	for name := range r.engines {
		names = append(names, name)
	}
	return names
}