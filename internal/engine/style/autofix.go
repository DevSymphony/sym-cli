package style

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Autofix applies style fixes to files using Prettier.
func (e *Engine) Autofix(ctx context.Context, rule core.Rule, files []string) ([]string, error) {
	if e.prettier == nil {
		return nil, fmt.Errorf("prettier not available for autofix")
	}

	files = e.filterFiles(files, rule.When)
	if len(files) == 0 {
		return nil, nil
	}

	// Generate Prettier config
	config, err := e.prettier.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Prettier config: %w", err)
	}

	// Execute Prettier --write
	_, err = e.prettier.ExecuteWithMode(ctx, config, files, "write")
	if err != nil {
		return nil, fmt.Errorf("prettier --write failed: %w", err)
	}

	return files, nil
}

// GenerateDiff generates a diff preview without modifying files.
func (e *Engine) GenerateDiff(ctx context.Context, rule core.Rule, files []string) (map[string]string, error) {
	diffs := make(map[string]string)

	for _, file := range files {
		// Read original
		original, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Format with Prettier (write to temp file)
		formatted, err := e.formatWithPrettier(ctx, rule, file)
		if err != nil {
			continue
		}

		// Generate diff
		diff := generateUnifiedDiff(file, string(original), formatted)
		if diff != "" {
			diffs[file] = diff
		}
	}

	return diffs, nil
}

// formatWithPrettier formats a file and returns the result.
func (e *Engine) formatWithPrettier(ctx context.Context, rule core.Rule, file string) (string, error) {
	config, err := e.prettier.GenerateConfig(&rule)
	if err != nil {
		return "", err
	}

	// Write config to temp file
	tmpConfig, err := os.CreateTemp("", "prettierrc-*.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpConfig.Name())

	if _, err := tmpConfig.Write(config); err != nil {
		return "", err
	}
	tmpConfig.Close()

	// Run prettier
	cmd := exec.CommandContext(ctx, "prettier", "--config", tmpConfig.Name(), file)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// generateUnifiedDiff generates a unified diff between original and formatted.
func generateUnifiedDiff(filename, original, formatted string) string {
	if original == formatted {
		return ""
	}

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- %s\n", filename))
	diff.WriteString(fmt.Sprintf("+++ %s (formatted)\n", filename))

	origLines := strings.Split(original, "\n")
	formattedLines := strings.Split(formatted, "\n")

	// Simple line-by-line diff (not optimal, but works)
	maxLines := len(origLines)
	if len(formattedLines) > maxLines {
		maxLines = len(formattedLines)
	}

	for i := 0; i < maxLines; i++ {
		var origLine, formattedLine string

		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(formattedLines) {
			formattedLine = formattedLines[i]
		}

		if origLine != formattedLine {
			if origLine != "" {
				diff.WriteString(fmt.Sprintf("-%s\n", origLine))
			}
			if formattedLine != "" {
				diff.WriteString(fmt.Sprintf("+%s\n", formattedLine))
			}
		}
	}

	return diff.String()
}
