package llm

import (
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/util/config"
)

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required (configure in .sym/config.json)")
	}
	return nil
}

// LoadConfig loads configuration from .sym/config.json.
func LoadConfig() Config {
	return LoadConfigFromDir("")
}

// LoadConfigFromDir loads configuration from .sym/config.json.
// Note: API keys are handled by individual providers (e.g., openaiapi uses envutil.GetAPIKey).
func LoadConfigFromDir(_ string) Config {
	cfg := Config{}

	// Load from .sym/config.json
	if projectCfg, err := config.LoadProjectConfig(); err == nil {
		cfg.Provider = projectCfg.LLM.Provider
		cfg.Model = projectCfg.LLM.Model
	}

	return cfg
}
