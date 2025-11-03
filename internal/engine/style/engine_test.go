package style

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

	if caps.Name != "style" {
		t.Errorf("Name = %s, want style", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "formatting") {
		t.Error("Expected formatting in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Style engine should not support autofix (removed by design)")
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
		Category: "style",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "style",
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

func TestValidate_WithInitialization(t *testing.T) {
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
		ID:       "TEST-RULE",
		Category: "style",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
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

func TestValidate_IndentRule(t *testing.T) {
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
		ID:       "TEST-INDENT",
		Category: "style",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 4,
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

func TestValidate_QuoteRule(t *testing.T) {
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
		ID:       "TEST-QUOTE",
		Category: "style",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"quote":  "double",
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

func TestValidate_SemiRule(t *testing.T) {
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
		ID:       "TEST-SEMI",
		Category: "style",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"semi":   true,
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
		ID:       "TEST-CUSTOM",
		Category: "style",
		Severity: "warning",
		Message:  "Custom style violation",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
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

func TestValidate_BraceStyleRule(t *testing.T) {
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
		ID:       "TEST-BRACE",
		Category: "style",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine":      "style",
			"brace_style": "1tbs",
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

func TestValidate_MultipleRules(t *testing.T) {
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
		ID:       "TEST-MULTI",
		Category: "style",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
			"semi":   true,
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
		if result.Engine != "style" {
			t.Errorf("Engine = %s, want style", result.Engine)
		}
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
			name: "javascript and typescript files",
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
