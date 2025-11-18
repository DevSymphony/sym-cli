package checkstyle

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// CheckstyleOutput represents the XML output from Checkstyle.
type CheckstyleOutput struct {
	XMLName xml.Name         `xml:"checkstyle"`
	Files   []CheckstyleFile `xml:"file"`
}

// CheckstyleFile represents a file with violations in Checkstyle output.
type CheckstyleFile struct {
	Name   string            `xml:"name,attr"`
	Errors []CheckstyleError `xml:"error"`
}

// CheckstyleError represents a single violation in Checkstyle output.
type CheckstyleError struct {
	Line     int    `xml:"line,attr"`
	Column   int    `xml:"column,attr"`
	Severity string `xml:"severity,attr"`
	Message  string `xml:"message,attr"`
	Source   string `xml:"source,attr"`
}

// parseOutput converts Checkstyle XML output to violations.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	if output == nil {
		return nil, fmt.Errorf("output is nil")
	}

	// If no output and exit code 0, no violations
	if output.Stdout == "" && output.ExitCode == 0 {
		return []adapter.Violation{}, nil
	}

	// If no output but non-zero exit code, something went wrong
	if output.Stdout == "" {
		if output.Stderr != "" {
			return nil, fmt.Errorf("checkstyle failed (exit code %d): %s", output.ExitCode, output.Stderr)
		}
		return nil, fmt.Errorf("checkstyle failed with exit code %d but produced no output", output.ExitCode)
	}

	// Parse XML output
	var result CheckstyleOutput
	if err := xml.Unmarshal([]byte(output.Stdout), &result); err != nil {
		// If XML parsing fails, try to extract errors from stderr
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
