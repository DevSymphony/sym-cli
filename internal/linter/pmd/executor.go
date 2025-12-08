package pmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs PMD with the given config and files.
func (l *Linter) execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	if len(files) == 0 {
		return &linter.ToolOutput{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 0,
			Duration: "0s",
		}, nil
	}

	// Create temp ruleset file
	rulesetFile, err := l.createTempRuleset(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp ruleset: %w", err)
	}
	defer func() { _ = os.Remove(rulesetFile) }()

	// Build command
	pmdPath := l.getPMDPath()

	// PMD command format: pmd check -d <files> -R <ruleset> -f json
	args := []string{
		"check",
		"-d", strings.Join(files, ","), // Comma-separated file list
		"-R", rulesetFile,
		"-f", "json", // JSON output format
		"--no-cache", // Disable cache for consistent results
	}

	// Execute (uses CWD by default)
	start := time.Now()

	output, err := l.executor.Execute(ctx, pmdPath, args...)
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

	// PMD returns exit code 4 when violations are found
	// This is expected, not an error
	if err != nil && (output.ExitCode == 4 || output.ExitCode == 0) {
		// Not an actual error, just violations found
		err = nil
	}

	// Only return error if it's a real execution error
	if err != nil && output.Stdout == "" && output.Stderr != "" {
		return output, fmt.Errorf("PMD execution failed: %w", err)
	}

	return output, nil
}

// createTempRuleset creates a temporary ruleset file.
func (l *Linter) createTempRuleset(config []byte) (string, error) {
	// Create temp file in tools directory
	tempFile := filepath.Join(l.ToolsDir, "pmd-ruleset-temp.xml")

	if err := os.WriteFile(tempFile, config, 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}
