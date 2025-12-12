package golangcilint

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// golangciOutput represents the JSON output from golangci-lint.
type golangciOutput struct {
	Issues []golangciIssue `json:"Issues"`
	Report golangciReport  `json:"Report"`
}

// golangciIssue represents a single issue in golangci-lint output.
type golangciIssue struct {
	FromLinter  string        `json:"FromLinter"`
	Text        string        `json:"Text"`
	Severity    string        `json:"Severity"`
	SourceLines []string      `json:"SourceLines"`
	Pos         golangciPos   `json:"Pos"`
	Replacement *golangciRepl `json:"Replacement,omitempty"`
}

// golangciPos represents the position of an issue.
type golangciPos struct {
	Filename string `json:"Filename"`
	Offset   int    `json:"Offset"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

// golangciRepl represents a suggested replacement (not used, but included for completeness).
type golangciRepl struct {
	Lines []string `json:"Lines"`
}

// golangciReport represents metadata about the run.
type golangciReport struct {
	Linters []golangciLinterInfo `json:"Linters"`
}

// golangciLinterInfo represents information about a linter.
type golangciLinterInfo struct {
	Name             string `json:"Name"`
	Enabled          bool   `json:"Enabled"`
	EnabledByDefault bool   `json:"EnabledByDefault"`
}

// parseOutput converts golangci-lint JSON output to violations.
func parseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	if output == nil {
		return nil, fmt.Errorf("output is nil")
	}

	// If no output and exit code 0, no violations
	if output.Stdout == "" && output.ExitCode == 0 {
		return []linter.Violation{}, nil
	}

	// If empty stdout but exit code 1, it might be an error in stderr
	if output.Stdout == "" && output.Stderr != "" {
		return nil, fmt.Errorf("golangci-lint error: %s", output.Stderr)
	}

	// Parse JSON output
	var result golangciOutput
	if err := json.Unmarshal([]byte(output.Stdout), &result); err != nil {
		// Provide context about parse error
		return nil, fmt.Errorf("failed to parse golangci-lint output: %w\nOutput: %.200s", err, output.Stdout)
	}

	// Convert issues to violations
	violations := make([]linter.Violation, 0, len(result.Issues))

	for _, issue := range result.Issues {
		violations = append(violations, linter.Violation{
			File:     issue.Pos.Filename,
			Line:     issue.Pos.Line,
			Column:   issue.Pos.Column,
			Message:  issue.Text,
			Severity: mapSeverity(issue.Severity),
			RuleID:   issue.FromLinter,
		})
	}

	return violations, nil
}

// mapSeverity maps golangci-lint severity to standard severity.
func mapSeverity(s string) string {
	switch strings.ToLower(s) {
	case "error":
		return "error"
	case "warning":
		return "warning"
	case "info":
		return "info"
	default:
		// Default to info for unknown severities
		return "info"
	}
}
