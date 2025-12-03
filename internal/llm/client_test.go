package llm

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		client := NewClient()
		assert.NotNil(t, client)
		assert.NotNil(t, client.GetConfig())
	})

	t.Run("with_options_and_config", func(t *testing.T) {
		cfg := &LLMConfig{
			Backend: engine.ModeAPI,
			APIKey:  "sk-test",
		}
		client := NewClient(WithConfig(cfg), WithVerbose(true))
		assert.NotNil(t, client)
		assert.Equal(t, engine.ModeAPI, client.config.Backend)
	})

	t.Run("with_mode_option", func(t *testing.T) {
		client := NewClient(WithMode(engine.ModeAPI))
		assert.NotNil(t, client)
	})
}

func TestClient_GetActiveEngine(t *testing.T) {
	t.Run("with API engine", func(t *testing.T) {
		cfg := &LLMConfig{
			Backend: engine.ModeAPI,
			APIKey:  "sk-test",
		}
		client := NewClient(WithConfig(cfg))
		eng := client.GetActiveEngine()
		assert.NotNil(t, eng)
		assert.Equal(t, "openai-api", eng.Name())
	})

	t.Run("no engine available", func(t *testing.T) {
		cfg := &LLMConfig{
			Backend: engine.ModeAPI,
			// No API key
		}
		client := NewClient(WithConfig(cfg))
		eng := client.GetActiveEngine()
		assert.Nil(t, eng)
	})
}

func TestRequestBuilder(t *testing.T) {
	client := NewClient()

	t.Run("basic request", func(t *testing.T) {
		builder := client.Request("system", "user")
		assert.NotNil(t, builder)
	})

	t.Run("with complexity", func(t *testing.T) {
		builder := client.Request("system", "user").
			WithComplexity(engine.ComplexityHigh)
		assert.NotNil(t, builder)
	})

	t.Run("with max tokens", func(t *testing.T) {
		builder := client.Request("system", "user").
			WithMaxTokens(2000)
		assert.NotNil(t, builder)
	})

	t.Run("with temperature", func(t *testing.T) {
		builder := client.Request("system", "user").
			WithTemperature(0.7)
		assert.NotNil(t, builder)
	})

	t.Run("chained options", func(t *testing.T) {
		builder := client.Request("system", "user").
			WithComplexity(engine.ComplexityMedium).
			WithMaxTokens(1500).
			WithTemperature(0.8)
		assert.NotNil(t, builder)
	})
}

func TestModeConstants(t *testing.T) {
	// Verify backward compatibility
	assert.Equal(t, engine.ModeAPI, ModeAPI)
	assert.Equal(t, engine.ModeMCP, ModeMCP)
	assert.Equal(t, engine.ModeCLI, ModeCLI)
	assert.Equal(t, engine.ModeAuto, ModeAuto)
}
