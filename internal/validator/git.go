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

// GetGitChanges returns all changed files in the current git repository
func GetGitChanges() ([]GitChange, error) {
	// Get list of changed files
	cmd := exec.Command("git", "diff", "--name-status", "HEAD")
	output, err := cmd.Output()
	if err != nil {
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

		// Get diff for this file
		diffCmd := exec.Command("git", "diff", "HEAD", "--", filePath)
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
