package tsc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// TSCDiagnostic represents a TypeScript diagnostic in JSON format.
type TSCDiagnostic struct {
	File struct {
		FileName string `json:"fileName"`
	} `json:"file"`
	Start    int    `json:"start"`
	Length   int    `json:"length"`
	Category int    `json:"category"` // 0=message, 1=error, 2=warning, 3=suggestion
	Code     int    `json:"code"`
	Message  string `json:"messageText"`
	Line     int    `json:"line"`   // Custom field we add
	Column   int    `json:"column"` // Custom field we add
}

// parseOutput parses tsc output and converts it to violations.
// TSC output format (without --pretty):
//
//	file.ts(line,col): error TS2304: Message here.
func parseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	if output == nil {
		return []linter.Violation{}, nil
	}

	// Try JSON format first (if we use --diagnostics or custom formatter)
	if strings.HasPrefix(strings.TrimSpace(output.Stdout), "[") {
		return parseJSONOutput(output.Stdout)
	}

	// Fall back to text format parsing
	return parseTextOutput(output.Stdout)
}

// parseTextOutput parses tsc text output.
// Format: src/main.ts(10,5): error TS2304: Cannot find name 'foo'.
func parseTextOutput(text string) ([]linter.Violation, error) {
	lines := strings.Split(text, "\n")
	violations := make([]linter.Violation, 0)

	// Regex to match tsc output:
	// file.ts(line,col): severity TScode: message
	re := regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s+(error|warning|suggestion)\s+TS(\d+):\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) != 7 {
			continue
		}

		file := matches[1]
		lineNum, _ := strconv.Atoi(matches[2])
		col, _ := strconv.Atoi(matches[3])
		severity := matches[4]
		code := matches[5]
		message := matches[6]

		violations = append(violations, linter.Violation{
			File:     file,
			Line:     lineNum,
			Column:   col,
			Message:  message,
			Severity: mapSeverity(severity),
			RuleID:   fmt.Sprintf("TS%s", code),
		})
	}

	return violations, nil
}

// parseJSONOutput parses tsc JSON output (if we implement custom formatter).
func parseJSONOutput(jsonStr string) ([]linter.Violation, error) {
	var diagnostics []TSCDiagnostic
	if err := json.Unmarshal([]byte(jsonStr), &diagnostics); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	violations := make([]linter.Violation, len(diagnostics))
	for i, diag := range diagnostics {
		violations[i] = linter.Violation{
			File:     diag.File.FileName,
			Line:     diag.Line,
			Column:   diag.Column,
			Message:  diag.Message,
			Severity: mapCategory(diag.Category),
			RuleID:   fmt.Sprintf("TS%d", diag.Code),
		}
	}

	return violations, nil
}

// mapSeverity maps tsc severity string to standard severity.
func mapSeverity(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "warning":
		return "warning"
	case "suggestion":
		return "info"
	default:
		return "error"
	}
}

// mapCategory maps tsc category number to severity.
func mapCategory(category int) string {
	switch category {
	case 1: // Error
		return "error"
	case 2: // Warning
		return "warning"
	case 3: // Suggestion
		return "info"
	default: // Message
		return "info"
	}
}
