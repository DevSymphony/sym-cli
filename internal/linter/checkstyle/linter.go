package checkstyle

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time interface check
var _ linter.Linter = (*Linter)(nil)

const (
	// DefaultVersion is the default Checkstyle version.
	DefaultVersion = "10.26.1"

	// GitHubReleaseURL is the GitHub Releases base URL for Checkstyle.
	GitHubReleaseURL = "https://github.com/checkstyle/checkstyle/releases/download"
)

// Linter wraps Checkstyle for Java validation.
//
// Checkstyle handles:
// - Pattern rules: naming conventions, regex patterns
// - Length rules: line length, file length
// - Style rules: indentation, whitespace
// - Naming rules: class names, method names, variable names
//
// Note: Linter is goroutine-safe and stateless. WorkDir is determined
// by CWD at execution time, not stored in the linter.
type Linter struct {
	// ToolsDir is where Checkstyle JAR is stored.
	// Default: ~/.sym/tools
	ToolsDir string

	// JavaPath is the path to java executable.
	// Empty = use system java
	JavaPath string

	// executor runs subprocess
	executor *linter.SubprocessExecutor
}

// New creates a new Checkstyle linter.
func New(toolsDir string) *Linter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
		toolsDir = filepath.Join(home, ".sym", "tools")
	}

	javaPath, _ := exec.LookPath("java")

	return &Linter{
		ToolsDir: toolsDir,
		JavaPath: javaPath,
		executor: linter.NewSubprocessExecutor(),
	}
}

// Name returns the linter name.
func (l *Linter) Name() string {
	return "checkstyle"
}

// GetCapabilities returns the Checkstyle linter capabilities.
func (l *Linter) GetCapabilities() linter.Capabilities {
	return linter.Capabilities{
		Name:                "checkstyle",
		SupportedLanguages:  []string{"java"},
		SupportedCategories: []string{"pattern", "length", "style", "naming"},
		Version:             DefaultVersion,
	}
}

// CheckAvailability checks if Java and Checkstyle JAR are available.
func (l *Linter) CheckAvailability(ctx context.Context) error {
	// Check Java
	if l.JavaPath == "" {
		return fmt.Errorf("java not found: please install Java")
	}

	// Verify Java version
	cmd := exec.CommandContext(ctx, l.JavaPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("java execution failed: %w", err)
	}

	// Check Checkstyle JAR
	jarPath := l.getJARPath()
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return fmt.Errorf("checkstyle JAR not found at %s: run Install first", jarPath)
	}

	return nil
}

// Install downloads Checkstyle JAR from Maven Central.
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

	// Download URL - use GitHub Releases for -all.jar
	jarName := fmt.Sprintf("checkstyle-%s-all.jar", version)
	url := fmt.Sprintf("%s/checkstyle-%s/%s", GitHubReleaseURL, version, jarName)

	// Destination path
	jarPath := filepath.Join(l.ToolsDir, jarName)

	// Check if already exists and not forcing reinstall
	if !config.Force {
		if _, err := os.Stat(jarPath); err == nil {
			return nil // Already installed
		}
	}

	// Download
	if err := l.downloadFile(ctx, url, jarPath); err != nil {
		return fmt.Errorf("failed to download checkstyle: %w", err)
	}

	return nil
}

// Execute runs Checkstyle with the given config and files.
func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
	return l.execute(ctx, config, files)
}

// ParseOutput converts Checkstyle JSON output to violations.
func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
	return parseOutput(output)
}

// getJARPath returns the path to Checkstyle JAR.
func (l *Linter) getJARPath() string {
	return filepath.Join(l.ToolsDir, fmt.Sprintf("checkstyle-%s-all.jar", DefaultVersion))
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
