package checkstyle

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// CheckstyleOutput represents the JSON output from Checkstyle.
type CheckstyleOutput struct {
	Files []CheckstyleFile `json:"files"`
}

// CheckstyleFile represents a file with violations in Checkstyle output.
type CheckstyleFile struct {
	Name   string              `json:"name"`
	Errors []CheckstyleError   `json:"errors"`
}

// CheckstyleError represents a single violation in Checkstyle output.
type CheckstyleError struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// parseOutput converts Checkstyle JSON output to violations.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	if output == nil {
		return nil, fmt.Errorf("output is nil")
	}

	// If no output and exit code 0, no violations
	if output.Stdout == "" && output.ExitCode == 0 {
		return []adapter.Violation{}, nil
	}

	// Parse JSON output
	var result CheckstyleOutput
	if err := json.Unmarshal([]byte(output.Stdout), &result); err != nil {
		// If JSON parsing fails, try to extract errors from stderr
		if output.Stderr != "" {
			return nil, fmt.Errorf("checkstyle failed: %s", output.Stderr)
		}
		return nil, fmt.Errorf("failed to parse checkstyle output: %w", err)
	}

	// Convert to violations
	violations := make([]adapter.Violation, 0)

	for _, file := range result.Files {
		for _, err := range file.Errors {
			violations = append(violations, adapter.Violation{
				File:     file.Name,
				Line:     err.Line,
				Column:   err.Column,
				Message:  err.Message,
				Severity: mapSeverity(err.Severity),
				RuleID:   extractRuleID(err.Source),
			})
		}
	}

	return violations, nil
}

// mapSeverity maps Checkstyle severity to standard severity.
func mapSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "error"
	case "warning", "warn":
		return "warning"
	case "info":
		return "info"
	default:
		return "warning"
	}
}

// extractRuleID extracts the rule ID from Checkstyle source string.
// Example: "com.puppycrawl.tools.checkstyle.checks.naming.TypeNameCheck" -> "TypeName"
func extractRuleID(source string) string {
	if source == "" {
		return "unknown"
	}

	// Split by dots and get the last part
	parts := strings.Split(source, ".")
	if len(parts) == 0 {
		return source
	}

	lastPart := parts[len(parts)-1]

	// Remove "Check" suffix if present
	lastPart = strings.TrimSuffix(lastPart, "Check")

	return lastPart
}
