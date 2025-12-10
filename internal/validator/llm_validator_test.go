package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractAddedLines is now tested in internal/git/changes_test.go

func TestParseValidationResponse_NoViolation(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "explicit false",
			response: `{"violates": false, "description": "", "suggestion": ""}`,
		},
		{
			name:     "does not violate text",
			response: `The code does not violate the convention.`,
		},
		{
			name:     "no violation text",
			response: `No violation found in this code.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValidationResponse(tt.response)
			assert.False(t, result.Violates, "should not violate")
		})
	}
}

func TestParseValidationResponse_WithViolation(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		expectDesc bool
		expectSugg bool
	}{
		{
			name:       "with description and suggestion",
			response:   `{"violates": true, "description": "Hardcoded API key found", "suggestion": "Use environment variables"}`,
			expectDesc: true,
			expectSugg: true,
		},
		{
			name:       "with description only",
			response:   `{"violates": true, "description": "Security issue detected", "suggestion": ""}`,
			expectDesc: true,
			expectSugg: false,
		},
		{
			name:       "minimal violation",
			response:   `{"violates": true}`,
			expectDesc: true, // Should have default description
			expectSugg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseValidationResponse(tt.response)
			assert.True(t, result.Violates, "should violate")

			if tt.expectDesc {
				assert.NotEmpty(t, result.Description, "should have description")
			}

			if tt.expectSugg {
				assert.NotEmpty(t, result.Suggestion, "should have suggestion")
			}
		})
	}
}

func TestExtractJSONField(t *testing.T) {
	tests := []struct {
		name     string
		response string
		field    string
		expected string
	}{
		{
			name:     "simple field",
			response: `{"description": "test message"}`,
			field:    "description",
			expected: "test message",
		},
		{
			name:     "field with spaces",
			response: `{"description": "test message with spaces"}`,
			field:    "description",
			expected: "test message with spaces",
		},
		{
			name:     "nested in response",
			response: `Some text before {"description": "found it"} some text after`,
			field:    "description",
			expected: "found it",
		},
		{
			name:     "field not found",
			response: `{"other": "value"}`,
			field:    "description",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONField(tt.response, tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

