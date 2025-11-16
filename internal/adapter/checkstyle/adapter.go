package checkstyle

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

const (
	// DefaultVersion is the default Checkstyle version.
	DefaultVersion = "10.12.0"

	// MavenCentralURL is the Maven Central repository base URL.
	MavenCentralURL = "https://repo1.maven.org/maven2/com/puppycrawl/tools/checkstyle"
)

// Adapter wraps Checkstyle for Java validation.
//
// Checkstyle handles:
// - Pattern rules: naming conventions, regex patterns
// - Length rules: line length, file length
// - Style rules: indentation, whitespace
// - Naming rules: class names, method names, variable names
type Adapter struct {
	// ToolsDir is where Checkstyle JAR is stored.
	// Default: ~/.sym/tools
	ToolsDir string

	// WorkDir is the project root.
	WorkDir string

	// JavaPath is the path to java executable.
	// Empty = use system java
	JavaPath string

	// executor runs subprocess
	executor *adapter.SubprocessExecutor
}

// NewAdapter creates a new Checkstyle adapter.
func NewAdapter(toolsDir, workDir string) *Adapter {
	if toolsDir == "" {
		home, _ := os.UserHomeDir()
		toolsDir = filepath.Join(home, ".sym", "tools")
	}

	javaPath, _ := exec.LookPath("java")

	return &Adapter{
		ToolsDir: toolsDir,
		WorkDir:  workDir,
		JavaPath: javaPath,
		executor: adapter.NewSubprocessExecutor(),
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "checkstyle"
}

// GetCapabilities returns the Checkstyle adapter capabilities.
func (a *Adapter) GetCapabilities() adapter.AdapterCapabilities {
	return adapter.AdapterCapabilities{
		Name:                "checkstyle",
		SupportedLanguages:  []string{"java"},
		SupportedCategories: []string{"pattern", "length", "style", "naming"},
		Version:             DefaultVersion,
	}
}

// CheckAvailability checks if Java and Checkstyle JAR are available.
func (a *Adapter) CheckAvailability(ctx context.Context) error {
	// Check Java
	if a.JavaPath == "" {
		return fmt.Errorf("java not found: please install Java")
	}

	// Verify Java version
	cmd := exec.CommandContext(ctx, a.JavaPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("java execution failed: %w", err)
	}

	// Check Checkstyle JAR
	jarPath := a.getJARPath()
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return fmt.Errorf("checkstyle JAR not found at %s: run Install first", jarPath)
	}

	return nil
}

// Install downloads Checkstyle JAR from Maven Central.
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

	// Download URL
	jarName := fmt.Sprintf("checkstyle-%s-all.jar", version)
	url := fmt.Sprintf("%s/%s/%s", MavenCentralURL, version, jarName)

	// Destination path
	jarPath := filepath.Join(a.ToolsDir, jarName)

	// Check if already exists and not forcing reinstall
	if !config.Force {
		if _, err := os.Stat(jarPath); err == nil {
			return nil // Already installed
		}
	}

	// Download
	if err := a.downloadFile(ctx, url, jarPath); err != nil {
		return fmt.Errorf("failed to download checkstyle: %w", err)
	}

	return nil
}

// GenerateConfig generates Checkstyle XML config from a rule.
func (a *Adapter) GenerateConfig(rule interface{}) ([]byte, error) {
	return generateConfig(rule)
}

// Execute runs Checkstyle with the given config and files.
func (a *Adapter) Execute(ctx context.Context, config []byte, files []string) (*adapter.ToolOutput, error) {
	return a.execute(ctx, config, files)
}

// ParseOutput converts Checkstyle JSON output to violations.
func (a *Adapter) ParseOutput(output *adapter.ToolOutput) ([]adapter.Violation, error) {
	return parseOutput(output)
}

// getJARPath returns the path to Checkstyle JAR.
func (a *Adapter) getJARPath() string {
	return filepath.Join(a.ToolsDir, fmt.Sprintf("checkstyle-%s-all.jar", DefaultVersion))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tempFile := destPath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(tempFile)
		return err
	}

	// Rename temp to final
	if err := os.Rename(tempFile, destPath); err != nil {
		os.Remove(tempFile)
		return err
	}

	return nil
}
