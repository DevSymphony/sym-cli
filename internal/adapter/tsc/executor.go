package tsc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// execute runs tsc with the given configuration.
func (a *Adapter) execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	// Parse config and add files to check
	var tsconfig map[string]interface{}
	if err := json.Unmarshal(config, &tsconfig); err != nil {
		return nil, fmt.Errorf("failed to parse tsconfig: %w", err)
	}

	// Add files to tsconfig if specific files are provided
	if len(files) > 0 {
		tsconfig["files"] = files
	}

	// Marshal updated config
	updatedConfig, err := json.MarshalIndent(tsconfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated tsconfig: %w", err)
	}

	// Write tsconfig.json to a temporary location
	configPath := filepath.Join(a.WorkDir, ".symphony-tsconfig.json")
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
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
	// Use --project to read config, --noEmit to only check types, --pretty false for machine-readable output
	args := []string{
		"--project", configPath,
		"--noEmit",
		"--pretty", "false",
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
