package tsc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time interface check
var _ linter.Linter = (*Linter)(nil)

// Linter wraps TypeScript Compiler (tsc) for type checking.
//
// TSC provides:
// - Type checking for TypeScript and JavaScript files
// - Compilation errors and warnings
// - Interface and type validation
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	// ToolsDir is where TypeScript is installed
	// Default: ~/.sym/tools
	ToolsDir string

	// executor runs tsc subprocess
	executor *linter.SubprocessExecutor
}

// New creates a new TSC linter.
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
	return "tsc"
}

// GetCapabilities returns the TSC linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:                "tsc",
		SupportedLanguages:  []string{"typescript"},
		SupportedCategories: []string{"typechecker"},
		Version:             "^5.0.0",
	}
}

// CheckAvailability checks if tsc is installed.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	// Try local installation first
	tscPath := l.getTSCPath()
	if _, err := os.Stat(tscPath); err == nil {
		return nil // Found in tools dir
	}

	// Try global installation
	cmd := exec.CommandContext(ctx, "tsc", "--version")
	if err := cmd.Run(); err == nil {
		return nil // Found globally
	}

	return fmt.Errorf("tsc not found (checked: %s and global PATH)", tscPath)
}

// Install installs TypeScript via npm.
func (l *Linter) Install(ctx context.Context, config linter.InstallConfig) error {
	// Ensure tools directory exists
	if err := os.MkdirAll(l.ToolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found: please install Node.js first")
	}

	// Determine version
	version := config.Version
	if version == "" {
		version = "^5.0.0" // Default to TypeScript 5.x
	}

	// Initialize package.json if needed
	packageJSON := filepath.Join(l.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := l.initPackageJSON(); err != nil {
			return fmt.Errorf("failed to init package.json: %w", err)
		}
	}

	// Install TypeScript
	l.executor.WorkDir = l.ToolsDir
	_, err := l.executor.Execute(ctx, "npm", "install", fmt.Sprintf("typescript@%s", version))
	if err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}


// Execute runs tsc with the given config and files.
// Returns type checking results.
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	return l.execute(ctx, config, files)
}

// ParseOutput converts tsc output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	return parseOutput(output)
}

// getTSCPath returns the path to local tsc binary.
func (l *Linter) getTSCPath() string {
	return filepath.Join(l.ToolsDir, "node_modules", ".bin", "tsc")
}

// initPackageJSON creates a minimal package.json.
func (l *Linter) initPackageJSON() error {
	pkg := map[string]interface{}{
		"name":        "symphony-tools",
		"version":     "1.0.0",
		"description": "Symphony validation tools",
		"private":     true,
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(l.ToolsDir, "package.json")
	return os.WriteFile(path, data, 0644)
}
