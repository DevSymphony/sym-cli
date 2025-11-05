package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		response    string
		expectError bool
		checkFunc   func(*testing.T, *RuleIntent)
	}{
		{
			name: "valid JSON response",
			response: `{
				"engine": "pattern",
				"category": "naming",
				"target": "identifier",
				"scope": "file",
				"patterns": ["^[A-Z][a-zA-Z0-9]*$"],
				"params": {"case": "PascalCase"},
				"confidence": 0.95
			}`,
			expectError: false,
			checkFunc: func(t *testing.T, intent *RuleIntent) {
				assert.Equal(t, "pattern", intent.Engine)
				assert.Equal(t, "naming", intent.Category)
				assert.Equal(t, 0.95, intent.Confidence)
			},
		},
		{
			name: "JSON in markdown code block",
			response: "```json\n" + `{
				"engine": "style",
				"category": "formatting",
				"confidence": 0.8
			}` + "\n```",
			expectError: false,
			checkFunc: func(t *testing.T, intent *RuleIntent) {
				assert.Equal(t, "style", intent.Engine)
				assert.Equal(t, "formatting", intent.Category)
			},
		},
		{
			name:        "missing engine field",
			response:    `{"category": "naming", "confidence": 0.9}`,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			response:    `{invalid json}`,
			expectError: true,
		},
		{
			name: "zero confidence sets default",
			response: `{
				"engine": "pattern",
				"category": "naming"
			}`,
			expectError: false,
			checkFunc: func(t *testing.T, intent *RuleIntent) {
				assert.Equal(t, 0.5, intent.Confidence)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			intent, err := parseIntent(tt.response)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, intent)
			if tt.checkFunc != nil {
				tt.checkFunc(t, intent)
			}
		})
	}
}

func TestInferenceCache(t *testing.T) {
	t.Parallel()

	cache := newInferenceCache()

	// Cache miss
	_, ok := cache.Get("test-key")
	assert.False(t, ok)

	// Cache set
	intent := &RuleIntent{
		Engine:     "pattern",
		Category:   "naming",
		Confidence: 0.9,
	}
	cache.Set("test-key", intent)

	// Cache hit
	cached, ok := cache.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, "pattern", cached.Engine)
	assert.Equal(t, 0.9, cached.Confidence)
}
