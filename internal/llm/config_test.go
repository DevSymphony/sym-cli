package llm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLLMConfig(t *testing.T) {
	cfg := DefaultLLMConfig()

	assert.Equal(t, ModeAuto, cfg.Backend)
	assert.Empty(t, cfg.CLI)
	assert.Empty(t, cfg.CLIPath)
	assert.Empty(t, cfg.Model)
}

func TestLLMConfig_HasCLI(t *testing.T) {
	t.Run("with CLI", func(t *testing.T) {
		cfg := &LLMConfig{CLI: "claude"}
		assert.True(t, cfg.HasCLI())
	})

	t.Run("without CLI", func(t *testing.T) {
		cfg := &LLMConfig{}
		assert.False(t, cfg.HasCLI())
	})
}

func TestLLMConfig_HasAPIKey(t *testing.T) {
	t.Run("with API key in config", func(t *testing.T) {
		cfg := &LLMConfig{APIKey: "sk-test"}
		assert.True(t, cfg.HasAPIKey())
	})

	t.Run("without API key", func(t *testing.T) {
		cfg := &LLMConfig{}
		assert.False(t, cfg.HasAPIKey())
	})
}

func TestLLMConfig_GetEffectiveBackend(t *testing.T) {
	t.Run("explicit mode", func(t *testing.T) {
		cfg := &LLMConfig{Backend: ModeCLI}
		assert.Equal(t, ModeCLI, cfg.GetEffectiveBackend())
	})

	t.Run("auto with CLI", func(t *testing.T) {
		cfg := &LLMConfig{Backend: ModeAuto, CLI: "claude"}
		assert.Equal(t, ModeCLI, cfg.GetEffectiveBackend())
	})

	t.Run("auto with API key", func(t *testing.T) {
		cfg := &LLMConfig{Backend: ModeAuto, APIKey: "sk-test"}
		assert.Equal(t, ModeAPI, cfg.GetEffectiveBackend())
	})

	t.Run("auto with nothing", func(t *testing.T) {
		cfg := &LLMConfig{Backend: ModeAuto}
		assert.Equal(t, ModeAuto, cfg.GetEffectiveBackend())
	})
}

func TestLLMConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &LLMConfig{
			Backend: ModeAuto,
			CLI:     "claude",
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid backend", func(t *testing.T) {
		cfg := &LLMConfig{Backend: Mode("invalid")}
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid CLI provider", func(t *testing.T) {
		cfg := &LLMConfig{CLI: "invalid-cli"}
		assert.Error(t, cfg.Validate())
	})

	t.Run("empty config is valid", func(t *testing.T) {
		cfg := &LLMConfig{}
		assert.NoError(t, cfg.Validate())
	})
}

func TestLLMConfig_String(t *testing.T) {
	cfg := &LLMConfig{
		Backend: ModeAuto,
		CLI:     "claude",
		Model:   "claude-3-opus",
	}

	str := cfg.String()
	assert.Contains(t, str, "Backend: auto")
	assert.Contains(t, str, "CLI: claude")
	assert.Contains(t, str, "Model: claude-3-opus")
}

func TestSaveLLMConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &LLMConfig{
		Backend:    ModeCLI,
		CLI:        "claude",
		Model:      "claude-3-opus",
		LargeModel: "claude-3-opus",
	}

	err := SaveLLMConfigToDir(tmpDir, cfg)
	require.NoError(t, err)

	// Verify file was created
	envPath := filepath.Join(tmpDir, ".env")
	_, err = os.Stat(envPath)
	require.NoError(t, err)

	// Read and verify content
	content, err := os.ReadFile(envPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "LLM_BACKEND=cli")
	assert.Contains(t, string(content), "LLM_CLI=claude")
	assert.Contains(t, string(content), "LLM_MODEL=claude-3-opus")
}

func TestLoadLLMConfigFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env file
	envContent := `# Test config
LLM_BACKEND=cli
LLM_CLI=gemini
LLM_MODEL=gemini-pro
`
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	cfg := LoadLLMConfigFromDir(tmpDir)

	assert.Equal(t, ModeCLI, cfg.Backend)
	assert.Equal(t, "gemini", cfg.CLI)
	assert.Equal(t, "gemini-pro", cfg.Model)
}

func TestLoadLLMConfigFromDir_NonExistent(t *testing.T) {
	cfg := LoadLLMConfigFromDir("/nonexistent/path")

	// Should return defaults
	assert.Equal(t, ModeAuto, cfg.Backend)
	assert.Empty(t, cfg.CLI)
}

