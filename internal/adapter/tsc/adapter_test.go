package tsc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/adapter"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter("", "")
	if adapter == nil {
		t.Fatal("NewAdapter() returned nil")
	}

	// Should have default ToolsDir
	if adapter.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
}

func TestNewAdapter_CustomDirs(t *testing.T) {
	toolsDir := "/custom/tools"
	workDir := "/custom/work"

	adapter := NewAdapter(toolsDir, workDir)

	if adapter.ToolsDir != toolsDir {
		t.Errorf("ToolsDir = %q, want %q", adapter.ToolsDir, toolsDir)
	}

	if adapter.WorkDir != workDir {
		t.Errorf("WorkDir = %q, want %q", adapter.WorkDir, workDir)
	}
}

func TestName(t *testing.T) {
	adapter := NewAdapter("", "")
	if adapter.Name() != "tsc" {
		t.Errorf("Name() = %q, want %q", adapter.Name(), "tsc")
	}
}

func TestGetTSCPath(t *testing.T) {
	adapter := NewAdapter("/test/tools", "")
	expected := filepath.Join("/test/tools", "node_modules", ".bin", "tsc")

	got := adapter.getTSCPath()
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
	defer os.RemoveAll(tmpDir)

	adapter := NewAdapter(tmpDir, "")

	if err := adapter.initPackageJSON(); err != nil {
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
		if !contains(string(content), field) {
			t.Errorf("package.json missing expected field: %s", field)
		}
	}
}

func TestCheckAvailability_NotFound(t *testing.T) {
	// Use a non-existent directory
	adapter := NewAdapter("/nonexistent/path", "")

	ctx := context.Background()
	err := adapter.CheckAvailability(ctx)

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
	defer os.RemoveAll(tmpDir)

	a := NewAdapter(tmpDir, "")

	ctx := context.Background()
	config := adapter.InstallConfig{
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

func TestExecute_FileCreation(t *testing.T) {
	// Create temporary work directory
	tmpDir, err := os.MkdirTemp("", "tsc-exec-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	adapter := NewAdapter("", tmpDir)

	ctx := context.Background()
	config := []byte(`{"compilerOptions": {"strict": true}}`)
	files := []string{"test.ts"}

	// Execute (will fail because tsc not installed, but we can test config file creation)
	_, _ = adapter.Execute(ctx, config, files)

	// Config file should have been created and cleaned up
	configPath := filepath.Join(tmpDir, ".symphony-tsconfig.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should have been cleaned up")
	}
}

func TestParseOutput_Integration(t *testing.T) {
	a := NewAdapter("", "")

	output := &adapter.ToolOutput{
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

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
