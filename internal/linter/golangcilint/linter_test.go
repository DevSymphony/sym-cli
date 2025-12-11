package golangcilint

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	l := New("/custom/tools")
	assert.NotNil(t, l)
	assert.Equal(t, "/custom/tools", l.ToolsDir)
	assert.NotNil(t, l.executor)
}

func TestNew_DefaultToolsDir(t *testing.T) {
	l := New("")
	assert.NotNil(t, l)
	assert.NotEmpty(t, l.ToolsDir)
	assert.Contains(t, l.ToolsDir, ".sym")
}

func TestName(t *testing.T) {
	l := New("")
	assert.Equal(t, "golangci-lint", l.Name())
}

func TestGetCapabilities(t *testing.T) {
	l := New("")
	caps := l.GetCapabilities()

	assert.Equal(t, "golangci-lint", caps.Name)
	assert.Equal(t, []string{"go"}, caps.SupportedLanguages)
	assert.Equal(t, DefaultVersion, caps.Version)

	expectedCategories := []string{
		"bugs", "style", "performance", "complexity",
		"error_handling", "security", "unused", "naming", "ast",
	}
	assert.Equal(t, expectedCategories, caps.SupportedCategories)
}

func TestGetGolangciLintPath(t *testing.T) {
	l := New("/test/tools")

	path := l.getGolangciLintPath()

	expectedDir := filepath.Join("/test/tools", "golangci-lint-"+DefaultVersion)
	expectedBin := "golangci-lint"
	if runtime.GOOS == "windows" {
		expectedBin = "golangci-lint.exe"
	}
	expectedPath := filepath.Join(expectedDir, expectedBin)

	assert.Equal(t, expectedPath, path)
}

func TestGetGolangciLintPath_CustomPath(t *testing.T) {
	l := New("/test/tools")
	l.GolangciLintPath = "/custom/path/golangci-lint"

	path := l.getGolangciLintPath()
	assert.Equal(t, "/custom/path/golangci-lint", path)
}

func TestGetDownloadURL_Linux_AMD64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping Linux AMD64 test on different platform")
	}

	l := New("")
	url, ext, err := l.getDownloadURL("2.7.2")

	require.NoError(t, err)
	assert.Equal(t, "tar.gz", ext)
	assert.Contains(t, url, "golangci-lint-2.7.2-linux-amd64.tar.gz")
	assert.Contains(t, url, GitHubReleaseURL)
}

func TestGetDownloadURL_Darwin_ARM64(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Skipping macOS ARM64 test on different platform")
	}

	l := New("")
	url, ext, err := l.getDownloadURL("2.7.2")

	require.NoError(t, err)
	assert.Equal(t, "tar.gz", ext)
	assert.Contains(t, url, "golangci-lint-2.7.2-darwin-arm64.tar.gz")
	assert.Contains(t, url, GitHubReleaseURL)
}

func TestGetDownloadURL_Windows_AMD64(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping Windows AMD64 test on different platform")
	}

	l := New("")
	url, ext, err := l.getDownloadURL("2.7.2")

	require.NoError(t, err)
	assert.Equal(t, "zip", ext)
	assert.Contains(t, url, "golangci-lint-2.7.2-windows-amd64.zip")
	assert.Contains(t, url, GitHubReleaseURL)
}

func TestGetDownloadURL_ValidVersion(t *testing.T) {
	l := New("")
	url, ext, err := l.getDownloadURL("2.5.0")

	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.NotEmpty(t, ext)
	assert.Contains(t, url, "v2.5.0")
	assert.Contains(t, url, "golangci-lint-2.5.0")
}

func TestCheckAvailability_NotInstalled(t *testing.T) {
	l := New("/nonexistent/path")

	err := l.CheckAvailability(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFilterGoFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "only go files",
			input:    []string{"main.go", "util.go", "test.go"},
			expected: []string{"main.go", "util.go", "test.go"},
		},
		{
			name:     "mixed files",
			input:    []string{"main.go", "README.md", "util.go", "config.json"},
			expected: []string{"main.go", "util.go"},
		},
		{
			name:     "no go files",
			input:    []string{"README.md", "config.json", "test.txt"},
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "paths with directories",
			input:    []string{"src/main.go", "docs/README.md", "pkg/util.go"},
			expected: []string{"src/main.go", "pkg/util.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterGoFiles(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompileTimeInterfaceCheck(t *testing.T) {
	var _ linter.Linter = (*Linter)(nil)
	// If this compiles, the interface is correctly implemented
}
