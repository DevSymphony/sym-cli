package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIEngine(t *testing.T) {
	t.Run("with api key", func(t *testing.T) {
		engine := NewAPIEngine("sk-test-key")
		assert.NotNil(t, engine)
		assert.Equal(t, "openai-api", engine.Name())
		assert.True(t, engine.IsAvailable())
	})

	t.Run("without api key", func(t *testing.T) {
		engine := NewAPIEngine("")
		assert.NotNil(t, engine)
		assert.False(t, engine.IsAvailable())
	})

	t.Run("with options", func(t *testing.T) {
		engine := NewAPIEngine("sk-test-key",
			WithAPIFastModel("gpt-4o"),
			WithAPIPowerModel("o3-mini"),
			WithAPIVerbose(true),
		)
		assert.NotNil(t, engine)
		caps := engine.Capabilities()
		assert.Contains(t, caps.Models, "gpt-4o")
		assert.Contains(t, caps.Models, "o3-mini")
	})
}

func TestAPIEngine_Capabilities(t *testing.T) {
	engine := NewAPIEngine("sk-test-key")
	caps := engine.Capabilities()

	assert.True(t, caps.SupportsTemperature)
	assert.True(t, caps.SupportsMaxTokens)
	assert.True(t, caps.SupportsComplexity)
	assert.True(t, caps.SupportsStreaming)
	assert.Equal(t, 128000, caps.MaxContextLength)
	assert.Len(t, caps.Models, 2)
}

func TestAPIEngine_Name(t *testing.T) {
	engine := NewAPIEngine("sk-test-key")
	assert.Equal(t, "openai-api", engine.Name())
}

func TestAPIEngine_IsAvailable(t *testing.T) {
	t.Run("available with key", func(t *testing.T) {
		engine := NewAPIEngine("sk-test-key")
		assert.True(t, engine.IsAvailable())
	})

	t.Run("not available without key", func(t *testing.T) {
		engine := NewAPIEngine("")
		assert.False(t, engine.IsAvailable())
	})
}

