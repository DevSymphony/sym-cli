package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// getTestdataDir returns the path to the testdata directory
func getTestdataDir(t *testing.T) string {
	t.Helper()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Go up two levels from tests/integration to project root
	projectRoot := filepath.Join(cwd, "../..")

	return projectRoot
}

// getToolsDir returns the path to tools directory for test
func getToolsDir(t *testing.T) string {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	return filepath.Join(home, ".symphony", "tools")
}
