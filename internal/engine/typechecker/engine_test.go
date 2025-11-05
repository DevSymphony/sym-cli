package typechecker

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

	if caps.Name != "typechecker" {
		t.Errorf("Name = %s, want typechecker", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "typescript") {
		t.Error("Expected typescript in supported languages")
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "type_safety") {
		t.Error("Expected type_safety in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("Type checker engine should not support autofix")
	}

	// Verify external tools
	if len(caps.ExternalTools) == 0 {
		t.Error("Expected external tools to be listed")
	}

	foundTSC := false
	for _, tool := range caps.ExternalTools {
		if tool.Name == "typescript" {
			foundTSC = true
			if tool.Optional {
				t.Error("TypeScript should not be optional")
			}
		}
	}

	if !foundTSC {
		t.Error("Expected typescript in external tools")
	}
}

func TestDetectLanguage(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "typescript file",
			files: []string{"src/main.ts"},
			want:  "typescript",
		},
		{
			name:  "javascript file",
			files: []string{"src/main.js"},
			want:  "javascript",
		},
		{
			name:  "jsx file",
			files: []string{"src/Component.jsx"},
			want:  "jsx",
		},
		{
			name:  "tsx file",
			files: []string{"src/Component.tsx"},
			want:  "tsx",
		},
		{
			name:  "empty files",
			files: []string{},
			want:  "typescript", // default
		},
		{
			name:  "multiple files - first determines language",
			files: []string{"src/main.js", "src/app.ts"},
			want:  "javascript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.detectLanguage(tt.files)
			if got != tt.want {
				t.Errorf("detectLanguage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClose(t *testing.T) {
	engine := NewEngine()
	if err := engine.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
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
		t.Logf("Init failed (expected if TSC not available): %v", err)
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
		t.Logf("Init with debug failed (expected if TSC not available): %v", err)
	}

	if !engine.config.Debug {
		t.Error("Expected debug to be true")
	}
}

func TestValidate_NoFiles(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	rule := core.Rule{
		ID:       "TEST-RULE",
		Category: "type_safety",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "typechecker",
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
		t.Skipf("Skipping test - TSC not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-RULE",
		Category: "type_safety",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "typechecker",
			"strict": true,
		},
	}

	testFile := t.TempDir() + "/test.ts"
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

func TestValidate_WithStrictMode(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - TSC not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-STRICT",
		Category: "type_safety",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "typechecker",
			"strict": true,
		},
	}

	testFile := t.TempDir() + "/test.ts"
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
		t.Skipf("Skipping test - TSC not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-CUSTOM",
		Category: "type_safety",
		Severity: "error",
		Message:  "Custom type error",
		Check: map[string]interface{}{
			"engine": "typechecker",
		},
	}

	testFile := t.TempDir() + "/test.ts"
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

func TestValidate_JavaScriptFile(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	config := core.EngineConfig{
		ToolsDir: t.TempDir(),
		WorkDir:  t.TempDir(),
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test - TSC not available: %v", err)
	}

	rule := core.Rule{
		ID:       "TEST-JS",
		Category: "type_safety",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "typechecker",
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

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Integration-like test (but mocked to avoid requiring tsc installation)
func TestValidate_FileFiltering(t *testing.T) {
	files := []string{
		"src/main.ts",
		"src/util.js",
		"test/main_test.ts",
		"README.md",
	}

	rule := core.Rule{
		ID:       "TYPE-CHECK",
		Category: "type_safety",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"typescript"},
			Exclude:   []string{"**/*_test.ts"},
		},
		Check: map[string]interface{}{
			"engine": "typechecker",
		},
	}

	// Filter files using the selector (same logic as in engine)
	filtered := core.FilterFiles(files, rule.When)

	// Should only include .ts files, excluding test files
	expected := []string{"src/main.ts"}

	if len(filtered) != len(expected) {
		t.Errorf("Filtered %d files, want %d", len(filtered), len(expected))
		t.Errorf("Got: %v", filtered)
		t.Errorf("Want: %v", expected)
	}

	for i, file := range filtered {
		if i < len(expected) && file != expected[i] {
			t.Errorf("Filtered[%d] = %q, want %q", i, file, expected[i])
		}
	}
}
