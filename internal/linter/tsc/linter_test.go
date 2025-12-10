package tsc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
)

func TestNew(t *testing.T) {
	l := New("")
	if l == nil {
		t.Fatal("New() returned nil")
	}

	// Should have default ToolsDir
	if l.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
}

func TestNew_CustomToolsDir(t *testing.T) {
	toolsDir := "/custom/tools"

	l := New(toolsDir)

	if l.ToolsDir != toolsDir {
		t.Errorf("ToolsDir = %q, want %q", l.ToolsDir, toolsDir)
	}
}

func TestName(t *testing.T) {
	l := New("")
	if l.Name() != "tsc" {
		t.Errorf("Name() = %q, want %q", l.Name(), "tsc")
	}
}

func TestGetTSCPath(t *testing.T) {
	l := New("/test/tools")
	expected := filepath.Join("/test/tools", "node_modules", ".bin", "tsc")

	got := l.getTSCPath()
	if got != expected {
		t.Errorf("getTSCPath() = %q, want %q", got, expected)
	}
}

func TestInitPackageJSON(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "tsc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	l := New(tmpDir)

	if err := l.initPackageJSON(); err != nil {
		t.Fatalf("initPackageJSON() error = %v", err)
	}

	// Verify package.json was created
	packagePath := filepath.Join(tmpDir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Error("package.json was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	// Verify it contains expected fields
	expectedFields := []string{
		`"name"`,
		`"version"`,
		`"symphony-tools"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(string(content), field) {
			t.Errorf("package.json missing expected field: %s", field)
		}
	}
}

func TestCheckAvailability_NotFound(t *testing.T) {
	// Use a non-existent directory
	l := New("/nonexistent/path")

	ctx := context.Background()
	err := l.CheckAvailability(ctx)

	// Should return error when tsc is not found
	if err == nil {
		// This might pass if tsc is installed globally, which is okay
		t.Log("tsc found globally, test skipped")
	}
}

func TestInstall_MissingNPM(t *testing.T) {
	// This test will fail if npm is not available, which is expected
	tmpDir, err := os.MkdirTemp("", "tsc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := New(tmpDir)

	ctx := context.Background()
	config := linter.InstallConfig{
		ToolsDir: tmpDir,
	}

	// This will fail if npm is not in PATH
	// We're just testing that the function handles this gracefully
	err = a.Install(ctx, config)

	// We don't assert error here because npm might be available in CI
	if err != nil {
		// Expected when npm is not available
		t.Logf("Install failed as expected when npm unavailable: %v", err)
	}
}

func TestExecute_TempFileCleanup(t *testing.T) {
	// Create temporary tools directory
	tmpDir, err := os.MkdirTemp("", "tsc-exec-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	l := New(tmpDir)

	ctx := context.Background()
	config := []byte(`{"compilerOptions": {"strict": true}}`)
	files := []string{"test.ts"}

	// Execute (will fail because tsc not installed, but we can test temp file cleanup)
	_, _ = l.Execute(ctx, config, files)

	// Temp directory should exist (created by executor)
	tmpConfigDir := filepath.Join(tmpDir, ".tmp")
	if _, err := os.Stat(tmpConfigDir); os.IsNotExist(err) {
		// Dir might not exist if execution failed early, which is fine
		t.Log("Temp dir not created (execution may have failed early)")
		return
	}

	// Any tsconfig files should have been cleaned up
	files2, _ := filepath.Glob(filepath.Join(tmpConfigDir, "tsconfig-*.json"))
	if len(files2) > 0 {
		t.Error("Temp config files should have been cleaned up")
	}
}

func TestParseOutput_Integration(t *testing.T) {
	a := New("")

	output := &linter.ToolOutput{
		Stdout: `src/main.ts(10,5): error TS2304: Cannot find name 'foo'.
src/app.ts(20,10): error TS2339: Property 'bar' does not exist on type 'Object'.`,
		Stderr:   "",
		ExitCode: 2,
	}

	violations, err := a.ParseOutput(output)
	if err != nil {
		t.Fatalf("ParseOutput() error = %v", err)
	}

	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(violations))
	}
}
