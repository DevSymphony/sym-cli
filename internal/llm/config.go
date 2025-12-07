package llm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/config"
)

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required (configure in .sym/config.json or set LLM_PROVIDER)")
	}
	if c.Provider == "openaiapi" && c.APIKey == "" {
		return fmt.Errorf("API key is required for openaiapi (set in .sym/.env or OPENAI_API_KEY)")
	}
	return nil
}

// LoadConfig loads configuration from config.json, .env file, and environment variables.
// Priority: environment variables > .sym/.env > .sym/config.json
func LoadConfig() Config {
	return LoadConfigFromDir("")
}

// LoadConfigFromDir loads configuration from a directory's config files and environment variables.
// Environment variables take precedence over file values.
func LoadConfigFromDir(dir string) Config {
	cfg := Config{}

	// 1. Load from .sym/config.json (lowest priority for provider/model)
	if projectCfg, err := config.LoadProjectConfig(); err == nil {
		cfg.Provider = projectCfg.LLM.Provider
		cfg.Model = projectCfg.LLM.Model
	}

	// 2. Load API key from .env file (for sensitive data only)
	envPath := filepath.Join(".sym", ".env")
	if dir != "" {
		envPath = filepath.Join(dir, ".env")
	}
	loadEnvFileAPIKey(envPath, &cfg)

	// 3. Environment variables override all file values
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.APIKey = v
	}

	return cfg
}

// loadEnvFileAPIKey reads .env file and loads only API key (sensitive data).
func loadEnvFileAPIKey(path string, cfg *Config) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only load API key from .env (sensitive data)
		if key == "OPENAI_API_KEY" && os.Getenv(key) == "" {
			cfg.APIKey = value
		}
	}
}

