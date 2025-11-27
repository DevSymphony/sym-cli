package integration

// Import adapters to trigger init() registration
import (
	_ "github.com/DevSymphony/sym-cli/internal/adapter/checkstyle"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/pmd"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/prettier"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/pylint"
	_ "github.com/DevSymphony/sym-cli/internal/adapter/tsc"
)
