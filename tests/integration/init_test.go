package integration

// Import adapters to trigger init() registration
import (
	_ "github.com/DevSymphony/sym-cli/internal/linter/checkstyle"
	_ "github.com/DevSymphony/sym-cli/internal/linter/eslint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pmd"
	_ "github.com/DevSymphony/sym-cli/internal/linter/prettier"
	_ "github.com/DevSymphony/sym-cli/internal/linter/pylint"
	_ "github.com/DevSymphony/sym-cli/internal/linter/tsc"
)
