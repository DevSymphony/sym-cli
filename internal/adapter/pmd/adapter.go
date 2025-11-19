package pmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

const (
	// DefaultVersion is the default PMD version.
	DefaultVersion = "7.0.0"

	// GitHubReleaseURL is the GitHub releases base URL.
	GitHubReleaseURL = "https://github.com/pmd/pmd/releases/download"
)

// Adapter wraps PMD for Java validation.
//
// PMD handles:
// - Pattern rules: custom XPath rules
// - Complexity rules: cyclomatic complexity, nesting depth
// - Performance rules: inefficient code patterns
// - Security rules: hardcoded credentials, SQL injection
// - Error handling rules: empty catch blocks, exception handling
type Adapter struct {
	// ToolsDir is where PMD is installed.
	// Default: ~/.sym/tools
	ToolsDir string

	// WorkDir is the project root.
	WorkDir string

	// PMDPath is the path to pmd executable.
	// Empty = use default location
	PMDPath string

	// executor runs subprocess
	executor *adapter.SubprocessExecutor
}

// NewAdapter creates a new PMD adapter.
func NewAdapter(toolsDir, workDir string) *Adapter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
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
	return "pmd"
}

// GetCapabilities returns the PMD adapter capabilities.
func (a *Adapter) GetCapabilities() adapter.AdapterCapabilities {
	return adapter.AdapterCapabilities{
		Name:                "pmd",
		SupportedLanguages:  []string{"java"},
		SupportedCategories: []string{"pattern", "complexity", "performance", "security", "error_handling", "ast"},
		Version:             DefaultVersion,
	}
}

// CheckAvailability checks if PMD is available.
func (a *Adapter) CheckAvailability(ctx context.Context) error {
	pmdPath := a.getPMDPath()

	// Check if PMD binary exists
	if _, err := os.Stat(pmdPath); os.IsNotExist(err) {
		return fmt.Errorf("pmd not found at %s: run Install first", pmdPath)
	}

	// Try to run PMD version check
	cmd := exec.CommandContext(ctx, pmdPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pmd execution failed: %w", err)
	}

	return nil
}

// Install downloads and extracts PMD from GitHub releases.
func (a *Adapter) Install(ctx context.Context, config adapter.InstallConfig) error {
	// Ensure tools directory exists
	if err := os.MkdirAll(a.ToolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}

	// Determine version
	version := config.Version
	if version == "" {
		version = DefaultVersion
	}

	// PMD distribution filename
	distName := fmt.Sprintf("pmd-dist-%s-bin.zip", version)
	url := fmt.Sprintf("%s/pmd_releases%%2F%s/%s", GitHubReleaseURL, version, distName)

	// Destination paths
	zipPath := filepath.Join(a.ToolsDir, distName)
	extractDir := filepath.Join(a.ToolsDir, fmt.Sprintf("pmd-bin-%s", version))

	// Check if already exists
	if !config.Force {
		if _, err := os.Stat(extractDir); err == nil {
			return nil // Already installed
		}
	}

	// Download
	if err := a.downloadFile(ctx, url, zipPath); err != nil {
		return fmt.Errorf("failed to download PMD: %w", err)
	}
	defer func() { _ = os.Remove(zipPath) }()

	// Extract (simplified - in production use archive/zip)
	// For now, assume unzip command is available
	cmd := exec.CommandContext(ctx, "unzip", "-q", "-o", zipPath, "-d", a.ToolsDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract PMD: %w (try installing unzip)", err)
	}

	// Make PMD binary executable
	pmdBin := a.getPMDPath()
	if err := os.Chmod(pmdBin, 0755); err != nil {
		return fmt.Errorf("failed to make PMD executable: %w", err)
	}

	return nil
}


// Execute runs PMD with the given config and files.
func (a *Adapter) Execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	return a.execute(ctx, config, files)
}

// ParseOutput converts PMD JSON output to violations.
func (a *Adapter) ParseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	return parseOutput(output)
}

// getPMDPath returns the path to PMD binary.
func (a *Adapter) getPMDPath() string {
	if a.PMDPath != "" {
		return a.PMDPath
	}

	pmdDir := filepath.Join(a.ToolsDir, fmt.Sprintf("pmd-bin-%s", DefaultVersion))

	// PMD binary name depends on OS
	binName := "pmd"
	if runtime.GOOS == "windows" {
		binName = "pmd.bat"
	}

	return filepath.Join(pmdDir, "bin", binName)
}

// downloadFile downloads a file from URL to destPath.
func (a *Adapter) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tempFile := destPath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	// Rename temp to final
	if err := os.Rename(tempFile, destPath); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	return nil
}
