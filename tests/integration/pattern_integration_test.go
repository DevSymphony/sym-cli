package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/pattern"
)

func TestPatternEngine_NamingViolations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()

	// Initialize engine
	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: ESLint not available: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test rule: class names must be PascalCase
	rule := core.Rule{
		ID:       "NAMING-CLASS-PASCAL",
		Category: "naming",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
		Message: "Class names must be PascalCase",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/pattern/naming-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for naming violations")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations to be detected")
	}

	t.Logf("Found %d violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestPatternEngine_SecurityViolations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()

	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: ESLint not available: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test rule: no hardcoded secrets
	rule := core.Rule{
		ID:       "SEC-NO-SECRETS",
		Category: "security",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "content",
			"pattern": "(api[_-]?key|password|secret|token)\\s*=\\s*['\"][^'\"]+['\"]",
			"flags":   "i",
		},
		Message: "No hardcoded secrets allowed",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/pattern/security-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for security violations")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations to be detected")
	}

	t.Logf("Found %d security violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestPatternEngine_ValidFile_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()

	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: ESLint not available: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test rule: class names must be PascalCase
	rule := core.Rule{
		ID:       "NAMING-CLASS-PASCAL",
		Category: "naming",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/pattern/valid.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Passed {
		t.Errorf("Expected validation to pass for valid file, got %d violations", len(result.Violations))
		for _, v := range result.Violations {
			t.Logf("  %s", v.String())
		}
	}
}
