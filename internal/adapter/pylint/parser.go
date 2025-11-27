package pylint

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// PylintMessage represents a single Pylint JSON message.
// Pylint outputs an array of these messages when using --output-format=json.
type PylintMessage struct {
	Type      string `json:"type"`       // "convention", "refactor", "warning", "error", "fatal"
	Module    string `json:"module"`     // Module name
	Obj       string `json:"obj"`        // Object name (function, class, etc.)
	Line      int    `json:"line"`       // Line number
	Column    int    `json:"column"`     // Column number
	EndLine   int    `json:"endLine"`    // End line number (optional)
	EndColumn int    `json:"endColumn"`  // End column number (optional)
	Path      string `json:"path"`       // File path
	Symbol    string `json:"symbol"`     // Rule symbol (e.g., "missing-docstring")
	Message   string `json:"message"`    // Human-readable message
	MessageID string `json:"message-id"` // Message ID (e.g., "C0116")
}

// PylintOutput is an array of Pylint messages.
type PylintOutput []PylintMessage

// parseOutput converts Pylint JSON output to violations.
func parseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	if output.Stdout == "" || output.Stdout == "[]" {
		return nil, nil // No violations
	}

	var pylintOutput PylintOutput
	if err := json.Unmarshal([]byte(output.Stdout), &pylintOutput); err != nil {
		// If JSON parsing fails, check for errors in stderr
		if output.Stderr != "" {
			return nil, fmt.Errorf("pylint error: %s", output.Stderr)
		}
		return nil, fmt.Errorf("failed to parse Pylint output: %w", err)
	}

	var violations []adapter.Violation

	for _, msg := range pylintOutput {
		violations = append(violations, adapter.Violation{
			File:     msg.Path,
			Line:     msg.Line,
			Column:   msg.Column,
			Message:  msg.Message,
			Severity: pylintTypeToSeverity(msg.Type),
			RuleID:   formatRuleID(msg.MessageID, msg.Symbol),
		})
	}

	return violations, nil
}

// pylintTypeToSeverity converts Pylint message type to severity string.
// Pylint types: F (fatal), E (error), W (warning), C (convention), R (refactor)
func pylintTypeToSeverity(pylintType string) string {
	switch strings.ToLower(pylintType) {
	case "fatal", "error":
		return "error"
	case "warning":
		return "warning"
	case "convention", "refactor":
		return "warning"
	case "info", "information":
		return "info"
	default:
		return "warning"
	}
}

// formatRuleID creates a rule ID from message ID and symbol.
// Returns format like "C0116/missing-function-docstring"
func formatRuleID(messageID, symbol string) string {
	if messageID != "" && symbol != "" {
		return fmt.Sprintf("%s/%s", messageID, symbol)
	}
	if messageID != "" {
		return messageID
	}
	if symbol != "" {
		return symbol
	}
	return "unknown"
}
