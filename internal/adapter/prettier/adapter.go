package prettier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

// Adapter wraps Prettier for code formatting.
//
// Prettier handles:
// - Style validation (--check mode)
// - Auto-fixing (--write mode)
// - Config: indent, quote, semi, trailingComma, etc.
type Adapter struct {
	ToolsDir string
	WorkDir  string
	executor *adapter.SubprocessExecutor
}

// NewAdapter creates a new Prettier adapter.
func NewAdapter(toolsDir, workDir string) *Adapter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
		// symphonyclient integration: .symphony → .sym directory
		toolsDir = filepath.Join(home, ".sym", "tools")
	}

	return &Adapter{
		ToolsDir: toolsDir,
		WorkDir:  workDir,
		executor: adapter.NewSubprocessExecutor(),
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "prettier"
}

// GetCapabilities returns the Prettier adapter capabilities.
func (a *Adapter) GetCapabilities() adapter.AdapterCapabilities {
	return adapter.AdapterCapabilities{
		Name:                "prettier",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx", "json", "yaml", "css", "html", "markdown"},
		SupportedCategories: []string{"style"},
		Version:             "^3.0.0",
	}
}

// CheckAvailability checks if Prettier is installed.
func (a *Adapter) CheckAvailability(ctx context.Context) error {
	prettierPath := a.getPrettierPath()
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
func (a *Adapter) Install(ctx context.Context, config adapter.InstallConfig) error {
	if err := os.MkdirAll(a.ToolsDir, 0755); err != nil {
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
	packageJSON := filepath.Join(a.ToolsDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		if err := a.initPackageJSON(); err != nil {
			return err
		}
	}

	a.executor.WorkDir = a.ToolsDir
	_, err := a.executor.Execute(ctx, "npm", "install", fmt.Sprintf("prettier@%s", version))
	return err
}


// Execute runs Prettier with the given config and files.
// mode: "check" or "write"
func (a *Adapter) Execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	return a.execute(ctx, config, files, "check")
}

// ExecuteWithMode runs Prettier with specified mode.
func (a *Adapter) ExecuteWithMode(ctx context.Context, config []byte, files []string, mode string) (*adapter.ToolOutput, error) {
	return a.execute(ctx, config, files, mode)
}

// ParseOutput converts Prettier output to violations.
func (a *Adapter) ParseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	return parseOutput(output)
}

func (a *Adapter) getPrettierPath() string {
	return filepath.Join(a.ToolsDir, "node_modules", ".bin", "prettier")
}

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
