package validator

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitChange represents a file change in git
type GitChange struct {
	FilePath string
	Status   string // A(dded), M(odified), D(eleted)
	Diff     string
}

// GetGitChanges returns all changed files in the current git repository (unstaged changes)
func GetGitChanges() ([]GitChange, error) {
	// Get list of unstaged changed files (working directory vs index)
	cmd := exec.Command("git", "diff", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		// Try to get more detailed error information
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to get git changes: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to get git changes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return []GitChange{}, nil
	}

	changes := make([]GitChange, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		// Get diff for this file (working directory vs index)
		diffCmd := exec.Command("git", "diff", "--", filePath)
		diffOutput, err := diffCmd.Output()
		if err != nil {
			continue
		}

		changes = append(changes, GitChange{
			FilePath: filePath,
			Status:   status,
			Diff:     string(diffOutput),
		})
	}

	return changes, nil
}

// GetStagedChanges returns staged changes
func GetStagedChanges() ([]GitChange, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged changes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return []GitChange{}, nil
	}

	changes := make([]GitChange, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		diffCmd := exec.Command("git", "diff", "--cached", "--", filePath)
		diffOutput, err := diffCmd.Output()
		if err != nil {
			continue
		}

		changes = append(changes, GitChange{
			FilePath: filePath,
			Status:   status,
			Diff:     string(diffOutput),
		})
	}

	return changes, nil
}

// ExtractAddedLines extracts only added lines from a diff
func ExtractAddedLines(diff string) []string {
	lines := strings.Split(diff, "\n")
	added := make([]string, 0)

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added = append(added, strings.TrimPrefix(line, "+"))
		}
	}

	return added
}
