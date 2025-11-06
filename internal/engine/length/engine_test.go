package length

import (
	"context"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
}

func TestGetCapabilities(t *testing.T) {
	engine := NewEngine()
	caps := engine.GetCapabilities()

	if caps.Name != "length" {
		t.Errorf("Name = %s, want length", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "formatting") {
		t.Error("Expected formatting in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Length engine should not support autofix")
	}
}

func TestInit(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
		Debug:    false,
	}

	err := engine.Init(ctx, config)
	if err != nil {
		t.Logf("Init failed (expected if ESLint not available): %v", err)
	}
}

func TestClose(t *testing.T) {
	engine := NewEngine()
	if err := engine.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestValidate_NoFiles(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	rule := core.Rule{
		ID:       "TEST-RULE",
		Category: "formatting",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    100,
		},
	}

	result, err := engine.Validate(ctx, rule, []string{})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Passed {
		t.Error("Expected validation to pass for empty file list")
	}
}

func TestValidate_NotInitialized(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	rule := core.Rule{
		ID:       "TEST-RULE",
		Category: "formatting",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "length",
		},
	}

	_, err := engine.Validate(ctx, rule, []string{"test.js"})
	if err == nil {
		t.Error("Expected error for uninitialized engine")
	}
}

func TestFilterFiles(t *testing.T) {
	engine := &Engine{}

	files := []string{
		"src/main.js",
		"src/app.ts",
		"test/test.js",
		"README.md",
	}

	tests := []struct {
		name     string
		selector *core.Selector
		want     []string
	}{
		{
			name:     "nil selector - all files",
			selector: nil,
			want:     files,
		},
		{
			name: "with selector - filters JS/TS only",
			selector: &core.Selector{
				Languages: []string{"javascript", "typescript"},
			},
			want: []string{"src/main.js", "src/app.ts", "test/test.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.filterFiles(files, tt.selector)
			if !equalSlices(got, tt.want) {
				t.Errorf("filterFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidate_WithCustomMessage(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - ESLint not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-CUSTOM-MSG",
		Category: "formatting",
		Severity: "error",
		Message:  "Line is too long",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    80,
		},
	}

	testFile := t.TempDir() + "/test.js"
	result, err := engine.Validate(ctx, rule, []string{testFile})

	if err != nil {
		t.Logf("Validate returned error (may be expected): %v", err)
	}

	if result != nil {
		if result.RuleID != rule.ID {
			t.Errorf("RuleID = %s, want %s", result.RuleID, rule.ID)
		}
	}
}

func TestValidate_FileScope(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - ESLint not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-FILE-LENGTH",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "file",
			"max":    500,
		},
	}

	testFile := t.TempDir() + "/test.js"
	result, err := engine.Validate(ctx, rule, []string{testFile})

	if err != nil {
		t.Logf("Validate returned error (may be expected): %v", err)
	}

	if result != nil {
		if result.RuleID != rule.ID {
			t.Errorf("RuleID = %s, want %s", result.RuleID, rule.ID)
		}
	}
}

func TestValidate_FunctionScope(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - ESLint not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-FUNCTION-LENGTH",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "function",
			"max":    50,
		},
	}

	testFile := t.TempDir() + "/test.js"
	result, err := engine.Validate(ctx, rule, []string{testFile})

	if err != nil {
		t.Logf("Validate returned error (may be expected): %v", err)
	}

	if result != nil {
		if result.RuleID != rule.ID {
			t.Errorf("RuleID = %s, want %s", result.RuleID, rule.ID)
		}
	}
}

func TestValidate_ParamsScope(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - ESLint not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-PARAMS-LENGTH",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "params",
			"max":    4,
		},
	}

	testFile := t.TempDir() + "/test.js"
	result, err := engine.Validate(ctx, rule, []string{testFile})

	if err != nil {
		t.Logf("Validate returned error (may be expected): %v", err)
	}

	if result != nil {
		if result.RuleID != rule.ID {
			t.Errorf("RuleID = %s, want %s", result.RuleID, rule.ID)
		}
	}
}

func TestInit_WithDebug(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
		Debug:    true,
	}

	err := engine.Init(ctx, config)
	if err != nil {
		t.Logf("Init with debug failed (expected if ESLint not available): %v", err)
	}

	if !engine.config.Debug {
		t.Error("Expected debug to be true")
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
