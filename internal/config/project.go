package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectConfig represents the .sym/config.json structure
type ProjectConfig struct {
	LLM        LLMConfig   `json:"llm,omitempty"`
	MCP        MCPConfig   `json:"mcp,omitempty"`
	PolicyPath string      `json:"policy_path,omitempty"`
}

// LLMConfig holds LLM provider settings
type LLMConfig struct {
	Provider string `json:"provider,omitempty"` // "claudecode", "geminicli", "openaiapi"
	Model    string `json:"model,omitempty"`    // Model name
}

// MCPConfig holds MCP tool registration settings
type MCPConfig struct {
	Tools []string `json:"tools,omitempty"` // ["vscode", "claude-code", "cursor"]
}

const (
	symDir            = ".sym"
	projectConfigFile = "config.json"
	projectEnvFile    = ".env"
)

// GetProjectConfigPath returns the path to .sym/config.json
func GetProjectConfigPath() string {
	return filepath.Join(symDir, projectConfigFile)
}

// GetProjectEnvPath returns the path to .sym/.env
func GetProjectEnvPath() string {
	return filepath.Join(symDir, projectEnvFile)
}

// LoadProjectConfig loads the project configuration from .sym/config.json
func LoadProjectConfig() (*ProjectConfig, error) {
	configPath := GetProjectConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &ProjectConfig{
				PolicyPath: ".sym/user-policy.json",
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &cfg, nil
}

// SaveProjectConfig saves the project configuration to .sym/config.json
func SaveProjectConfig(cfg *ProjectConfig) error {
	// Ensure .sym directory exists
	if err := os.MkdirAll(symDir, 0755); err != nil {
		return fmt.Errorf("failed to create .sym directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := GetProjectConfigPath()
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// UpdateProjectConfigLLM updates only the LLM section of project config
func UpdateProjectConfigLLM(provider, model string) error {
	cfg, err := LoadProjectConfig()
	if err != nil {
		cfg = &ProjectConfig{}
	}

	cfg.LLM.Provider = provider
	cfg.LLM.Model = model

	return SaveProjectConfig(cfg)
}

// UpdateProjectConfigMCP updates only the MCP section of project config
func UpdateProjectConfigMCP(tools []string) error {
	cfg, err := LoadProjectConfig()
	if err != nil {
		cfg = &ProjectConfig{}
	}

	cfg.MCP.Tools = tools

	return SaveProjectConfig(cfg)
}

// ProjectConfigExists checks if .sym/config.json exists
func ProjectConfigExists() bool {
	_, err := os.Stat(GetProjectConfigPath())
	return err == nil
}
