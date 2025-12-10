package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config represents the Symphony CLI configuration
type Config struct {
	PolicyPath string `json:"policy_path,omitempty"` // Custom path for user-policy.json
}

var (
	configDir  string
	configPath string
)

func init() {
	homeDir := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	}
	configDir = filepath.Join(homeDir, ".config", "sym")
	configPath = filepath.Join(configDir, "config.json")
}

// ensureConfigDir creates the config directory if it doesn't exist
func ensureConfigDir() error {
	return os.MkdirAll(configDir, 0700)
}

// LoadConfig loads the configuration from file
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration not found. Run 'sym init' to set up")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(cfg *Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}
