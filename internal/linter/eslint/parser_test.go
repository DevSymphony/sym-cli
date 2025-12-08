package eslint

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

func TestParseOutput_Empty(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout: "[]",
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Fatalf("parseOutput failed: %v", err)
	}

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestParseOutput_WithViolations(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout: `[
			{
				"filePath": "src/app.js",
				"messages": [
					{
						"ruleId": "id-match",
						"severity": 2,
						"message": "Identifier 'myClass' does not match pattern",
						"line": 10,
						"column": 7
					},
					{
						"ruleId": "max-len",
						"severity": 2,
						"message": "Line exceeds maximum length",
						"line": 15,
						"column": 1
					}
				]
			}
		]`,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Fatalf("parseOutput failed: %v", err)
	}

	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(violations))
	}

	// Check first violation
	v := violations[0]
	if v.File != "src/app.js" {
		t.Errorf("file = %q, want %q", v.File, "src/app.js")
	}
	if v.Line != 10 {
		t.Errorf("line = %d, want 10", v.Line)
	}
	if v.Severity != "error" {
		t.Errorf("severity = %q, want %q", v.Severity, "error")
	}
	if v.RuleID != "id-match" {
		t.Errorf("ruleId = %q, want %q", v.RuleID, "id-match")
	}
}

func TestSeverityToString(t *testing.T) {
	tests := []struct {
		severity int
		want     string
	}{
		{0, "info"},
		{1, "warning"},
		{2, "error"},
	}

	for _, tt := range tests {
		got := severityToString(tt.severity)
		if got != tt.want {
			t.Errorf("severityToString(%d) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}
