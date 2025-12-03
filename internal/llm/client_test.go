package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockEngine is a test implementation of LLMEngine
type mockEngine struct {
	name      string
	available bool
}

func (m *mockEngine) Name() string                                  { return m.name }
func (m *mockEngine) IsAvailable() bool                             { return m.available }
func (m *mockEngine) Capabilities() Capabilities                    { return Capabilities{} }
func (m *mockEngine) Execute(_ context.Context, _ *Request) (string, error) { return "mock", nil }

func init() {
	// Register mock engine for testing
	Register(&Registration{
		Name:     "mock",
		Priority: 100,
		Factory: func(cfg *EngineConfig) (LLMEngine, error) {
			// Return available engine only if API key is set
			if cfg.APIKey != "" {
				return &mockEngine{name: "mock", available: true}, nil
			}
			return &mockEngine{name: "mock", available: false}, nil
		},
	})
}

func TestNewClient(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		client := NewClient()
		assert.NotNil(t, client)
		assert.NotNil(t, client.GetConfig())
	})

	t.Run("with_options_and_config", func(t *testing.T) {
		cfg := &LLMConfig{
			Backend: ModeAPI,
			APIKey:  "sk-test",
		}
		client := NewClient(WithConfig(cfg), WithVerbose(true))
		assert.NotNil(t, client)
		assert.Equal(t, ModeAPI, client.config.Backend)
	})

	t.Run("with_mode_option", func(t *testing.T) {
		client := NewClient(WithMode(ModeAPI))
		assert.NotNil(t, client)
	})
}

func TestClient_GetActiveEngine(t *testing.T) {
	t.Run("with API key", func(t *testing.T) {
		cfg := &LLMConfig{
			APIKey: "sk-test",
		}
		// Use WithMode(ModeAuto) explicitly to allow all providers
		client := NewClient(WithConfig(cfg), WithMode(ModeAuto))
		eng := client.GetActiveEngine()
		assert.NotNil(t, eng)
		assert.Equal(t, "mock", eng.Name())
	})

	t.Run("no engine available", func(t *testing.T) {
		cfg := &LLMConfig{
			// No API key - mock engine will not be available
		}
		client := NewClient(WithConfig(cfg), WithMode(ModeAuto))
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
			WithComplexity(ComplexityHigh)
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
			WithComplexity(ComplexityMedium).
			WithMaxTokens(1500).
			WithTemperature(0.8)
		assert.NotNil(t, builder)
	})
}

func TestModeConstants(t *testing.T) {
	// Verify mode constants exist
	assert.Equal(t, Mode("api"), ModeAPI)
	assert.Equal(t, Mode("mcp"), ModeMCP)
	assert.Equal(t, Mode("cli"), ModeCLI)
	assert.Equal(t, Mode("auto"), ModeAuto)
}
