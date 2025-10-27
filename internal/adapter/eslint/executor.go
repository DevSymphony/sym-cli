package eslint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// execute runs ESLint with the given config and files.
func (a *Adapter) execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	if len(files) == 0 {
		return &adapter.ToolOutput{
			Stdout:   "[]",
			ExitCode: 0,
		}, nil
	}

	// Write config to temp file
	configPath, err := a.writeConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	defer os.Remove(configPath)

	// Determine ESLint command
	eslintCmd := a.getESLintCommand()

	// Build arguments
	args := []string{
		"--config", configPath,
		"--format", "json",
		"--no-eslintrc", // Don't load user's .eslintrc
	}
	args = append(args, files...)

	// Execute
	a.executor.WorkDir = a.WorkDir
	return a.executor.Execute(ctx, eslintCmd, args...)
}

// getESLintCommand returns the ESLint command to use.
func (a *Adapter) getESLintCommand() string {
	// Try local installation first
	localPath := a.getESLintPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Fall back to global
	return "eslint"
}

// writeConfigFile writes ESLint config to a temp file.
func (a *Adapter) writeConfigFile(config []byte) (string, error) {
	tmpDir := filepath.Join(a.ToolsDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(tmpDir, "eslintrc-*.json")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(config); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
