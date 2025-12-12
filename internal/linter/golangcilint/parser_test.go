package golangcilint

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOutput_Success(t *testing.T) {
	jsonOutput := `{
		"Issues": [
			{
				"FromLinter": "errcheck",
				"Text": "Error return value is not checked",
				"Severity": "error",
				"SourceLines": ["fmt.Println(\"hello\")"],
				"Pos": {
					"Filename": "/path/to/file.go",
					"Line": 10,
					"Column": 5
				}
			},
			{
				"FromLinter": "govet",
				"Text": "printf: missing argument for Sprintf",
				"Severity": "warning",
				"SourceLines": ["fmt.Sprintf(\"%s %s\", name)"],
				"Pos": {
					"Filename": "/path/to/other.go",
					"Line": 20,
					"Column": 10
				}
			}
		],
		"Report": {
			"Linters": [
				{"Name": "errcheck", "Enabled": true}
			]
		}
	}`

	output := &linter.ToolOutput{
		Stdout:   jsonOutput,
		Stderr:   "",
		ExitCode: 1,
		Duration: "1s",
	}

	violations, err := parseOutput(output)
	require.NoError(t, err)
	assert.Len(t, violations, 2)

	// First violation
	assert.Equal(t, "/path/to/file.go", violations[0].File)
	assert.Equal(t, 10, violations[0].Line)
	assert.Equal(t, 5, violations[0].Column)
	assert.Equal(t, "Error return value is not checked", violations[0].Message)
	assert.Equal(t, "error", violations[0].Severity)
	assert.Equal(t, "errcheck", violations[0].RuleID)

	// Second violation
	assert.Equal(t, "/path/to/other.go", violations[1].File)
	assert.Equal(t, 20, violations[1].Line)
	assert.Equal(t, 10, violations[1].Column)
	assert.Equal(t, "printf: missing argument for Sprintf", violations[1].Message)
	assert.Equal(t, "warning", violations[1].Severity)
	assert.Equal(t, "govet", violations[1].RuleID)
}

func TestParseOutput_Empty(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
		Duration: "0s",
	}

	violations, err := parseOutput(output)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestParseOutput_EmptyIssues(t *testing.T) {
	jsonOutput := `{
		"Issues": [],
		"Report": {
			"Linters": []
		}
	}`

	output := &linter.ToolOutput{
		Stdout:   jsonOutput,
		Stderr:   "",
		ExitCode: 0,
		Duration: "1s",
	}

	violations, err := parseOutput(output)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestParseOutput_InvalidJSON(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "not json",
		Stderr:   "",
		ExitCode: 2,
		Duration: "0s",
	}

	violations, err := parseOutput(output)
	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "failed to parse golangci-lint output")
}

func TestParseOutput_NilOutput(t *testing.T) {
	violations, err := parseOutput(nil)
	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "output is nil")
}

func TestParseOutput_StderrOnly(t *testing.T) {
	output := &linter.ToolOutput{
		Stdout:   "",
		Stderr:   "configuration error: unknown linter",
		ExitCode: 2,
		Duration: "0s",
	}

	violations, err := parseOutput(output)
	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "golangci-lint error")
	assert.Contains(t, err.Error(), "configuration error")
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"error", "error"},
		{"Error", "error"},
		{"ERROR", "error"},
		{"warning", "warning"},
		{"Warning", "warning"},
		{"info", "info"},
		{"Info", "info"},
		{"unknown", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
