package eslint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter("", "")
	if adapter == nil {
		t.Fatal("NewAdapter() returned nil")
		return
	}

	if adapter.ToolsDir == "" {
		t.Error("ToolsDir should not be empty")
	}
}

func TestNewAdapter_CustomDirs(t *testing.T) {
	toolsDir := "/custom/tools"
	workDir := "/custom/work"

	a := NewAdapter(toolsDir, workDir)

	if a.ToolsDir != toolsDir {
		t.Errorf("ToolsDir = %q, want %q", a.ToolsDir, toolsDir)
	}

	if a.WorkDir != workDir {
		t.Errorf("WorkDir = %q, want %q", a.WorkDir, workDir)
	}
}

func TestName(t *testing.T) {
	a := NewAdapter("", "")
	if a.Name() != "eslint" {
		t.Errorf("Name() = %q, want %q", a.Name(), "eslint")
	}
}

func TestGetESLintPath(t *testing.T) {
	a := NewAdapter("/test/tools", "")
	expected := filepath.Join("/test/tools", "node_modules", ".bin", "eslint")

	got := a.getESLintPath()
	if got != expected {
		t.Errorf("getESLintPath() = %q, want %q", got, expected)
	}
}

func TestInitPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eslint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter(tmpDir, "")

	if err := a.initPackageJSON(); err != nil {
		t.Fatalf("initPackageJSON() error = %v", err)
	}

	packagePath := filepath.Join(tmpDir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Error("package.json was not created")
	}

	content, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	expectedFields := []string{`"name"`, `"symphony-tools"`}
	for _, field := range expectedFields {
		if !contains(string(content), field) {
			t.Errorf("package.json missing expected field: %s", field)
		}
	}
}

func TestCheckAvailability_NotFound(t *testing.T) {
	a := NewAdapter("/nonexistent/path", "")

	ctx := context.Background()
	err := a.CheckAvailability(ctx)

	if err == nil {
		t.Log("ESLint found globally, test skipped")
	}
}

func TestInstall(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eslint-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	a := NewAdapter(tmpDir, "")

	ctx := context.Background()
	config := adapter.InstallConfig{
		ToolsDir: tmpDir,
	}

	err = a.Install(ctx, config)
	if err != nil {
		t.Logf("Install failed (expected if npm unavailable): %v", err)
	}
}

func TestGenerateConfig(t *testing.T) {
	a := NewAdapter("", "")

	rule := &core.Rule{
		ID:       "TEST-RULE",
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z]",
		},
	}

	config, err := a.GenerateConfig(rule)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if len(config) == 0 {
		t.Error("GenerateConfig() returned empty config")
	}
}

func TestExecute_InvalidConfig(t *testing.T) {
	a := NewAdapter("", t.TempDir())

	ctx := context.Background()
	config := []byte(`{"rules": {}}`)
	files := []string{"test.js"}

	_, err := a.Execute(ctx, config, files)
	if err == nil {
		t.Log("Execute succeeded (ESLint may be available)")
	}
}

func TestParseOutput(t *testing.T) {
	a := NewAdapter("", "")

	output := &adapter.ToolOutput{
		Stdout: `[{"filePath":"test.js","messages":[{"ruleId":"no-unused-vars","severity":2,"message":"'x' is defined but never used","line":1,"column":5}]}]`,
		Stderr: "",
		ExitCode: 1,
	}

	violations, err := a.ParseOutput(output)
	if err != nil {
		t.Fatalf("ParseOutput() error = %v", err)
	}

	if len(violations) == 0 {
		t.Error("Expected violations to be parsed")
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"error", "error"},
		{"warning", "warn"},
		{"info", "off"},
		{"unknown", "error"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapSeverity(tt.input)
			if got != tt.want {
				t.Errorf("MapSeverity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMarshalConfig(t *testing.T) {
	config := map[string]interface{}{
		"rules": map[string]interface{}{
			"semi": []interface{}{2, "always"},
		},
	}

	data, err := MarshalConfig(config)
	if err != nil {
		t.Fatalf("MarshalConfig() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalConfig() returned empty data")
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
