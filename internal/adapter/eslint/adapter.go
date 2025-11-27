package eslint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Adapter wraps ESLint for JavaScript/TypeScript validation.
//
// ESLint is the universal adapter for JavaScript:
// - Pattern rules: id-match, no-restricted-syntax, no-restricted-imports
// - Length rules: max-len, max-lines, max-params, max-lines-per-function
// - Style rules: indent, quotes, semi, comma-dangle
// - AST rules: Custom rule generation
//
// Note: Adapter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the adapter.
type Adapter struct {
	// ToolsDir is where ESLint is installed
	// Default: ~/.sym/tools
	ToolsDir string

	// executor runs ESLint subprocess
	executor *adapter.SubprocessExecutor
}

// NewAdapter creates a new ESLint adapter.
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
	return "eslint"
}

// GetCapabilities returns the ESLint adapter capabilities.
func (a *Adapter) GetCapabilities() adapter.AdapterCapabilities {
	return adapter.AdapterCapabilities{
		Name:                "eslint",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"pattern", "length", "style", "ast"},
		Version:             "^8.0.0",
	}
}

// CheckAvailability checks if ESLint is installed.
func (a *Adapter) CheckAvailability(ctx context.Context) error {
	// Try local installation first
	eslintPath := a.getESLintPath()
	if _, err := os.Stat(eslintPath); err == nil {
		return nil // Found in tools dir
	}

	// Try global installation
	cmd := exec.CommandContext(ctx, "eslint", "--version")
	if err := cmd.Run(); err == nil {
		return nil // Found globally
	}

	return fmt.Errorf("eslint not found (checked: %s and global PATH)", eslintPath)
}

// Install installs ESLint via npm.
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
		version = "^8.0.0" // Default to ESLint 8.x
	}

	// Initialize package.json if needed
	packageJSON := filepath.Join(a.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := a.initPackageJSON(); err != nil {
			return fmt.Errorf("failed to init package.json: %w", err)
		}
	}

	// Install ESLint
	a.executor.WorkDir = a.ToolsDir
	_, err := a.executor.Execute(ctx, "npm", "install", fmt.Sprintf("eslint@%s", version))
	if err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}


// Execute runs ESLint with the given config and files.
func (a *Adapter) Execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	// Implementation in executor.go
	return a.execute(ctx, config, files)
}

// ParseOutput converts ESLint JSON output to violations.
func (a *Adapter) ParseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	// Implementation in parser.go
	return parseOutput(output)
}

// getESLintPath returns the path to local ESLint binary.
func (a *Adapter) getESLintPath() string {
	return filepath.Join(a.ToolsDir, "node_modules", ".bin", "eslint")
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
