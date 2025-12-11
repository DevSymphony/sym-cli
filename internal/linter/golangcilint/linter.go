package golangcilint

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time interface check
var _ linter.Linter = (*Linter)(nil)

const (
	// DefaultVersion is the default golangci-lint version.
	DefaultVersion = "2.7.2"

	// GitHubReleaseURL is the GitHub releases base URL.
	GitHubReleaseURL = "https://github.com/golangci/golangci-lint/releases/download"
)

// Linter wraps golangci-lint for Go validation.
//
// golangci-lint is a meta-linter that runs 50+ Go linters in parallel:
// - errcheck: Check for unchecked errors
// - govet: Vet examines Go source code
// - staticcheck: Advanced static analysis
// - gosec: Security checker for Go code
// - ineffassign: Detects ineffectual assignments
// - unused: Checks for unused code
// - goconst: Finds repeated strings
// - gocyclo: Cyclomatic complexity checker
// - And many more...
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	// ToolsDir is where golangci-lint is installed.
	// Default: ~/.sym/tools
	ToolsDir string

	// GolangciLintPath is the path to golangci-lint executable.
	// Empty = use default location
	GolangciLintPath string

	// executor runs subprocess
	executor *linter.SubprocessExecutor
}

// New creates a new golangci-lint linter.
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
	return "golangci-lint"
}

// GetCapabilities returns the golangci-lint linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:               "golangci-lint",
		SupportedLanguages: []string{"go"},
		SupportedCategories: []string{
			"bugs",
			"style",
			"performance",
			"complexity",
			"error_handling",
			"security",
			"unused",
			"naming",
			"ast",
		},
		Version: DefaultVersion,
	}
}

// CheckAvailability checks if golangci-lint is available.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	golangciLintPath := l.getGolangciLintPath()

	// Check if golangci-lint binary exists
	if _, err := os.Stat(golangciLintPath); os.IsNotExist(err) {
		return fmt.Errorf("golangci-lint not found at %s: run Install first", golangciLintPath)
	}

	// Try to run golangci-lint version check
	cmd := exec.CommandContext(ctx, golangciLintPath, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golangci-lint execution failed: %w", err)
	}

	return nil
}

// Install downloads and extracts golangci-lint from GitHub releases.
func (l *Linter) Install(ctx context.Context, config linter.InstallConfig) error {
	// Ensure tools directory exists
	if err := os.MkdirAll(l.ToolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools dir: %w", err)
	}

	// Determine version
	version := config.Version
	if version == "" {
		version = DefaultVersion
	}

	// Get download URL and archive extension
	url, ext, err := l.getDownloadURL(version)
	if err != nil {
		return err
	}

	// Destination paths
	archiveName := filepath.Base(url)
	archivePath := filepath.Join(l.ToolsDir, archiveName)
	installDir := filepath.Join(l.ToolsDir, fmt.Sprintf("golangci-lint-%s", version))

	// Check if already exists
	if !config.Force {
		if _, err := os.Stat(installDir); err == nil {
			return nil // Already installed
		}
	}

	// Download
	if err := l.downloadFile(ctx, url, archivePath); err != nil {
		return fmt.Errorf("failed to download golangci-lint: %w", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	// Extract based on archive type
	if err := l.extractArchive(ctx, archivePath, ext, version, installDir); err != nil {
		return fmt.Errorf("failed to extract golangci-lint: %w", err)
	}

	// Make golangci-lint binary executable (Unix only)
	if runtime.GOOS != "windows" {
		binaryPath := l.getGolangciLintPath()
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make golangci-lint executable: %w", err)
		}
	}

	return nil
}

// Execute runs golangci-lint with the given config and files.
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	return l.execute(ctx, config, files)
}

// ParseOutput converts golangci-lint JSON output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	return parseOutput(output)
}

// getGolangciLintPath returns the path to golangci-lint binary.
func (l *Linter) getGolangciLintPath() string {
	if l.GolangciLintPath != "" {
		return l.GolangciLintPath
	}

	installDir := filepath.Join(l.ToolsDir, fmt.Sprintf("golangci-lint-%s", DefaultVersion))

	// Binary name depends on OS
	binName := "golangci-lint"
	if runtime.GOOS == "windows" {
		binName = "golangci-lint.exe"
	}

	return filepath.Join(installDir, binName)
}

// getDownloadURL constructs the download URL based on OS and architecture.
func (l *Linter) getDownloadURL(version string) (string, string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go OS/ARCH to golangci-lint naming
	var osName, archName, ext string
	switch goos {
	case "linux":
		osName = "linux"
		ext = "tar.gz"
	case "darwin":
		osName = "darwin"
		ext = "tar.gz"
	case "windows":
		osName = "windows"
		ext = "zip"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", goos)
	}

	switch goarch {
	case "amd64":
		archName = "amd64"
	case "arm64":
		archName = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	fileName := fmt.Sprintf("golangci-lint-%s-%s-%s.%s", version, osName, archName, ext)
	url := fmt.Sprintf("%s/v%s/%s", GitHubReleaseURL, version, fileName)

	return url, ext, nil
}

// extractArchive extracts the downloaded archive.
func (l *Linter) extractArchive(ctx context.Context, archivePath, ext, version, installDir string) error {
	// Create temporary extraction directory
	tempDir := filepath.Join(l.ToolsDir, ".tmp-extract")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	var cmd *exec.Cmd
	if ext == "tar.gz" {
		// Extract tar.gz for Linux/macOS
		cmd = exec.CommandContext(ctx, "tar", "-xzf", archivePath, "-C", tempDir)
	} else {
		// Extract zip for Windows
		cmd = exec.CommandContext(ctx, "unzip", "-q", "-o", archivePath, "-d", tempDir)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extraction failed: %w (ensure tar/unzip is installed)", err)
	}

	// Find the extracted directory (format: golangci-lint-{version}-{os}-{arch})
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp dir: %w", err)
	}

	var extractedDir string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "golangci-lint-") {
			extractedDir = filepath.Join(tempDir, entry.Name())
			break
		}
	}

	if extractedDir == "" {
		return fmt.Errorf("extracted directory not found in %s", tempDir)
	}

	// Move to final installation directory
	if err := os.RemoveAll(installDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old installation: %w", err)
	}

	if err := os.Rename(extractedDir, installDir); err != nil {
		return fmt.Errorf("failed to move to installation dir: %w", err)
	}

	return nil
}

// downloadFile downloads a file from URL to destPath.
func (l *Linter) downloadFile(ctx context.Context, url, destPath string) error {
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
		return fmt.Errorf("download failed: HTTP %d for URL %s", resp.StatusCode, url)
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
