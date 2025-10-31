package linters

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESLintConverter_Convert_Pattern(t *testing.T) {
	converter := NewESLintConverter(false)

	userRule := &schema.UserRule{
		ID:       "test-naming",
		Say:      "Class names must be PascalCase",
		Category: "naming",
		Severity: "error",
	}

	intent := &llm.RuleIntent{
		Engine:   "pattern",
		Category: "naming",
		Target:   "identifier",
		Params: map[string]any{
			"case": "PascalCase",
		},
		Confidence: 0.9,
	}

	result, err := converter.Convert(userRule, intent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Severity)
	assert.NotEmpty(t, result.Config)
	assert.Contains(t, result.Config, "id-match")
}

func TestESLintConverter_Convert_Length(t *testing.T) {
	converter := NewESLintConverter(false)

	userRule := &schema.UserRule{
		ID:       "test-max-line",
		Say:      "Maximum line length is 100 characters",
		Category: "length",
		Severity: "error",
	}

	intent := &llm.RuleIntent{
		Engine:   "length",
		Category: "length",
		Scope:    "line",
		Params: map[string]any{
			"max": 100,
		},
		Confidence: 0.9,
	}

	result, err := converter.Convert(userRule, intent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Config, "max-len")
}

func TestESLintConverter_Convert_Style(t *testing.T) {
	converter := NewESLintConverter(false)

	userRule := &schema.UserRule{
		ID:       "test-indent",
		Say:      "Use 4 spaces for indentation",
		Category: "style",
		Severity: "error",
	}

	intent := &llm.RuleIntent{
		Engine:   "style",
		Category: "style",
		Params: map[string]any{
			"indent": 4,
		},
		Confidence: 0.9,
	}

	result, err := converter.Convert(userRule, intent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Config, "indent")
}

func TestESLintConverter_GenerateConfig(t *testing.T) {
	converter := NewESLintConverter(false)

	rules := []*LinterRule{
		{
			ID:       "rule-1",
			Severity: "error",
			Config: map[string]any{
				"indent": []any{"error", 4},
			},
			Comment: "Use 4 spaces",
		},
		{
			ID:       "rule-2",
			Severity: "error",
			Config: map[string]any{
				"max-len": []any{"error", map[string]any{"code": 100}},
			},
			Comment: "Max line length 100",
		},
	}

	config, err := converter.GenerateConfig(rules)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, ".eslintrc.json", config.Filename)
	assert.NotEmpty(t, config.Content)
}

func TestESLintConverter_MapSeverity(t *testing.T) {
	converter := NewESLintConverter(false)

	tests := []struct {
		input    string
		expected string
	}{
		{"error", "error"},
		{"warning", "warn"},
		{"warn", "warn"},
		{"info", "off"},
		{"off", "off"},
		{"unknown", "error"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := converter.mapSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestESLintConverter_CaseToRegex(t *testing.T) {
	converter := NewESLintConverter(false)

	tests := []struct {
		caseStyle string
		expected  string
	}{
		{"PascalCase", "^[A-Z][a-zA-Z0-9]*$"},
		{"camelCase", "^[a-z][a-zA-Z0-9]*$"},
		{"snake_case", "^[a-z][a-z0-9_]*$"},
		{"SCREAMING_SNAKE_CASE", "^[A-Z][A-Z0-9_]*$"},
		{"kebab-case", "^[a-z][a-z0-9-]*$"},
	}

	for _, tt := range tests {
		t.Run(tt.caseStyle, func(t *testing.T) {
			result := converter.caseToRegex(tt.caseStyle)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestESLintConverter_SupportedLanguages(t *testing.T) {
	converter := NewESLintConverter(false)
	langs := converter.SupportedLanguages()

	assert.Contains(t, langs, "javascript")
	assert.Contains(t, langs, "typescript")
}
