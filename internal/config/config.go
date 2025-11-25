package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	// Authentication mode: "server" (default) or "custom"
	AuthMode string `json:"auth_mode,omitempty"`

	// Server authentication (default)
	ServerURL string `json:"server_url,omitempty"` // Symphony auth server URL

	// Custom OAuth (Enterprise용, 선택사항)
	GitHubHost   string `json:"github_host,omitempty"`   // "github.com" or custom GHES host
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`

	PolicyPath string `json:"policy_path,omitempty"` // symphonyclient integration: Custom path for user-policy.json (default: .sym/user-policy.json)
}

type Token struct {
	AccessToken string `json:"access_token"`
}

var (
	configDir  string
	configPath string
	tokenPath  string
)

func init() {
	homeDir := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	}
	// symphonyclient integration: symphony → sym directory
	configDir = filepath.Join(homeDir, ".config", "sym")
	configPath = filepath.Join(configDir, "config.json")
	tokenPath = filepath.Join(configDir, "token.json")
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
			// symphonyclient integration: symphony → sym command
			return nil, fmt.Errorf("configuration not found. Run 'sym config' to set up")
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

// LoadToken loads the access token from file
func LoadToken() (*Token, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			// symphonyclient integration: symphony → sym command
			return nil, fmt.Errorf("not logged in. Run 'sym login' first")
		}
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("invalid token file: %w", err)
	}

	return &token, nil
}

// SaveToken saves the access token to file
func SaveToken(token *Token) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

// DeleteToken removes the token file (logout)
func DeleteToken() error {
	err := os.Remove(tokenPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsLoggedIn checks if a valid token exists
func IsLoggedIn() bool {
	_, err := LoadToken()
	return err == nil
}

// GetConfigPath returns the config file path
func GetConfigPath() string {
	return configPath
}

// GetTokenPath returns the token file path
func GetTokenPath() string {
	return tokenPath
}

// GetAuthMode returns the authentication mode (defaults to "server")
func (c *Config) GetAuthMode() string {
	if c.AuthMode == "" {
		// symphonyclient integration: SYMPHONY → SYM environment variable
		if mode := os.Getenv("SYM_AUTH_MODE"); mode != "" {
			return mode
		}
		return "server" // default
	}
	return c.AuthMode
}

// GetServerURL returns the auth server URL (with defaults)
func (c *Config) GetServerURL() string {
	if c.ServerURL == "" {
		// symphonyclient integration: SYMPHONY → SYM environment variable
		if url := os.Getenv("SYM_SERVER_URL"); url != "" {
			return url
		}
		// Default server URL (symphonyclient auth server - kept for compatibility)
		return "https://symphony-server-98207.web.app"
	}
	return c.ServerURL
}

// IsCustomOAuth returns true if using custom OAuth mode
func (c *Config) IsCustomOAuth() bool {
	return c.GetAuthMode() == "custom"
}

// GetGitHubHost returns the GitHub host (defaults to github.com for server mode)
func (c *Config) GetGitHubHost() string {
	if c.GitHubHost == "" {
		return "github.com" // default
	}
	return c.GitHubHost
}
