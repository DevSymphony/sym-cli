package bootstrap

import (
	// Import LLM providers for registration side-effects.
	// Each provider's register.go file contains an init() function
	// that registers the provider with the global registry.
	_ "github.com/DevSymphony/sym-cli/internal/llm/claudecode"
	_ "github.com/DevSymphony/sym-cli/internal/llm/geminicli"
	_ "github.com/DevSymphony/sym-cli/internal/llm/openai"
)

// This file imports LLM provider packages for their init() side-effects.
// The bootstrap package is imported from main.go to ensure all providers are registered.
