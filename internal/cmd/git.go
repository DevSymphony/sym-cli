package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// findGitRoot finds the git repository root by looking for .git directory
func findGitRoot() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		// Check if we've reached the root
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in a git repository")
}

// getSymDir returns the .sym directory path in the git root
func getSymDir() (string, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return "", err
	}

	symDir := filepath.Join(gitRoot, ".sym")
	return symDir, nil
}
