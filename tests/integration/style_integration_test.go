package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/style"
)

func TestStyleEngine_IndentViolations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := style.NewEngine()
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

	// Test rule: indent with 2 spaces
	rule := core.Rule{
		ID:       "STYLE-INDENT-2",
		Category: "style",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":  "style",
			"indent":  2,
			"quote":   "single",
			"semi":    true,
		},
		Message: "Code must use 2-space indentation",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/style/style-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for style violations")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations to be detected")
	}

	t.Logf("Found %d style violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestStyleEngine_ValidFile_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := style.NewEngine()
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

	// Test rule: indent with 2 spaces
	rule := core.Rule{
		ID:       "STYLE-INDENT-2",
		Category: "style",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
			"semi":   true,
		},
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/style/valid.js"),
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
