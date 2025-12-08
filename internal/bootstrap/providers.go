package bootstrap

import (
	// Import LLM providers for registration side-effects.
	_ "github.com/DevSymphony/sym-cli/internal/llm/claudecode"
	_ "github.com/DevSymphony/sym-cli/internal/llm/geminicli"
	_ "github.com/DevSymphony/sym-cli/internal/llm/openaiapi"
)
