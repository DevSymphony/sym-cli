package llm

import (
	"context"
	"testing"

	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestFallbackInference(t *testing.T) {
	// Create inferencer without client (will use fallback)
	inferencer := NewInferencer(nil)

	tests := []struct {
		name           string
		rule           schema.UserRule
		expectedEngine string
		minConfidence  float64
	}{
		{
			name: "naming rule with PascalCase",
			rule: schema.UserRule{
				Say:      "Class names must be PascalCase",
				Category: "naming",
				Params: map[string]any{
					"case": "PascalCase",
				},
			},
			expectedEngine: "pattern",
			minConfidence:  0.6,
		},
		{
			name: "length rule with max",
			rule: schema.UserRule{
				Say:      "Maximum line length is 100 characters",
				Category: "length",
				Params: map[string]any{
					"max": 100,
				},
			},
			expectedEngine: "length",
			minConfidence:  0.6,
		},
		{
			name: "style rule with indent",
			rule: schema.UserRule{
				Say:      "Use 4 spaces for indentation",
				Category: "style",
				Params: map[string]any{
					"indent": 4,
				},
			},
			expectedEngine: "style",
			minConfidence:  0.6,
		},
		{
			name: "security rule",
			rule: schema.UserRule{
				Say:      "No hardcoded secrets",
				Category: "security",
			},
			expectedEngine: "pattern",
			minConfidence:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := inferencer.InferFromUserRule(context.Background(), &tt.rule)
			assert.NoError(t, err)
			assert.NotNil(t, result.Intent)
			assert.Equal(t, tt.expectedEngine, result.Intent.Engine)
			assert.GreaterOrEqual(t, result.Intent.Confidence, tt.minConfidence)
		})
	}
}

func TestInferenceCache(t *testing.T) {
	inferencer := NewInferencer(nil)

	rule := schema.UserRule{
		Say:      "Class names must be PascalCase",
		Category: "naming",
	}

	// First inference
	result1, err := inferencer.InferFromUserRule(context.Background(), &rule)
	assert.NoError(t, err)
	assert.False(t, result1.UsedCache)

	// Second inference (should use cache)
	result2, err := inferencer.InferFromUserRule(context.Background(), &rule)
	assert.NoError(t, err)
	assert.True(t, result2.UsedCache)
	assert.Equal(t, result1.Intent.Engine, result2.Intent.Engine)
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		keywords []string
		expected bool
	}{
		{
			name:     "contains keyword",
			s:        "class names must be pascalcase",
			keywords: []string{"class", "method"},
			expected: true,
		},
		{
			name:     "does not contain keyword",
			s:        "use tabs for indentation",
			keywords: []string{"spaces", "semicolon"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.keywords)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractNumber(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected int
	}{
		{
			name:     "extract from beginning",
			s:        "100 characters maximum",
			expected: 100,
		},
		{
			name:     "extract from middle",
			s:        "maximum line length is 80",
			expected: 0, // fmt.Sscanf only gets first number
		},
		{
			name:     "no number",
			s:        "use spaces for indentation",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNumber(tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}
