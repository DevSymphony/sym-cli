package eslint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	// Get command and arguments
	eslintCmd, args := a.getExecutionArgs(configPath, files)

	// Execute with environment variable to support both ESLint 8 and 9
	a.executor.WorkDir = a.WorkDir
	a.executor.Env = map[string]string{
		"ESLINT_USE_FLAT_CONFIG": "false",
	}
	return a.executor.Execute(ctx, eslintCmd, args...)
}

// getESLintCommand returns the ESLint command to use.
func (a *Adapter) getESLintCommand() string {
	// Try local installation first
	localPath := a.getESLintPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Try global eslint
	if _, err := exec.LookPath("eslint"); err == nil {
		return "eslint"
	}

	// Fall back to npx with ESLint 8.x
	return "npx"
}

// getExecutionArgs returns the command and arguments for ESLint execution.
func (a *Adapter) getExecutionArgs(configPath string, files []string) (string, []string) {
	eslintCmd := a.getESLintCommand()

	var args []string

	// If using npx, specify eslint@8
	if eslintCmd == "npx" {
		args = []string{"eslint@8"}
	}

	// Add ESLint arguments
	args = append(args,
		"--config", configPath,
		"--format", "json",
		"--no-eslintrc", // Don't load user's .eslintrc
	)
	args = append(args, files...)

	return eslintCmd, args
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
