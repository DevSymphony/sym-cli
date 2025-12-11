package golangcilint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs golangci-lint with the given config and files.
func (l *Linter) execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	if len(files) == 0 {
		return &linter.ToolOutput{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 0,
			Duration: "0s",
		}, nil
	}

	// Filter to only .go files
	goFiles := filterGoFiles(files)
	if len(goFiles) == 0 {
		return &linter.ToolOutput{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 0,
			Duration: "0s",
		}, nil
	}

	// Create temp config file
	configFile, err := l.createTempConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp config: %w", err)
	}
	defer func() { _ = os.Remove(configFile) }()

	// Build command
	golangciLintPath := l.getGolangciLintPath()

	// golangci-lint command format:
	// golangci-lint run --config <config> --out-format json --path-prefix="" <files>
	args := []string{
		"run",
		"--config", configFile,
		"--out-format", "json",
		"--path-prefix", "", // Disable path prefix to get absolute paths
	}

	// Add files
	args = append(args, goFiles...)

	// Execute
	start := time.Now()

	output, err := l.executor.Execute(ctx, golangciLintPath, args...)
	duration := time.Since(start)

	if output == nil {
		output = &linter.ToolOutput{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 1,
			Duration: duration.String(),
		}
		if err != nil {
			output.Stderr = err.Error()
		}
	} else {
		output.Duration = duration.String()
	}

	// golangci-lint returns exit code 1 when violations are found
	// This is expected, not an error
	if err != nil && (output.ExitCode == 1 || output.ExitCode == 0) {
		// Not an actual error, just violations found
		err = nil
	}

	// Only return error if it's a real execution error (exit code 2 = config error)
	if err != nil && output.ExitCode == 2 {
		return output, fmt.Errorf("golangci-lint configuration error: %s", output.Stderr)
	}

	// Other execution errors
	if err != nil && output.Stdout == "" && output.Stderr != "" {
		return output, fmt.Errorf("golangci-lint execution failed: %w", err)
	}

	return output, nil
}

// createTempConfig creates a temporary config file.
func (l *Linter) createTempConfig(config []byte) (string, error) {
	// Ensure temp directory exists
	tempDir := filepath.Join(l.ToolsDir, ".tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create temp file
	tempFile := filepath.Join(tempDir, "golangci-lint-config-temp.yml")

	if err := os.WriteFile(tempFile, config, 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}

// filterGoFiles filters the file list to only include .go files.
func filterGoFiles(files []string) []string {
	goFiles := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.Ext(file) == ".go" {
			goFiles = append(goFiles, file)
		}
	}
	return goFiles
}
