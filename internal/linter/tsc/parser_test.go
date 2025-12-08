package tsc

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

func TestParseTextOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int // number of violations
		wantErr bool
	}{
		{
			name:  "single error",
			input: "src/main.ts(10,5): error TS2304: Cannot find name 'foo'.",
			want:  1,
		},
		{
			name: "multiple errors",
			input: `src/main.ts(10,5): error TS2304: Cannot find name 'foo'.
src/app.ts(20,10): error TS2339: Property 'bar' does not exist on type 'Object'.`,
			want: 2,
		},
		{
			name:  "warning",
			input: "src/util.ts(5,1): warning TS6133: 'unused' is declared but its value is never read.",
			want:  1,
		},
		{
			name:  "suggestion",
			input: "src/index.ts(1,1): suggestion TS80001: File is a CommonJS module.",
			want:  1,
		},
		{
			name:  "empty output",
			input: "",
			want:  0,
		},
		{
			name:  "no violations",
			input: "Compilation complete. No errors.",
			want:  0,
		},
		{
			name: "mixed severity",
			input: `src/main.ts(10,5): error TS2304: Cannot find name 'foo'.
src/app.ts(20,10): warning TS6133: 'bar' is declared but never used.
src/util.ts(30,15): suggestion TS80001: Consider using const.`,
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTextOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTextOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("parseTextOutput() returned %d violations, want %d", len(got), tt.want)
			}

			// Verify first violation details if present
			if len(got) > 0 && len(tt.input) > 0 {
				first := got[0]
				if first.File == "" {
					t.Error("First violation should have a file")
				}
				if first.Line == 0 {
					t.Error("First violation should have a line number")
				}
				if first.Message == "" {
					t.Error("First violation should have a message")
				}
			}
		})
	}
}

func TestParseTextOutput_Details(t *testing.T) {
	input := "src/main.ts(10,5): error TS2304: Cannot find name 'foo'."
	violations, err := parseTextOutput(input)

	if err != nil {
		t.Fatalf("parseTextOutput() unexpected error: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]

	if v.File != "src/main.ts" {
		t.Errorf("File = %q, want %q", v.File, "src/main.ts")
	}

	if v.Line != 10 {
		t.Errorf("Line = %d, want %d", v.Line, 10)
	}

	if v.Column != 5 {
		t.Errorf("Column = %d, want %d", v.Column, 5)
	}

	if v.Severity != "error" {
		t.Errorf("Severity = %q, want %q", v.Severity, "error")
	}

	if v.RuleID != "TS2304" {
		t.Errorf("RuleID = %q, want %q", v.RuleID, "TS2304")
	}

	if v.Message != "Cannot find name 'foo'." {
		t.Errorf("Message = %q, want %q", v.Message, "Cannot find name 'foo'.")
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"error", "error"},
		{"warning", "warning"},
		{"suggestion", "info"},
		{"unknown", "error"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapSeverity(tt.input)
			if got != tt.want {
				t.Errorf("mapSeverity(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapCategory(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "info"},    // Message
		{1, "error"},   // Error
		{2, "warning"}, // Warning
		{3, "info"},    // Suggestion
		{99, "info"},   // Unknown (default)
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.input)), func(t *testing.T) {
			got := mapCategory(tt.input)
			if got != tt.want {
				t.Errorf("mapCategory(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseOutput_EmptyOutput(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Errorf("parseOutput() error = %v, want nil", err)
	}

	if len(violations) != 0 {
		t.Errorf("parseOutput() returned %d violations, want 0", len(violations))
	}
}

func TestParseOutput_NilOutput(t *testing.T) {
	violations, err := parseOutput(nil)
	if err != nil {
		t.Errorf("parseOutput() error = %v, want nil", err)
	}

	if len(violations) != 0 {
		t.Errorf("parseOutput() returned %d violations, want 0", len(violations))
	}
}

func TestParseOutput_RealWorldExample(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout: `src/index.ts(15,7): error TS2322: Type 'string' is not assignable to type 'number'.
src/utils/helper.ts(42,15): error TS2339: Property 'nonExistent' does not exist on type 'MyType'.
src/components/Button.tsx(8,3): warning TS6133: 'props' is declared but its value is never read.`,
		Stderr:   "",
		ExitCode: 2,
	}

	violations, err := parseOutput(output)
	if err != nil {
		t.Fatalf("parseOutput() unexpected error: %v", err)
	}

	if len(violations) != 3 {
		t.Fatalf("Expected 3 violations, got %d", len(violations))
	}

	// Verify first violation
	if violations[0].File != "src/index.ts" {
		t.Errorf("First violation file = %q, want %q", violations[0].File, "src/index.ts")
	}
	if violations[0].Severity != "error" {
		t.Errorf("First violation severity = %q, want %q", violations[0].Severity, "error")
	}

	// Verify third violation (warning)
	if violations[2].Severity != "warning" {
		t.Errorf("Third violation severity = %q, want %q", violations[2].Severity, "warning")
	}
	if violations[2].RuleID != "TS6133" {
		t.Errorf("Third violation RuleID = %q, want %q", violations[2].RuleID, "TS6133")
	}
}
