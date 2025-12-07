package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRawProvider is a test provider for unit tests.
type mockRawProvider struct {
	name     string
	response string
	err      error
}

func (m *mockRawProvider) ExecuteRaw(ctx context.Context, prompt string, format ResponseFormat) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockRawProvider) Name() string {
	return m.name
}

func (m *mockRawProvider) Close() error {
	return nil
}

func init() {
	// Register a test provider
	RegisterProvider("test-provider", func(cfg Config) (RawProvider, error) {
		return &mockRawProvider{
			name:     "test-provider",
			response: "test response",
		}, nil
	}, ProviderInfo{
		Name:         "test-provider",
		DisplayName:  "Test Provider",
		DefaultModel: "test-model",
		Available:    true,
	})
}

func TestNew(t *testing.T) {
	t.Run("creates provider from config", func(t *testing.T) {
		provider, err := New(Config{Provider: "test-provider"})
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-provider", provider.Name())
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		_, err := New(Config{Provider: "unknown-provider"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}

func TestProvider_Execute(t *testing.T) {
	provider, err := New(Config{Provider: "test-provider"})
	require.NoError(t, err)

	t.Run("executes with text format", func(t *testing.T) {
		result, err := provider.Execute(context.Background(), "test prompt", Text)
		require.NoError(t, err)
		assert.Equal(t, "test response", result)
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("returns error when provider is empty", func(t *testing.T) {
		cfg := Config{}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
	})

	t.Run("valid config with CLI provider", func(t *testing.T) {
		cfg := Config{Provider: "claudecode"}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid config with openaiapi", func(t *testing.T) {
		// Note: API key validation is now handled by the provider itself
		cfg := Config{Provider: "openaiapi"}
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("returns config from LoadConfigFromDir", func(t *testing.T) {
		// LoadConfig is a simple wrapper around LoadConfigFromDir
		cfg := LoadConfig()
		// Without .sym/config.json, it should return empty config
		// We just verify it doesn't panic
		_ = cfg.Provider
		_ = cfg.Model
	})
}

func TestLoadConfigFromDir(t *testing.T) {
	t.Run("returns empty config when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := LoadConfigFromDir(tmpDir)
		assert.Empty(t, cfg.Provider)
		assert.Empty(t, cfg.Model)
	})
}

func TestGetProviderInfo(t *testing.T) {
	t.Run("returns info for registered provider", func(t *testing.T) {
		info := GetProviderInfo("test-provider")
		require.NotNil(t, info)
		assert.Equal(t, "test-provider", info.Name)
		assert.Equal(t, "Test Provider", info.DisplayName)
	})

	t.Run("returns nil for unknown provider", func(t *testing.T) {
		info := GetProviderInfo("unknown")
		assert.Nil(t, info)
	})
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	assert.NotEmpty(t, providers)

	// Should include our test provider
	var found bool
	for _, p := range providers {
		if p.Name == "test-provider" {
			found = true
			break
		}
	}
	assert.True(t, found, "test-provider should be in list")
}

func Test_parse(t *testing.T) {
	t.Run("parses JSON from response", func(t *testing.T) {
		response := `Here is the result: {"key": "value"}`
		result, err := parse(response, JSON)
		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"}`, result)
	})

	t.Run("parses XML from response", func(t *testing.T) {
		response := `Here is XML: <root>value</root>`
		result, err := parse(response, XML)
		require.NoError(t, err)
		assert.Equal(t, `<root>value</root>`, result)
	})

	t.Run("returns text as-is", func(t *testing.T) {
		response := "Just plain text"
		result, err := parse(response, Text)
		require.NoError(t, err)
		assert.Equal(t, response, result)
	})
}
