package prettier

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

// Linter wraps Prettier for code formatting.
//
// Prettier handles:
// - Style validation (--check mode)
// - Auto-fixing (--write mode)
// - Config: indent, quote, semi, trailingComma, etc.
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	ToolsDir string
	executor *linter.SubprocessExecutor
}

// New creates a new Prettier linter.
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
	return "prettier"
}

// GetCapabilities returns the Prettier linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:                "prettier",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx", "json", "yaml", "css", "html", "markdown"},
		SupportedCategories: []string{"style"},
		Version:             "^3.0.0",
	}
}

// CheckAvailability checks if Prettier is installed.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	prettierPath := l.getPrettierPath()
	if _, err := os.Stat(prettierPath); err == nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, "prettier", "--version")
	if err := cmd.Run(); err == nil {
		return nil
	}

	return fmt.Errorf("prettier not found")
}

// Install installs Prettier via npm.
func (l *Linter) Install(ctx context.Context, config linter.InstallConfig) error {
	if err := os.MkdirAll(l.ToolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}

	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found: please install Node.js first")
	}

	version := config.Version
	if version == "" {
		version = "^3.0.0"
	}

	// Init package.json if needed
	packageJSON := filepath.Join(l.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := l.initPackageJSON(); err != nil {
			return err
		}
	}

	l.executor.WorkDir = l.ToolsDir
	_, err := l.executor.Execute(ctx, "npm", "install", fmt.Sprintf("prettier@%s", version))
	return err
}


// Execute runs Prettier with the given config and files.
// mode: "check" or "write"
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	return l.execute(ctx, config, files, "check")
}

// ExecuteWithMode runs Prettier with specified mode.
func (l *Linter) ExecuteWithMode(ctx context.Context, config []byte, files []string, mode string) (*linter.ToolOutput, error) {
	return l.execute(ctx, config, files, mode)
}

// ParseOutput converts Prettier output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	return parseOutput(output)
}

func (l *Linter) getPrettierPath() string {
	return filepath.Join(l.ToolsDir, "node_modules", ".bin", "prettier")
}

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
