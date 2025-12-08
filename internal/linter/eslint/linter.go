package eslint

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

// Linter wraps ESLint for JavaScript/TypeScript validation.
//
// ESLint is the universal linter for JavaScript:
// - Pattern rules: id-match, no-restricted-syntax, no-restricted-imports
// - Length rules: max-len, max-lines, max-params, max-lines-per-function
// - Style rules: indent, quotes, semi, comma-dangle
// - AST rules: Custom rule generation
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	// ToolsDir is where ESLint is installed
	// Default: ~/.sym/tools
	ToolsDir string

	// executor runs ESLint subprocess
	executor *linter.SubprocessExecutor
}

// New creates a new ESLint linter.
func New(toolsDir string) *Linter {
	if toolsDir == "" {
		toolsDir = linter.DefaultToolsDir()
	}

	return &Linter{
		ToolsDir: toolsDir,
		executor: linter.NewSubprocessExecutor(),
	}
}

// Name returns the linter name.
func (l *Linter) Name() string {
	return "eslint"
}

// GetCapabilities returns the ESLint linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:                "eslint",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"pattern", "length", "style", "ast"},
		Version:             "^8.0.0",
	}
}

// CheckAvailability checks if ESLint is installed.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	// Try local installation first
	eslintPath := l.getESLintPath()
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
		version = "^8.0.0" // Default to ESLint 8.x
	}

	// Initialize package.json if needed
	packageJSON := filepath.Join(l.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := l.initPackageJSON(); err != nil {
			return fmt.Errorf("failed to init package.json: %w", err)
		}
	}

	// Install ESLint
	l.executor.WorkDir = l.ToolsDir
	_, err := l.executor.Execute(ctx, "npm", "install", fmt.Sprintf("eslint@%s", version))
	if err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}

// Execute runs ESLint with the given config and files.
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	// Implementation in executor.go
	return l.execute(ctx, config, files)
}

// ParseOutput converts ESLint JSON output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	// Implementation in parser.go
	return parseOutput(output)
}

// getESLintPath returns the path to local ESLint binary.
func (l *Linter) getESLintPath() string {
	return filepath.Join(l.ToolsDir, "node_modules", ".bin", "eslint")
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
