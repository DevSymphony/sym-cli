package llm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm/engine"
)

const (
	// Default .sym/.env file location relative to repo root
	defaultEnvFile = ".sym/.env"

	// Environment variable keys
	envKeyLLMBackend = "LLM_BACKEND"
	envKeyLLMCLI     = "LLM_CLI"
	envKeyLLMCLIPath = "LLM_CLI_PATH"
	envKeyLLMModel   = "LLM_MODEL"
	envKeyLLMLarge   = "LLM_LARGE_MODEL"
	envKeyAPIKey     = "OPENAI_API_KEY"
)

// LLMConfig holds LLM engine configuration.
type LLMConfig struct {
	// Backend is the preferred engine mode (auto, mcp, cli, api).
	Backend engine.Mode `json:"backend"`

	// CLI is the CLI provider type (claude, gemini).
	CLI string `json:"cli"`

	// CLIPath is a custom path to the CLI executable (optional).
	CLIPath string `json:"cli_path"`

	// Model is the default model name for CLI engine.
	Model string `json:"model"`

	// LargeModel is the model for high complexity tasks (optional).
	LargeModel string `json:"large_model"`

	// APIKey is loaded from environment (not saved to config).
	APIKey string `json:"-"`
}

// DefaultLLMConfig returns the default configuration.
func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		Backend: engine.ModeAuto,
		CLI:     "",
		CLIPath: "",
		Model:   "",
	}
}

// LoadLLMConfig loads LLM configuration from .sym/.env file and environment.
func LoadLLMConfig() *LLMConfig {
	cfg := DefaultLLMConfig()

	// Load from .sym/.env file first
	envPath := defaultEnvFile
	loadConfigFromEnvFile(envPath, cfg)

	// Override with system environment variables
	loadConfigFromEnv(cfg)

	return cfg
}

// LoadLLMConfigFromDir loads LLM configuration from a specific directory.
func LoadLLMConfigFromDir(dir string) *LLMConfig {
	cfg := DefaultLLMConfig()

	// Load from .env file in the specified directory
	envPath := filepath.Join(dir, ".env")
	loadConfigFromEnvFile(envPath, cfg)

	// Override with system environment variables
	loadConfigFromEnv(cfg)

	return cfg
}

// loadConfigFromEnvFile reads config values from .env file.
func loadConfigFromEnvFile(envPath string, cfg *LLMConfig) {
	file, err := os.Open(envPath)
	if err != nil {
		return // File doesn't exist, use defaults
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case envKeyLLMBackend:
			if engine.Mode(value).IsValid() {
				cfg.Backend = engine.Mode(value)
			}
		case envKeyLLMCLI:
			cfg.CLI = value
		case envKeyLLMCLIPath:
			cfg.CLIPath = value
		case envKeyLLMModel:
			cfg.Model = value
		case envKeyLLMLarge:
			cfg.LargeModel = value
		case envKeyAPIKey:
			cfg.APIKey = value
		}
	}
}

// loadConfigFromEnv loads config from system environment variables.
func loadConfigFromEnv(cfg *LLMConfig) {
	if backend := os.Getenv(envKeyLLMBackend); backend != "" {
		if engine.Mode(backend).IsValid() {
			cfg.Backend = engine.Mode(backend)
		}
	}

	if cli := os.Getenv(envKeyLLMCLI); cli != "" {
		cfg.CLI = cli
	}

	if cliPath := os.Getenv(envKeyLLMCLIPath); cliPath != "" {
		cfg.CLIPath = cliPath
	}

	if model := os.Getenv(envKeyLLMModel); model != "" {
		cfg.Model = model
	}

	if large := os.Getenv(envKeyLLMLarge); large != "" {
		cfg.LargeModel = large
	}

	if apiKey := os.Getenv(envKeyAPIKey); apiKey != "" {
		cfg.APIKey = apiKey
	}
}

// SaveLLMConfig saves LLM configuration to .sym/.env file.
func SaveLLMConfig(cfg *LLMConfig) error {
	return SaveLLMConfigToDir(".sym", cfg)
}

// SaveLLMConfigToDir saves LLM configuration to a specific directory.
func SaveLLMConfigToDir(dir string, cfg *LLMConfig) error {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	envPath := filepath.Join(dir, ".env")

	// Read existing content
	existingLines, existingKeys := readExistingEnvFile(envPath)

	// Prepare new values
	newValues := map[string]string{}

	if cfg.Backend != "" && cfg.Backend != engine.ModeAuto {
		newValues[envKeyLLMBackend] = string(cfg.Backend)
	}

	if cfg.CLI != "" {
		newValues[envKeyLLMCLI] = cfg.CLI
	}

	if cfg.CLIPath != "" {
		newValues[envKeyLLMCLIPath] = cfg.CLIPath
	}

	if cfg.Model != "" {
		newValues[envKeyLLMModel] = cfg.Model
	}

	if cfg.LargeModel != "" {
		newValues[envKeyLLMLarge] = cfg.LargeModel
	}

	// Build output lines
	var outputLines []string

	// Update existing lines
	for _, line := range existingLines {
		trimmed := strings.TrimSpace(line)

		// Keep comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			outputLines = append(outputLines, line)
			continue
		}

		// Parse key
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			outputLines = append(outputLines, line)
			continue
		}

		key := strings.TrimSpace(parts[0])

		// Check if we have a new value for this key
		if newValue, ok := newValues[key]; ok {
			outputLines = append(outputLines, fmt.Sprintf("%s=%s", key, newValue))
			delete(newValues, key) // Mark as processed
		} else {
			outputLines = append(outputLines, line)
		}
	}

	// Add LLM config section header if needed
	hasLLMSection := false
	for key := range existingKeys {
		if strings.HasPrefix(key, "LLM_") {
			hasLLMSection = true
			break
		}
	}

	// Add new keys that weren't in the file
	if len(newValues) > 0 {
		if !hasLLMSection {
			outputLines = append(outputLines, "", "# LLM Backend Configuration")
		}

		for key, value := range newValues {
			outputLines = append(outputLines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Write to file
	content := strings.Join(outputLines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(envPath, []byte(content), 0600)
}

// readExistingEnvFile reads existing .env file content.
func readExistingEnvFile(envPath string) ([]string, map[string]bool) {
	var lines []string
	keys := make(map[string]bool)

	file, err := os.Open(envPath)
	if err != nil {
		return lines, keys
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Track existing keys
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				keys[strings.TrimSpace(parts[0])] = true
			}
		}
	}

	return lines, keys
}

// GetAPIKey returns the API key from config or environment.
func (c *LLMConfig) GetAPIKey() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return os.Getenv(envKeyAPIKey)
}

// HasCLI returns true if CLI is configured.
func (c *LLMConfig) HasCLI() bool {
	return c.CLI != ""
}

// HasAPIKey returns true if API key is available.
func (c *LLMConfig) HasAPIKey() bool {
	return c.GetAPIKey() != ""
}

// GetEffectiveBackend returns the actual engine to use based on availability.
func (c *LLMConfig) GetEffectiveBackend() engine.Mode {
	if c.Backend != engine.ModeAuto {
		return c.Backend
	}

	// Auto mode: prefer CLI if available, then API
	if c.HasCLI() {
		return engine.ModeCLI
	}

	if c.HasAPIKey() {
		return engine.ModeAPI
	}

	return engine.ModeAuto
}

// Validate checks if the configuration is valid.
func (c *LLMConfig) Validate() error {
	if c.Backend != "" && !c.Backend.IsValid() {
		return fmt.Errorf("invalid engine mode: %s", c.Backend)
	}

	if c.CLI != "" && !engine.CLIProviderType(c.CLI).IsValid() {
		return fmt.Errorf("unsupported CLI provider: %s", c.CLI)
	}

	return nil
}

// String returns a human-readable representation of the config.
func (c *LLMConfig) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Backend: %s", c.Backend))

	if c.CLI != "" {
		parts = append(parts, fmt.Sprintf("CLI: %s", c.CLI))
	}

	if c.CLIPath != "" {
		parts = append(parts, fmt.Sprintf("CLI Path: %s", c.CLIPath))
	}

	if c.Model != "" {
		parts = append(parts, fmt.Sprintf("Model: %s", c.Model))
	}

	if c.LargeModel != "" {
		parts = append(parts, fmt.Sprintf("Large Model: %s", c.LargeModel))
	}

	if c.HasAPIKey() {
		parts = append(parts, "API Key: configured")
	} else {
		parts = append(parts, "API Key: not set")
	}

	return strings.Join(parts, ", ")
}
