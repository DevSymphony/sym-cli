package cliprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"claude", TypeClaude, true},
		{"gemini", TypeGemini, true},
		{"invalid", Type("invalid"), false},
		{"empty", Type(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.typ.IsValid())
		})
	}
}

func TestSupported(t *testing.T) {
	providers := Supported()

	assert.Len(t, providers, 2)
	assert.Contains(t, providers, TypeClaude)
	assert.Contains(t, providers, TypeGemini)
}

func TestGet(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		provider, err := Get(TypeClaude)
		require.NoError(t, err)
		assert.Equal(t, TypeClaude, provider.Type)
		assert.Equal(t, "Claude CLI", provider.DisplayName)
		assert.Equal(t, "claude", provider.Command)
	})

	t.Run("gemini", func(t *testing.T) {
		provider, err := Get(TypeGemini)
		require.NoError(t, err)
		assert.Equal(t, TypeGemini, provider.Type)
		assert.Equal(t, "Gemini CLI", provider.DisplayName)
		assert.Equal(t, "gemini", provider.Command)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := Get(Type("invalid"))
		assert.Error(t, err)
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		provider := newClaudeProvider()
		args := provider.BuildArgs("claude-3-opus", "Hello!")

		assert.Contains(t, args, "-p")
		assert.Contains(t, args, "Hello!")
		assert.Contains(t, args, "--model")
		assert.Contains(t, args, "claude-3-opus")
	})

	t.Run("gemini", func(t *testing.T) {
		provider := newGeminiProvider()
		args := provider.BuildArgs("gemini-pro", "Hello!")

		assert.Contains(t, args, "prompt")
		assert.Contains(t, args, "-m")
		assert.Contains(t, args, "gemini-pro")
	})
}

func TestParseResponse(t *testing.T) {
	providers := Supported()

	for typ, provider := range providers {
		t.Run(string(typ), func(t *testing.T) {
			resp, err := provider.ParseResponse([]byte("  trimmed response  \n"))
			require.NoError(t, err)
			assert.Equal(t, "trimmed response", resp)
		})
	}
}

func TestDetect(t *testing.T) {
	info := Detect()
	assert.Len(t, info, 2)

	for _, cli := range info {
		assert.NotEmpty(t, cli.Provider)
		assert.NotEmpty(t, cli.Name)
	}
}

func TestGetByCommand(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		provider, err := GetByCommand("claude")
		require.NoError(t, err)
		assert.Equal(t, TypeClaude, provider.Type)
	})

	t.Run("gemini", func(t *testing.T) {
		provider, err := GetByCommand("gemini")
		require.NoError(t, err)
		assert.Equal(t, TypeGemini, provider.Type)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := GetByCommand("unknown")
		assert.Error(t, err)
	})
}
