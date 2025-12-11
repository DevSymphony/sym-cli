package main

import (
	// Import linters for registration side-effects.
	// Each linter's register.go file contains an init() function
	// that registers the linter with the global registry.
	_ "github.com/DevSymphony/sym-cli/internal/linter/checkstyle"
	_ "github.com/DevSymphony/sym-cli/internal/linter/eslint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/golangcilint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pmd"
	_ "github.com/DevSymphony/sym-cli/internal/linter/prettier"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pylint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/tsc"

	// Import LLM providers for registration side-effects.
	_ "github.com/DevSymphony/sym-cli/internal/llm/claudecode"
	_ "github.com/DevSymphony/sym-cli/internal/llm/geminicli"
	_ "github.com/DevSymphony/sym-cli/internal/llm/openaiapi"
)
