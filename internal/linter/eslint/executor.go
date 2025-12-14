package eslint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs ESLint with the given config and files.
func (l *Linter) execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	if len(files) == 0 {
		return &linter.ToolOutput{
			Stdout:   "[]",
			ExitCode: 0,
		}, nil
	}

	// Write config to temp file
	configPath, err := l.writeConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	// Get command and arguments
	eslintCmd, args := l.getExecutionArgs(configPath, files)

	// Execute with environment variable to support both ESLint 8 and 9
	// Reset WorkDir to use CWD (Install() may have set it to ToolsDir)
	l.executor.WorkDir = ""
	l.executor.Env = map[string]string{
		"ESLINT_USE_FLAT_CONFIG": "false",
	}
	return l.executor.Execute(ctx, eslintCmd, args...)
}

// getESLintCommand returns the ESLint command to use.
func (l *Linter) getESLintCommand() string {
	// Try local installation first
	localPath := l.getESLintPath()
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
func (l *Linter) getExecutionArgs(configPath string, files []string) (string, []string) {
	eslintCmd := l.getESLintCommand()

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
func (l *Linter) writeConfigFile(config []byte) (string, error) {
	tmpDir := filepath.Join(l.ToolsDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(tmpDir, "eslintrc-*.json")
	if err != nil {
		return "", err
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.Write(config); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
