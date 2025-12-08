package bootstrap

import (
	// Import linters for registration side-effects.
	// Each linter's register.go file contains an init() function
	// that registers the linter with the global registry.
	_ "github.com/DevSymphony/sym-cli/internal/linter/checkstyle"
	_ "github.com/DevSymphony/sym-cli/internal/linter/eslint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pmd"
	_ "github.com/DevSymphony/sym-cli/internal/linter/prettier"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pylint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/tsc"
)

// This package only imports linter packages for their init() side-effects.
// Import this package from main.go to ensure all linters are registered.
