package prettier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// execute runs Prettier with the given config and files.
// mode: "check" (validation only) or "write" (autofix)
func (l *Linter) execute(ctx context.Context, config []byte, files []string, mode string) (*linter.ToolOutput, error) {
	if len(files) == 0 {
		return &linter.ToolOutput{ExitCode: 0}, nil
	}

	// Write config to temp file
	configPath, err := l.writeConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	// Determine Prettier command
	prettierCmd := l.getPrettierCommand()

	// Build arguments
	args := []string{
		"--config", configPath,
	}

	switch mode {
	case "check":
		args = append(args, "--check")
	case "write":
		args = append(args, "--write")
	}

	args = append(args, files...)

	// Execute
	// Reset WorkDir to use CWD (Install() may have set it to ToolsDir)
	l.executor.WorkDir = ""
	output, err := l.executor.Execute(ctx, prettierCmd, args...)

	// Prettier returns non-zero exit code if files need formatting (in --check mode)
	// This is expected, not an error
	if err != nil {
		return output, nil
	}

	return output, nil
}

func (l *Linter) getPrettierCommand() string {
	localPath := l.getPrettierPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}
	return "prettier"
}

func (l *Linter) writeConfigFile(config []byte) (string, error) {
	tmpDir := filepath.Join(l.ToolsDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(tmpDir, "prettierrc-*.json")
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
