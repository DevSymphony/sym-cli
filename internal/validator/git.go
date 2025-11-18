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

// GetGitChanges returns all uncommitted changes in the current git repository
// This includes: staged changes, unstaged changes, and untracked files
func GetGitChanges() ([]GitChange, error) {
	changes := make([]GitChange, 0)
	seenFiles := make(map[string]bool) // Track files we've already processed

	// 1. Get staged changes (index vs HEAD)
	stagedCmd := exec.Command("git", "diff", "--cached", "--name-status", "HEAD")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		// If there's no HEAD (initial commit), try without HEAD
		stagedCmd = exec.Command("git", "diff", "--cached", "--name-status")
		stagedOutput, err = stagedCmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("failed to get staged changes: %w (stderr: %s)", err, string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("failed to get staged changes: %w", err)
		}
	}

	if strings.TrimSpace(string(stagedOutput)) != "" {
		lines := strings.Split(strings.TrimSpace(string(stagedOutput)), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			status := parts[0]
			filePath := parts[1]
			seenFiles[filePath] = true

			// Get diff for this file (staged)
			diffCmd := exec.Command("git", "diff", "--cached", "HEAD", "--", filePath)
			diffOutput, err := diffCmd.Output()
			if err != nil {
				// Try without HEAD for initial commit
				diffCmd = exec.Command("git", "diff", "--cached", "--", filePath)
				diffOutput, _ = diffCmd.Output()
			}

			changes = append(changes, GitChange{
				FilePath: filePath,
				Status:   status,
				Diff:     string(diffOutput),
			})
		}
	}

	// 2. Get unstaged changes (working directory vs index)
	unstagedCmd := exec.Command("git", "diff", "--name-status")
	unstagedOutput, err := unstagedCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to get unstaged changes: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to get unstaged changes: %w", err)
	}

	if strings.TrimSpace(string(unstagedOutput)) != "" {
		lines := strings.Split(strings.TrimSpace(string(unstagedOutput)), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			status := parts[0]
			filePath := parts[1]

			// Skip if we already processed this file from staged changes
			if seenFiles[filePath] {
				continue
			}
			seenFiles[filePath] = true

			// Get diff for this file (unstaged)
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
	}

	// 3. Get untracked files
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untrackedOutput, err := untrackedCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to get untracked files: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	if strings.TrimSpace(string(untrackedOutput)) != "" {
		lines := strings.Split(strings.TrimSpace(string(untrackedOutput)), "\n")
		for _, filePath := range lines {
			filePath = strings.TrimSpace(filePath)
			if filePath == "" || seenFiles[filePath] {
				continue
			}
			seenFiles[filePath] = true

			// For untracked files, read the entire file content as "added"
			diffCmd := exec.Command("git", "diff", "--no-index", "/dev/null", filePath)
			diffOutput, err := diffCmd.CombinedOutput()
			// For untracked files, git diff --no-index returns exit code 1, which is expected
			// We still get the output in diffOutput regardless of error
			_ = err // Ignore error since exit code 1 is expected for diffs

			changes = append(changes, GitChange{
				FilePath: filePath,
				Status:   "A", // Treat untracked files as Added
				Diff:     string(diffOutput),
			})
		}
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
