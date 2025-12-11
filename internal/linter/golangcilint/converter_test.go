package golangcilint

import (
	"context"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// mockProvider is a mock LLM provider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Execute(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Close() error {
	return nil
}

func TestNewConverter(t *testing.T) {
	c := NewConverter()
	assert.NotNil(t, c)
}

func TestConverter_Name(t *testing.T) {
	c := NewConverter()
	assert.Equal(t, "golangci-lint", c.Name())
}

func TestConverter_SupportedLanguages(t *testing.T) {
	c := NewConverter()
	assert.Equal(t, []string{"go"}, c.SupportedLanguages())
}

func TestConverter_GetLLMDescription(t *testing.T) {
	c := NewConverter()
	desc := c.GetLLMDescription()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "golangci-lint")
	assert.Contains(t, desc, "Go")
}

func TestConverter_GetRoutingHints(t *testing.T) {
	c := NewConverter()
	hints := c.GetRoutingHints()
	assert.NotEmpty(t, hints)
	assert.True(t, len(hints) > 0)

	// Check that at least one hint mentions Go
	hasGoHint := false
	for _, hint := range hints {
		if assert.Contains(t, hint, "Go") {
			hasGoHint = true
			break
		}
	}
	assert.True(t, hasGoHint, "At least one hint should mention Go")
}

func TestConverter_ConvertSingleRule_Success(t *testing.T) {
	c := NewConverter()

	mockLLM := &mockProvider{
		response: `{
			"linter": "errcheck",
			"settings": {"check-type-assertions": true}
		}`,
	}

	rule := schema.UserRule{
		ID:  "rule-1",
		Say: "Check for unchecked errors",
	}

	result, err := c.ConvertSingleRule(context.Background(), rule, mockLLM)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "rule-1", result.RuleID)

	data, ok := result.Data.(golangciLinterData)
	require.True(t, ok)
	assert.Equal(t, "errcheck", data.Linter)
	assert.NotEmpty(t, data.Settings)
}

func TestConverter_ConvertSingleRule_EmptyLinter(t *testing.T) {
	c := NewConverter()

	mockLLM := &mockProvider{
		response: `{
			"linter": "",
			"settings": {}
		}`,
	}

	rule := schema.UserRule{
		ID:  "rule-1",
		Say: "File names must be snake_case",
	}

	result, err := c.ConvertSingleRule(context.Background(), rule, mockLLM)
	require.NoError(t, err)
	assert.Nil(t, result, "Should return nil when linter is empty")
}

func TestConverter_ConvertSingleRule_InvalidLinter(t *testing.T) {
	c := NewConverter()

	mockLLM := &mockProvider{
		response: `{
			"linter": "invalid-linter-name",
			"settings": {}
		}`,
	}

	rule := schema.UserRule{
		ID:  "rule-1",
		Say: "Some rule",
	}

	result, err := c.ConvertSingleRule(context.Background(), rule, mockLLM)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid linter name")
}

func TestConverter_ConvertSingleRule_NilProvider(t *testing.T) {
	c := NewConverter()

	rule := schema.UserRule{
		ID:  "rule-1",
		Say: "Check for unchecked errors",
	}

	result, err := c.ConvertSingleRule(context.Background(), rule, nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "LLM provider is required")
}

func TestConverter_BuildConfig_Success(t *testing.T) {
	c := NewConverter()

	results := []*linter.SingleRuleResult{
		{
			RuleID: "rule-1",
			Data: golangciLinterData{
				Linter:   "errcheck",
				Settings: map[string]interface{}{"check-type-assertions": true},
			},
		},
		{
			RuleID: "rule-2",
			Data: golangciLinterData{
				Linter:   "gocyclo",
				Settings: map[string]interface{}{"min-complexity": 10},
			},
		},
	}

	config, err := c.BuildConfig(results)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, ".golangci.yml", config.Filename)
	assert.Equal(t, "yaml", config.Format)
	assert.NotEmpty(t, config.Content)

	// Parse YAML to verify structure
	var parsedConfig golangciConfig
	err = yaml.Unmarshal(config.Content, &parsedConfig)
	require.NoError(t, err)

	assert.Equal(t, "2", parsedConfig.Version)
	assert.Contains(t, parsedConfig.Linters.Enable, "errcheck")
	assert.Contains(t, parsedConfig.Linters.Enable, "gocyclo")
	assert.Len(t, parsedConfig.Linters.Enable, 2)
}

func TestConverter_BuildConfig_Empty(t *testing.T) {
	c := NewConverter()

	config, err := c.BuildConfig([]*linter.SingleRuleResult{})
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestConverter_BuildConfig_InvalidData(t *testing.T) {
	c := NewConverter()

	results := []*linter.SingleRuleResult{
		{
			RuleID: "rule-1",
			Data:   "invalid data type",
		},
	}

	config, err := c.BuildConfig(results)
	require.NoError(t, err)
	assert.Nil(t, config, "Should return nil when no valid results")
}

func TestValidateConfig_Success(t *testing.T) {
	config := &golangciConfig{
		Version: "2",
		Linters: golangciLinters{
			Enable: []string{"errcheck", "govet"},
		},
	}

	err := validateConfig(config)
	assert.NoError(t, err)
}

func TestValidateConfig_NoVersion(t *testing.T) {
	config := &golangciConfig{
		Linters: golangciLinters{
			Enable: []string{"errcheck"},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidateConfig_NoLinters(t *testing.T) {
	config := &golangciConfig{
		Version: "2",
		Linters: golangciLinters{
			Enable: []string{},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no linters enabled")
}

func TestIsValidLinter(t *testing.T) {
	tests := []struct {
		name     string
		linter   string
		expected bool
	}{
		{"errcheck", "errcheck", true},
		{"govet", "govet", true},
		{"staticcheck", "staticcheck", true},
		{"gosec", "gosec", true},
		{"gocyclo", "gocyclo", true},
		{"invalid", "invalid-linter", false},
		{"empty", "", false},
		{"case insensitive", "ERRCHECK", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidLinter(tt.linter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompileTimeInterfaceCheck_Converter(t *testing.T) {
	var _ linter.Converter = (*Converter)(nil)
	// If this compiles, the interface is correctly implemented
}
