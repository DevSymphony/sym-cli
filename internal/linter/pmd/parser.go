package pmd

import (
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// PMDOutput represents the JSON output from PMD.
type PMDOutput struct {
	FormatVersion int              `json:"formatVersion"`
	PMDVersion    string           `json:"pmdVersion"`
	Files         []PMDFile        `json:"files"`
	ProcessingErrors []PMDProcessingError `json:"processingErrors"`
}

// PMDFile represents a file with violations in PMD output.
type PMDFile struct {
	Filename   string         `json:"filename"`
	Violations []PMDViolation `json:"violations"`
}

// PMDViolation represents a single violation in PMD output.
type PMDViolation struct {
	BeginLine   int    `json:"beginLine"`
	BeginColumn int    `json:"beginColumn"`
	EndLine     int    `json:"endLine"`
	EndColumn   int    `json:"endColumn"`
	Description string `json:"description"`
	Rule        string `json:"rule"`
	RuleSet     string `json:"ruleSet"`
	Priority    int    `json:"priority"`
	ExternalInfo string `json:"externalInfoUrl"`
}

// PMDProcessingError represents an error during PMD analysis.
type PMDProcessingError struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
}

// parseOutput converts PMD JSON output to violations.
func parseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	if output == nil {
		return nil, fmt.Errorf("output is nil")
	}

	// If no output and exit code 0, no violations
	if output.Stdout == "" && output.ExitCode == 0 {
		return []linter.Violation{}, nil
	}

	// Parse JSON output
	var result PMDOutput
	if err := json.Unmarshal([]byte(output.Stdout), &result); err != nil {
		// If JSON parsing fails, try to extract errors from stderr
		if output.Stderr != "" {
			return nil, fmt.Errorf("PMD failed: %s", output.Stderr)
		}
		return nil, fmt.Errorf("failed to parse PMD output: %w", err)
	}

	// Convert to violations
	violations := make([]linter.Violation, 0)

	for _, file := range result.Files {
		for _, v := range file.Violations {
			violations = append(violations, linter.Violation{
				File:     file.Filename,
				Line:     v.BeginLine,
				Column:   v.BeginColumn,
				Message:  v.Description,
				Severity: mapPriority(v.Priority),
				RuleID:   v.Rule,
			})
		}
	}

	// Add processing errors as violations
	for _, err := range result.ProcessingErrors {
		violations = append(violations, linter.Violation{
			File:     err.Filename,
			Line:     0,
			Column:   0,
			Message:  fmt.Sprintf("Processing error: %s", err.Message),
			Severity: "error",
			RuleID:   "PMDProcessingError",
		})
	}

	return violations, nil
}

// mapPriority maps PMD priority to standard severity.
// PMD priority: 1 (high) to 5 (low)
func mapPriority(priority int) string {
	switch priority {
	case 1, 2:
		return "error"
	case 3:
		return "warning"
	case 4, 5:
		return "info"
	default:
		return "warning"
	}
}
