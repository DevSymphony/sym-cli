package eslint

import (
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// ESLintOutput represents ESLint JSON output format.
// ESLint outputs an array of file results.
type ESLintOutput []ESLintFileResult

// ESLintFileResult represents results for a single file.
type ESLintFileResult struct {
	FilePath string          `json:"filePath"`
	Messages []ESLintMessage `json:"messages"`
}

// ESLintMessage represents a single violation.
type ESLintMessage struct {
	RuleID    string `json:"ruleId"`
	Severity  int    `json:"severity"` // 0=off, 1=warn, 2=error
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
}

// parseOutput converts ESLint JSON output to violations.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	if output.Stdout == "" || output.Stdout == "[]" {
		return nil, nil // No violations
	}

	var eslintOutput ESLintOutput
	if err := json.Unmarshal([]byte(output.Stdout), &eslintOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ESLint output: %w", err)
	}

	var violations []adapter.Violation

	for _, fileResult := range eslintOutput {
		for _, msg := range fileResult.Messages {
			violations = append(violations, adapter.Violation{
				File:     fileResult.FilePath,
				Line:     msg.Line,
				Column:   msg.Column,
				Message:  msg.Message,
				Severity: severityToString(msg.Severity),
				RuleID:   msg.RuleID,
			})
		}
	}

	return violations, nil
}

// severityToString converts ESLint severity to string.
func severityToString(severity int) string {
	switch severity {
	case 2:
		return "error"
	case 1:
		return "warning"
	default:
		return "info"
	}
}
