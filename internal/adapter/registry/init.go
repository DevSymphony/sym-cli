package registry

import (
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter/checkstyle"
	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/adapter/pmd"
	"github.com/DevSymphony/sym-cli/internal/adapter/prettier"
	"github.com/DevSymphony/sym-cli/internal/adapter/tsc"
)

// DefaultRegistry creates and populates a registry with all available adapters.
// Note: Adapters are stateless and use CWD at execution time for working directory.
func DefaultRegistry() *Registry {
	reg := NewRegistry()

	// Determine tools directory
	toolsDir := getToolsDir()

	// Register JavaScript/TypeScript adapters
	_ = reg.Register(eslint.NewAdapter(toolsDir))
	_ = reg.Register(prettier.NewAdapter(toolsDir))
	_ = reg.Register(tsc.NewAdapter(toolsDir))

	// Register Java adapters
	_ = reg.Register(checkstyle.NewAdapter(toolsDir))
	_ = reg.Register(pmd.NewAdapter(toolsDir))

	return reg
}

// getToolsDir returns the default tools directory.
func getToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sym", "tools")
}
