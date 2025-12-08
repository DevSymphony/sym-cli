package pylint

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

func TestParseOutput_Empty(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
	}{
		{"empty string", ""},
		{"empty array", "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &linter.ToolOutput{
				Stdout:   tt.stdout,
				ExitCode: 0,
			}

			violations, err := parseOutput(output)
			if err != nil {
				t.Errorf("parseOutput() error = %v", err)
			}
			if len(violations) != 0 {
				t.Errorf("parseOutput() returned %d violations, want 0", len(violations))
			}
		})
	}
}

func TestParseOutput_SingleViolation(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout: `[{
			"type": "error",
			"module": "mymodule",
			"obj": "MyClass.my_method",
			"line": 15,
			"column": 4,
			"endLine": 15,
			"endColumn": 10,
			"path": "src/mymodule.py",
			"symbol": "undefined-variable",
			"message": "Undefined variable 'foo'",
			"message-id": "E0602"
		}]`,
		ExitCode: 2,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Fatalf("parseOutput() error = %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("parseOutput() returned %d violations, want 1", len(violations))
	}

	v := violations[0]
	if v.File != "src/mymodule.py" {
		t.Errorf("File = %q, want %q", v.File, "src/mymodule.py")
	}
	if v.Line != 15 {
		t.Errorf("Line = %d, want %d", v.Line, 15)
	}
	if v.Column != 4 {
		t.Errorf("Column = %d, want %d", v.Column, 4)
	}
	if v.Message != "Undefined variable 'foo'" {
		t.Errorf("Message = %q, want %q", v.Message, "Undefined variable 'foo'")
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want %q", v.Severity, "error")
	}
	if v.RuleID != "E0602/undefined-variable" {
		t.Errorf("RuleID = %q, want %q", v.RuleID, "E0602/undefined-variable")
	}
}

func TestParseOutput_MultipleViolations(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout: `[
			{
				"type": "convention",
				"module": "test",
				"line": 1,
				"column": 0,
				"path": "test.py",
				"symbol": "missing-module-docstring",
				"message": "Missing module docstring",
				"message-id": "C0114"
			},
			{
				"type": "warning",
				"module": "test",
				"line": 5,
				"column": 4,
				"path": "test.py",
				"symbol": "unused-variable",
				"message": "Unused variable 'x'",
				"message-id": "W0612"
			},
			{
				"type": "refactor",
				"module": "test",
				"line": 10,
				"column": 0,
				"path": "test.py",
				"symbol": "too-many-branches",
				"message": "Too many branches (15/12)",
				"message-id": "R0912"
			}
		]`,
		ExitCode: 28,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Fatalf("parseOutput() error = %v", err)
	}

	if len(violations) != 3 {
		t.Fatalf("parseOutput() returned %d violations, want 3", len(violations))
	}
}

func TestPylintTypeToSeverity(t *testing.T) {
	tests := []struct {
		pylintType string
		want       string
	}{
		{"fatal", "error"},
		{"error", "error"},
		{"warning", "warning"},
		{"convention", "warning"},
		{"refactor", "warning"},
		{"info", "info"},
		{"information", "info"},
		{"FATAL", "error"},
		{"ERROR", "error"},
		{"WARNING", "warning"},
		{"CONVENTION", "warning"},
		{"unknown", "warning"},
		{"", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.pylintType, func(t *testing.T) {
			got := pylintTypeToSeverity(tt.pylintType)
			if got != tt.want {
				t.Errorf("pylintTypeToSeverity(%q) = %q, want %q", tt.pylintType, got, tt.want)
			}
		})
	}
}

func TestFormatRuleID(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		symbol    string
		want      string
	}{
		{"both present", "C0116", "missing-function-docstring", "C0116/missing-function-docstring"},
		{"only message ID", "C0116", "", "C0116"},
		{"only symbol", "", "missing-function-docstring", "missing-function-docstring"},
		{"neither", "", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRuleID(tt.messageID, tt.symbol)
			if got != tt.want {
				t.Errorf("formatRuleID(%q, %q) = %q, want %q", tt.messageID, tt.symbol, got, tt.want)
			}
		})
	}
}

func TestParseOutput_InvalidJSON(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "not valid json",
		ExitCode: 1,
	}

	_, err := parseOutput(output)
	if err == nil {
		t.Error("parseOutput() expected error for invalid JSON")
	}
}

func TestParseOutput_WithStderr(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "invalid",
		Stderr:   "pylint: error: no such module",
		ExitCode: 1,
	}

	_, err := parseOutput(output)
	if err == nil {
		t.Error("parseOutput() expected error")
	}
}
