package git

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// GetRepoInfo returns the owner and repo name from the current git repository
func GetRepoInfo() (owner, repo string, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("not a git repository or no remote origin configured")
	}

	url := strings.TrimSpace(string(output))

	// Parse various Git URL formats:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// https://ghes.company.com/owner/repo.git

	// SSH format: git@host:owner/repo.git
	sshRegex := regexp.MustCompile(`git@[^:]+:([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshRegex.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// HTTPS format: https://host/owner/repo.git
	httpsRegex := regexp.MustCompile(`https?://[^/]+/([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpsRegex.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("could not parse repository URL: %s", url)
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetCurrentUser returns the current git user name
// Checks GIT_AUTHOR_NAME environment variable first (for testing),
// then falls back to git config user.name
func GetCurrentUser() (string, error) {
	// Check environment variable first (for testing)
	if user := os.Getenv("GIT_AUTHOR_NAME"); user != "" {
		return user, nil
	}

	// Fallback to git config
	cmd := exec.Command("git", "config", "--get", "user.name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user.name: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
