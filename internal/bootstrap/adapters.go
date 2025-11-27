package bootstrap

import (
	// Import adapters for registration side-effects.
	// Each adapter's register.go file contains an init() function
	// that registers the adapter with the global registry.
	_ "github.com/DevSymphony/sym-cli/internal/adapter/checkstyle"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/pmd"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/prettier"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/pylint"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/tsc"
)

// This package only imports adapter packages for their init() side-effects.
// Import this package from main.go to ensure all adapters are registered.
