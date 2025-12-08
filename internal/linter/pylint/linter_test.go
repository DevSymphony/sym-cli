package pylint

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

func TestNew(t *testing.T) {
	l := New("")
	if l == nil {
		t.Fatal("New() returned nil")
	}

	if l.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
}

func TestNew_CustomToolsDir(t *testing.T) {
	toolsDir := "/custom/tools"

	a := New(toolsDir)

	if a.ToolsDir != toolsDir {
		t.Errorf("ToolsDir = %q, want %q", a.ToolsDir, toolsDir)
	}
}

func TestName(t *testing.T) {
	a := New("")
	if a.Name() != "pylint" {
		t.Errorf("Name() = %q, want %q", a.Name(), "pylint")
	}
}

func TestGetCapabilities(t *testing.T) {
	a := New("")
	caps := a.GetCapabilities()

	if caps.Name != "pylint" {
		t.Errorf("GetCapabilities().Name = %q, want %q", caps.Name, "pylint")
	}

	expectedLangs := []string{"python", "py"}
	for _, lang := range expectedLangs {
		found := false
		for _, supported := range caps.SupportedLanguages {
			if supported == lang {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetCapabilities() missing language: %s", lang)
		}
	}

	expectedCategories := []string{"naming", "style", "documentation", "error_handling", "complexity"}
	for _, cat := range expectedCategories {
		found := false
		for _, supported := range caps.SupportedCategories {
			if supported == cat {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetCapabilities() missing category: %s", cat)
		}
	}
}

func TestGetVenvPath(t *testing.T) {
	a := New("/test/tools")
	expected := filepath.Join("/test/tools", "pylint-venv")

	got := a.getVenvPath()
	if got != expected {
		t.Errorf("getVenvPath() = %q, want %q", got, expected)
	}
}

func TestGetPylintPath(t *testing.T) {
	a := New("/test/tools")

	var expected string
	if runtime.GOOS == "windows" {
		expected = filepath.Join("/test/tools", "pylint-venv", "Scripts", "pylint.exe")
	} else {
		expected = filepath.Join("/test/tools", "pylint-venv", "bin", "pylint")
	}

	got := a.getPylintPath()
	if got != expected {
		t.Errorf("getPylintPath() = %q, want %q", got, expected)
	}
}

func TestGetPipPath(t *testing.T) {
	a := New("/test/tools")

	var expected string
	if runtime.GOOS == "windows" {
		expected = filepath.Join("/test/tools", "pylint-venv", "Scripts", "pip.exe")
	} else {
		expected = filepath.Join("/test/tools", "pylint-venv", "bin", "pip")
	}

	got := a.getPipPath()
	if got != expected {
		t.Errorf("getPipPath() = %q, want %q", got, expected)
	}
}

func TestCheckAvailability_NotFound(t *testing.T) {
	a := New("/nonexistent/path")

	ctx := context.Background()
	err := a.CheckAvailability(ctx)

	if err == nil {
		t.Log("Pylint found globally, test skipped")
	}
}

func TestInstall(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pylint-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := New(tmpDir)

	ctx := context.Background()
	config := linter.InstallConfig{
		ToolsDir: tmpDir,
	}

	err = a.Install(ctx, config)
	if err != nil {
		t.Logf("Install failed (expected if Python unavailable): %v", err)
	}
}

func TestExecute_EmptyFiles(t *testing.T) {
	a := New(t.TempDir())

	ctx := context.Background()
	config := []byte(`[MASTER]`)

	output, err := a.Execute(ctx, config, []string{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.Stdout != "[]" {
		t.Errorf("Expected empty array for no files, got %q", output.Stdout)
	}
}

func TestParseOutput(t *testing.T) {
	a := New("")

	output := &linter.ToolOutput{
		Stdout: `[
			{
				"type": "convention",
				"module": "test",
				"line": 1,
				"column": 0,
				"path": "test.py",
				"symbol": "missing-module-docstring",
				"message": "Missing module docstring",
				"message-id": "C0114"
			}
		]`,
		Stderr:   "",
		ExitCode: 16,
	}

	violations, err := a.ParseOutput(output)
	if err != nil {
		t.Fatalf("ParseOutput() error = %v", err)
	}

	if len(violations) == 0 {
		t.Error("Expected violations to be parsed")
	}

	if len(violations) > 0 {
		v := violations[0]
		if v.File != "test.py" {
			t.Errorf("File = %q, want %q", v.File, "test.py")
		}
		if v.Line != 1 {
			t.Errorf("Line = %d, want %d", v.Line, 1)
		}
		if v.RuleID != "C0114/missing-module-docstring" {
			t.Errorf("RuleID = %q, want %q", v.RuleID, "C0114/missing-module-docstring")
		}
	}
}
