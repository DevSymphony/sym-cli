package adapter

import (
	"os"
	"path/filepath"
)

// DefaultToolsDir returns the standard tools directory (~/.sym/tools).
// Used by all adapters for consistent tool installation location.
func DefaultToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sym", "tools")
}
