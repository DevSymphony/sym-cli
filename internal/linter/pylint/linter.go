package pylint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time interface check
var _ linter.Linter = (*Linter)(nil)

// Linter wraps Pylint for Python static analysis.
//
// Pylint is the comprehensive Python linter:
// - Naming rules: invalid-name, disallowed-name
// - Style rules: line-too-long, trailing-whitespace
// - Documentation rules: missing-docstring variants
// - Error handling rules: bare-except, broad-except
// - Complexity rules: too-many-branches, too-many-arguments
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	// ToolsDir is where Pylint virtualenv is installed
	// Default: ~/.sym/tools
	ToolsDir string

	// PylintPath is the path to pylint executable (optional override)
	PylintPath string

	// executor runs Pylint subprocess
	executor *linter.SubprocessExecutor
}

// New creates a new Pylint linter.
func New(toolsDir string) *Linter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
		toolsDir = filepath.Join(home, ".sym", "tools")
	}

	return &Linter{
		ToolsDir: toolsDir,
		executor: linter.NewSubprocessExecutor(),
	}
}

// Name returns the linter name.
func (l *Linter) Name() string {
	return "pylint"
}

// GetCapabilities returns the Pylint linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:               "pylint",
		SupportedLanguages: []string{"python", "py"},
		SupportedCategories: []string{
			"naming",
			"style",
			"documentation",
			"error_handling",
			"complexity",
			"pattern",
			"security",
			"ast",
		},
		Version: ">=3.0.0",
	}
}

// CheckAvailability checks if Pylint is installed.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	// Try local installation first (virtualenv)
	pylintPath := l.getPylintPath()
	if _, err := os.Stat(pylintPath); err == nil {
		return nil // Found in tools dir
	}

	// Try global installation
	cmd := exec.CommandContext(ctx, "pylint", "--version")
	if err := cmd.Run(); err == nil {
		return nil // Found globally
	}

	return fmt.Errorf("pylint not found (checked: %s and global PATH)", pylintPath)
}

// Install installs Pylint via pip in a virtualenv.
func (l *Linter) Install(ctx context.Context, config linter.InstallConfig) error {
	// Ensure tools directory exists
	if err := os.MkdirAll(l.ToolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}

	// Check if Python is available
	pythonCmd := l.getPythonCommand()
	if _, err := exec.LookPath(pythonCmd); err != nil {
		return fmt.Errorf("python not found: please install Python 3.8+ first")
	}

	venvPath := l.getVenvPath()
	pipPath := l.getPipPath()
	pylintPath := l.getPylintPath()

	// Check if venv exists but is incomplete (no pip or no pylint)
	if _, err := os.Stat(venvPath); err == nil {
		hasPip := true
		hasPylint := true
		if _, err := os.Stat(pipPath); os.IsNotExist(err) {
			hasPip = false
		}
		if _, err := os.Stat(pylintPath); os.IsNotExist(err) {
			hasPylint = false
		}
		// If venv exists but is incomplete, remove it
		if !hasPip || !hasPylint {
			if err := os.RemoveAll(venvPath); err != nil {
				return fmt.Errorf("failed to remove incomplete venv: %w", err)
			}
		}
	}

	// Create virtualenv if it doesn't exist
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		output, err := l.executor.Execute(ctx, pythonCmd, "-m", "venv", venvPath)
		if err != nil {
			return fmt.Errorf("failed to create virtualenv: %w", err)
		}
		if output.ExitCode != 0 {
			// Check for common venv issues (Python venv outputs errors to stdout)
			errMsg := output.Stderr
			if errMsg == "" {
				errMsg = output.Stdout
			}
			if errMsg == "" {
				errMsg = "venv creation failed (no error message)"
			}
			if strings.Contains(errMsg, "ensurepip") || strings.Contains(errMsg, "python3-venv") {
				return fmt.Errorf("failed to create virtualenv: python3-venv package not installed. " +
					"On Debian/Ubuntu, run: sudo apt install python3-venv")
			}
			return fmt.Errorf("failed to create virtualenv: %s", errMsg)
		}
	}

	// Ensure pip is available (some Linux distros don't include pip in venv by default)
	if _, err := os.Stat(pipPath); os.IsNotExist(err) {
		pythonInVenv := l.getPythonInVenv()
		output, err := l.executor.Execute(ctx, pythonInVenv, "-m", "ensurepip", "--upgrade")
		if err != nil {
			return fmt.Errorf("failed to install pip via ensurepip: %w", err)
		}
		if output.ExitCode != 0 {
			return fmt.Errorf("failed to install pip via ensurepip: %s", output.Stderr)
		}
	}

	// Determine version
	version := config.Version
	if version == "" {
		version = ">=3.0.0"
	}

	// Install Pylint in virtualenv
	output, err := l.executor.Execute(ctx, pipPath, "install", fmt.Sprintf("pylint%s", version))
	if err != nil {
		return fmt.Errorf("pip install failed: %w", err)
	}
	if output.ExitCode != 0 {
		return fmt.Errorf("pip install failed: %s", output.Stderr)
	}

	return nil
}

// Execute runs Pylint with the given config and files.
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	// Implementation in executor.go
	return l.execute(ctx, config, files)
}

// ParseOutput converts Pylint JSON output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	// Implementation in parser.go
	return parseOutput(output)
}

// getVenvPath returns the path to Pylint virtualenv.
func (l *Linter) getVenvPath() string {
	return filepath.Join(l.ToolsDir, "pylint-venv")
}

// getPylintPath returns the path to local Pylint binary.
func (l *Linter) getPylintPath() string {
	venvPath := l.getVenvPath()
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "pylint.exe")
	}
	return filepath.Join(venvPath, "bin", "pylint")
}

// getPipPath returns the path to pip in virtualenv.
func (l *Linter) getPipPath() string {
	venvPath := l.getVenvPath()
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "pip.exe")
	}
	return filepath.Join(venvPath, "bin", "pip")
}

// getPythonCommand returns the Python command to use.
func (l *Linter) getPythonCommand() string {
	// Try python3 first, then python
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	return "python"
}

// getPythonInVenv returns the path to Python in virtualenv.
func (l *Linter) getPythonInVenv() string {
	venvPath := l.getVenvPath()
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// getPylintCommand returns the Pylint command to use.
func (l *Linter) getPylintCommand() string {
	// If explicit path is set, use it
	if l.PylintPath != "" {
		return l.PylintPath
	}

	// Try local installation first (virtualenv)
	localPath := l.getPylintPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Try global pylint
	if _, err := exec.LookPath("pylint"); err == nil {
		return "pylint"
	}

	// Fall back to local path (will fail with proper error)
	return localPath
}
