package checkstyle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs Checkstyle with the given config and files.
func (l *Linter) execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	if len(files) == 0 {
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
	jarPath := l.getJARPath()

	args := []string{
		"-jar", jarPath,
		"-c", configFile,
		"-f", "xml", // XML output format
	}
	args = append(args, files...)

	// Execute (uses CWD by default)
	start := time.Now()

	output, err := l.executor.Execute(ctx, l.JavaPath, args...)
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

	// Checkstyle returns non-zero exit code when violations are found
	// This is expected, not an error
	if err != nil && output.ExitCode != 0 {
		// Only return error if it's not a violations-found error
		if output.Stdout == "" && output.Stderr != "" {
			return output, fmt.Errorf("checkstyle execution failed: %w", err)
		}
	}

	return output, nil
}

// createTempConfig creates a temporary config file.
func (l *Linter) createTempConfig(config []byte) (string, error) {
	// Create temp file in tools directory
	tempFile := filepath.Join(l.ToolsDir, "checkstyle-config-temp.xml")

	if err := os.WriteFile(tempFile, config, 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}
