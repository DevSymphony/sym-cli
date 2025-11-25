package checkstyle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// execute runs Checkstyle with the given config and files.
func (a *Adapter) execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	if len(files) == 0 {
		return &adapter.ToolOutput{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 0,
			Duration: "0s",
		}, nil
	}

	// Create temp config file
	configFile, err := a.createTempConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp config: %w", err)
	}
	defer func() { _ = os.Remove(configFile) }()

	// Build command
	jarPath := a.getJARPath()

	args := []string{
		"-jar", jarPath,
		"-c", configFile,
		"-f", "xml", // XML output format
	}
	args = append(args, files...)

	// Execute (uses CWD by default)
	start := time.Now()

	output, err := a.executor.Execute(ctx, a.JavaPath, args...)
	duration := time.Since(start)

	if output == nil {
		output = &adapter.ToolOutput{
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
func (a *Adapter) createTempConfig(config []byte) (string, error) {
	// Create temp file in tools directory
	tempFile := filepath.Join(a.ToolsDir, "checkstyle-config-temp.xml")

	if err := os.WriteFile(tempFile, config, 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}
