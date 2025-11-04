package policy

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PolicyCommit represents a single commit in policy history
type PolicyCommit struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Date      time.Time `json:"date"`
	Message   string    `json:"message"`
	FilesChanged int    `json:"filesChanged"`
}

// GetPolicyHistory returns the git commit history for the policy file
func GetPolicyHistory(customPath string, limit int) ([]PolicyCommit, error) {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	// Git log command with custom format
	// Format: hash|author|email|timestamp|subject
	cmd := exec.Command("git", "log",
		fmt.Sprintf("--max-count=%d", limit),
		"--pretty=format:%H|%an|%ae|%at|%s",
		"--", policyPath)

	output, err := cmd.Output()
	if err != nil {
		// If file has no history yet, return empty list
		return []PolicyCommit{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]PolicyCommit, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}

		var timestamp int64
		if _, err := fmt.Sscanf(parts[3], "%d", &timestamp); err != nil {
			// Skip malformed timestamp
			continue
		}

		commit := PolicyCommit{
			Hash:    parts[0],
			Author:  parts[1],
			Email:   parts[2],
			Date:    time.Unix(timestamp, 0),
			Message: parts[4],
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// GetPolicyDiff returns the diff for a specific commit
func GetPolicyDiff(commitHash string, customPath string) (string, error) {
	policyPath, err := GetPolicyPath(customPath)
	if err != nil {
		return "", err
	}

	// Get diff for specific commit
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitHash, policyPath))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetLatestCommit returns the most recent commit for the policy file
func GetLatestCommit(customPath string) (*PolicyCommit, error) {
	commits, err := GetPolicyHistory(customPath, 1)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found for policy file")
	}

	return &commits[0], nil
}
