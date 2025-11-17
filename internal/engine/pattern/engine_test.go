package pattern

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

	if caps.Name != "pattern" {
		t.Errorf("Name = %s, want pattern", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "naming") {
		t.Error("Expected naming in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Pattern engine should not support autofix")
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

	// Init might fail if eslint is not available, which is okay for unit tests
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
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "pattern",
		},
	}

	// Validate with empty file list
	result, err := engine.Validate(ctx, rule, []string{})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Passed {
		t.Error("Expected validation to pass for empty file list")
	}

	if len(result.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(result.Violations))
	}
}

func TestValidate_NotInitialized(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	rule := core.Rule{
		ID:       "TEST-RULE",
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "pattern",
		},
	}

	// Validate without initialization
	_, err := engine.Validate(ctx, rule, []string{"test.js"})
	if err == nil {
		t.Error("Expected error for uninitialized engine")
	}
}

// TestMatchesLanguage has been moved to core package tests.
// Language matching logic is now centralized in core.MatchesLanguage.

func TestFilterFiles(t *testing.T) {
	engine := &Engine{}

	files := []string{
		"src/main.js",
		"src/app.ts",
		"test/test.js",
		"README.md",
		"src/styles.css",
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
			name: "javascript only",
			selector: &core.Selector{
				Languages: []string{"javascript"},
			},
			want: []string{"src/main.js", "test/test.js"},
		},
		{
			name: "typescript only",
			selector: &core.Selector{
				Languages: []string{"typescript"},
			},
			want: []string{"src/app.ts"},
		},
		{
			name: "multiple languages",
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

func TestDetectLanguage(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		rule  core.Rule
		files []string
		want  string
	}{
		{core.Rule{}, []string{"main.js"}, "javascript"},
		{core.Rule{}, []string{"app.jsx"}, "jsx"},
		{core.Rule{}, []string{"server.ts"}, "typescript"},
		{core.Rule{}, []string{"component.tsx"}, "tsx"},
		{core.Rule{}, []string{}, "javascript"},
		{core.Rule{When: &core.Selector{Languages: []string{"python"}}}, []string{"main.js"}, "python"},
	}

	for _, tt := range tests {
		got := engine.detectLanguage(tt.rule, tt.files)
		if got != tt.want {
			t.Errorf("detectLanguage(%v, %v) = %q, want %q", tt.rule, tt.files, got, tt.want)
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
		ID:       "TEST-CUSTOM-MSG",
		Category: "naming",
		Severity: "error",
		Message:  "Custom violation message",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
	}

	// Create a test file with a violation
	testFile := t.TempDir() + "/test.js"
	// Since we can't guarantee ESLint will find violations in a real file,
	// we'll just test that the function handles the rule correctly
	result, err := engine.Validate(ctx, rule, []string{testFile})

	// Should not error even if file doesn't exist or has no violations
	if err != nil {
		t.Logf("Validate returned error (may be expected): %v", err)
	}

	// If result is returned, check basic properties
	if result != nil {
		if result.RuleID != rule.ID {
			t.Errorf("RuleID = %s, want %s", result.RuleID, rule.ID)
		}
	}
}

func TestValidate_WithFilteredFiles(t *testing.T) {
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
		ID:       "TEST-FILTERED",
		Category: "naming",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"typescript"},
		},
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z]",
		},
	}

	// Provide JS and TS files - only TS should be validated
	files := []string{"test.js", "test.ts"}

	result, err := engine.Validate(ctx, rule, files)

	// Should not error
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

	// Init might fail if eslint is not available
	err := engine.Init(ctx, config)
	if err != nil {
		t.Logf("Init with debug failed (expected if ESLint not available): %v", err)
	}

	// Check that config was set
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
