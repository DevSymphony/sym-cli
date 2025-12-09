package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentUser returns the current git user name
func GetCurrentUser() (string, error) {
	cmd := exec.Command("git", "config", "--get", "user.name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user.name: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
