package pylint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs Pylint with the given config and files.
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
	pylintCmd := l.getPylintCommand()
	args := l.getExecutionArgs(configPath, files)

	// Execute - uses CWD by default
	return l.executor.Execute(ctx, pylintCmd, args...)
}

// getExecutionArgs returns the arguments for Pylint execution.
func (l *Linter) getExecutionArgs(configPath string, files []string) []string {
	args := []string{
		"--output-format=json",
		"--rcfile=" + configPath, // Use .pylintrc settings as-is
	}
	args = append(args, files...)

	return args
}

// writeConfigFile writes Pylint config to a temp file.
func (l *Linter) writeConfigFile(config []byte) (string, error) {
	tmpDir := filepath.Join(l.ToolsDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(tmpDir, "pylintrc-*")
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
