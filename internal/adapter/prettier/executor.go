package prettier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// execute runs Prettier with the given config and files.
// mode: "check" (validation only) or "write" (autofix)
func (a *Adapter) execute(ctx context.Context, config []byte, files []string, mode string) (*adapter.ToolOutput, error) {
	if len(files) == 0 {
		return &adapter.ToolOutput{ExitCode: 0}, nil
	}

	// Write config to temp file
	configPath, err := a.writeConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	// Determine Prettier command
	prettierCmd := a.getPrettierCommand()

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
	a.executor.WorkDir = a.WorkDir
	output, err := a.executor.Execute(ctx, prettierCmd, args...)

	// Prettier returns non-zero exit code if files need formatting (in --check mode)
	// This is expected, not an error
	if err != nil {
		return output, nil
	}

	return output, nil
}

func (a *Adapter) getPrettierCommand() string {
	localPath := a.getPrettierPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}
	return "prettier"
}

func (a *Adapter) writeConfigFile(config []byte) (string, error) {
	tmpDir := filepath.Join(a.ToolsDir, ".tmp")
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
