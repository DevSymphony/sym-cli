package tsc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Adapter wraps TypeScript Compiler (tsc) for type checking.
//
// TSC provides:
// - Type checking for TypeScript and JavaScript files
// - Compilation errors and warnings
// - Interface and type validation
//
// Note: Adapter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the adapter.
type Adapter struct {
	// ToolsDir is where TypeScript is installed
	// Default: ~/.sym/tools
	ToolsDir string

	// executor runs tsc subprocess
	executor *adapter.SubprocessExecutor
}

// NewAdapter creates a new TSC adapter.
func NewAdapter(toolsDir string) *Adapter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
		toolsDir = filepath.Join(home, ".sym", "tools")
	}

	return &Adapter{
		ToolsDir: toolsDir,
		executor: adapter.NewSubprocessExecutor(),
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "tsc"
}

// GetCapabilities returns the TSC adapter capabilities.
func (a *Adapter) GetCapabilities() adapter.AdapterCapabilities {
	return adapter.AdapterCapabilities{
		Name:                "tsc",
		SupportedLanguages:  []string{"typescript"},
		SupportedCategories: []string{"typechecker"},
		Version:             "^5.0.0",
	}
}

// CheckAvailability checks if tsc is installed.
func (a *Adapter) CheckAvailability(ctx context.Context) error {
	// Try local installation first
	tscPath := a.getTSCPath()
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
func (a *Adapter) Install(ctx context.Context, config adapter.InstallConfig) error {
	// Ensure tools directory exists
	if err := os.MkdirAll(a.ToolsDir, 0755); err != nil {
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
	packageJSON := filepath.Join(a.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := a.initPackageJSON(); err != nil {
			return fmt.Errorf("failed to init package.json: %w", err)
		}
	}

	// Install TypeScript
	a.executor.WorkDir = a.ToolsDir
	_, err := a.executor.Execute(ctx, "npm", "install", fmt.Sprintf("typescript@%s", version))
	if err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}


// Execute runs tsc with the given config and files.
// Returns type checking results.
func (a *Adapter) Execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	return a.execute(ctx, config, files)
}

// ParseOutput converts tsc output to violations.
func (a *Adapter) ParseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	return parseOutput(output)
}

// getTSCPath returns the path to local tsc binary.
func (a *Adapter) getTSCPath() string {
	return filepath.Join(a.ToolsDir, "node_modules", ".bin", "tsc")
}

// initPackageJSON creates a minimal package.json.
func (a *Adapter) initPackageJSON() error {
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

	path := filepath.Join(a.ToolsDir, "package.json")
	return os.WriteFile(path, data, 0644)
}
