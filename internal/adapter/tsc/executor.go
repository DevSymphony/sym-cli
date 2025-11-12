package tsc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// execute runs tsc with the given configuration.
func (a *Adapter) execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	// Write tsconfig.json to a temporary location
	configPath := filepath.Join(a.WorkDir, ".symphony-tsconfig.json")
	if err := os.WriteFile(configPath, config, 0644); err != nil {
		return nil, fmt.Errorf("failed to write tsconfig: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	// Determine tsc binary path
	tscPath := a.getTSCPath()
	if _, err := os.Stat(tscPath); os.IsNotExist(err) {
		// Try global tsc
		tscPath = "tsc"
	}

	// Build tsc command
	// Use --noEmit to only check types without generating output
	// Use --pretty false to get machine-readable output
	args := []string{
		"--project", configPath,
		"--noEmit",
		"--pretty", "false",
	}

	// If specific files are provided, add them
	if len(files) > 0 {
		args = append(args, files...)
	}

	// Execute tsc
	a.executor.WorkDir = a.WorkDir
	output, err := a.executor.Execute(ctx, tscPath, args...)

	// TSC returns non-zero exit code when there are type errors
	// This is expected, so we don't treat it as an error
	if output != nil {
		return output, nil
	}

	return nil, err
}
