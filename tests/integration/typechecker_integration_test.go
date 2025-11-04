package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/typechecker"
)

func TestTypeChecker_TypeErrors_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := typechecker.NewEngine()
	ctx := context.Background()

	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: TypeScript not available: %v", err)
	}
	defer engine.Close()

	// Test rule: type checking with strict mode
	rule := core.Rule{
		ID:       "TYPE-CHECK-STRICT",
		Category: "type_safety",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"typescript"},
		},
		Check: map[string]interface{}{
			"engine": "typechecker",
			"strict": true,
		},
		Message: "TypeScript type errors detected",
	}

	files := []string{
		filepath.Join(workDir, "testdata/typescript/type-errors.ts"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for type errors")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected type errors to be detected")
	}

	t.Logf("Found %d type errors", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestTypeChecker_StrictModeErrors_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := typechecker.NewEngine()
	ctx := context.Background()

	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: TypeScript not available: %v", err)
	}
	defer engine.Close()

	// Test rule: strict mode violations
	rule := core.Rule{
		ID:       "TYPE-STRICT-MODE",
		Category: "type_safety",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"typescript"},
		},
		Check: map[string]interface{}{
			"engine":         "typechecker",
			"strict":         true,
			"noImplicitAny":  true,
			"strictNullChecks": true,
		},
		Message: "Strict mode violations detected",
	}

	files := []string{
		filepath.Join(workDir, "testdata/typescript/strict-mode-errors.ts"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for strict mode violations")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected strict mode violations to be detected")
	}

	t.Logf("Found %d strict mode violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestTypeChecker_ValidFile_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := typechecker.NewEngine()
	ctx := context.Background()

	workDir := getTestdataDir(t)
	config := core.EngineConfig{
		ToolsDir: getToolsDir(t),
		WorkDir:  workDir,
		Debug:    true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Skipping test: TypeScript not available: %v", err)
	}
	defer engine.Close()

	// Test rule: type checking
	rule := core.Rule{
		ID:       "TYPE-CHECK",
		Category: "type_safety",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"typescript"},
		},
		Check: map[string]interface{}{
			"engine": "typechecker",
			"strict": true,
		},
	}

	files := []string{
		filepath.Join(workDir, "testdata/typescript/valid.ts"),
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
