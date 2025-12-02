package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComplexity_String(t *testing.T) {
	tests := []struct {
		name       string
		complexity Complexity
		want       string
	}{
		{"low", ComplexityLow, "low"},
		{"medium", ComplexityMedium, "medium"},
		{"high", ComplexityHigh, "high"},
		{"unknown", Complexity(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.complexity.String())
		})
	}
}

func TestRequest_CombinedPrompt(t *testing.T) {
	tests := []struct {
		name   string
		req    Request
		want   string
	}{
		{
			name: "with system and user prompt",
			req: Request{
				SystemPrompt: "You are a helpful assistant.",
				UserPrompt:   "Hello!",
			},
			want: "You are a helpful assistant.\n\nHello!",
		},
		{
			name: "only user prompt",
			req: Request{
				SystemPrompt: "",
				UserPrompt:   "Hello!",
			},
			want: "Hello!",
		},
		{
			name: "empty prompts",
			req: Request{
				SystemPrompt: "",
				UserPrompt:   "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.req.CombinedPrompt())
		})
	}
}

func TestMode_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		mode  Mode
		valid bool
	}{
		{"auto", ModeAuto, true},
		{"mcp", ModeMCP, true},
		{"cli", ModeCLI, true},
		{"api", ModeAPI, true},
		{"invalid", Mode("invalid"), false},
		{"empty", Mode(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.mode.IsValid())
		})
	}
}

func TestCapabilities_Default(t *testing.T) {
	caps := Capabilities{}

	assert.False(t, caps.SupportsTemperature)
	assert.False(t, caps.SupportsMaxTokens)
	assert.False(t, caps.SupportsComplexity)
	assert.False(t, caps.SupportsStreaming)
	assert.Equal(t, 0, caps.MaxContextLength)
	assert.Nil(t, caps.Models)
}

