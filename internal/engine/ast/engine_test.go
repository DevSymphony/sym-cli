package ast

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

	if caps.Name != "ast" {
		t.Errorf("Name = %s, want ast", caps.Name)
	}

	if !contains(caps.SupportedLanguages, "javascript") {
		t.Error("Expected javascript in supported languages")
	}

	if !contains(caps.SupportedCategories, "error_handling") {
		t.Error("Expected error_handling in supported categories")
	}

	if caps.SupportsAutofix {
		t.Error("AST engine should not support autofix")
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
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "CallExpression",
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
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "CallExpression",
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

func TestValidate_WithWhereClause(t *testing.T) {
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
		ID:       "TEST-WHERE",
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "CallExpression",
			"where": map[string]interface{}{
				"func": map[string]interface{}{
					"in": []string{"open", "readFile"},
				},
			},
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

func TestValidate_WithHasClause(t *testing.T) {
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
		ID:       "TEST-HAS",
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "CallExpression",
			"has":    []string{"TryStatement"},
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

func TestValidate_WithNotHasClause(t *testing.T) {
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
		ID:       "TEST-NOT-HAS",
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "FunctionDeclaration",
			"notHas": []string{"JSDocComment"},
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
		Category: "error_handling",
		Severity: "error",
		Message:  "Custom AST violation",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "CallExpression",
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

// TestMatchesSelector has been moved to core package tests.
// File filtering logic is now centralized in core.FilterFiles.

// TestMatchesLanguage has been moved to core package tests.
// Language matching logic is now centralized in core.MatchesLanguage.

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
