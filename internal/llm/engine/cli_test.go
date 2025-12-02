package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLIEngine(t *testing.T) {
	t.Run("valid provider", func(t *testing.T) {
		engine, err := NewCLIEngine(ProviderClaude)
		require.NoError(t, err)
		assert.NotNil(t, engine)
		assert.Equal(t, "cli-claude", engine.Name())
	})

	t.Run("with options", func(t *testing.T) {
		engine, err := NewCLIEngine(
			ProviderClaude,
			WithCLIModel("custom-model"),
			WithCLILargeModel("large-model"),
			WithCLIVerbose(true),
		)
		require.NoError(t, err)
		assert.Equal(t, "custom-model", engine.GetModel())
	})

	t.Run("invalid provider", func(t *testing.T) {
		_, err := NewCLIEngine(CLIProviderType("invalid"))
		assert.Error(t, err)
	})
}

func TestCLIEngine_Capabilities(t *testing.T) {
	engine, err := NewCLIEngine(ProviderClaude)
	require.NoError(t, err)

	caps := engine.Capabilities()

	assert.True(t, caps.SupportsMaxTokens)
	assert.False(t, caps.SupportsStreaming)
	assert.True(t, caps.SupportsComplexity) // Has LargeModel
	assert.NotEmpty(t, caps.Models)
}

func TestDetectAvailableCLIs(t *testing.T) {
	clis := DetectAvailableCLIs()

	// Should return info for all supported providers
	assert.Len(t, clis, 2)

	// Each CLI should have provider and name set
	for _, cli := range clis {
		assert.NotEmpty(t, cli.Provider)
		assert.NotEmpty(t, cli.Name)
	}
}

func TestGetProviderByCommand(t *testing.T) {
	t.Run("claude command", func(t *testing.T) {
		provider, err := GetProviderByCommand("claude")
		require.NoError(t, err)
		assert.Equal(t, ProviderClaude, provider.Type)
	})

	t.Run("gemini command", func(t *testing.T) {
		provider, err := GetProviderByCommand("gemini")
		require.NoError(t, err)
		assert.Equal(t, ProviderGemini, provider.Type)
	})

	t.Run("unknown command", func(t *testing.T) {
		_, err := GetProviderByCommand("unknown")
		assert.Error(t, err)
	})
}
