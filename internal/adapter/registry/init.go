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
func DefaultRegistry() *Registry {
	reg := NewRegistry()

	// Determine tools directory
	toolsDir := getToolsDir()
	workDir := getWorkDir()

	// Register JavaScript/TypeScript adapters
	_ = reg.Register(eslint.NewAdapter(toolsDir, workDir))
	_ = reg.Register(prettier.NewAdapter(toolsDir, workDir))
	_ = reg.Register(tsc.NewAdapter(toolsDir, workDir))

	// Register Java adapters
	_ = reg.Register(checkstyle.NewAdapter(toolsDir, workDir))
	_ = reg.Register(pmd.NewAdapter(toolsDir, workDir))

	return reg
}

// getToolsDir returns the default tools directory.
func getToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sym", "tools")
}

// getWorkDir returns the current working directory.
func getWorkDir() string {
	wd, _ := os.Getwd()
	return wd
}
