package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/length"
)

func TestLengthEngine_LineLengthViolations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := length.NewEngine()
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

	// Test rule: max line length 100
	rule := core.Rule{
		ID:       "FMT-LINE-100",
		Category: "formatting",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    100,
		},
		Message: "Line length must not exceed 100 characters",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/length-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for line length violations")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations to be detected")
	}

	t.Logf("Found %d line length violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestLengthEngine_MaxParams_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := length.NewEngine()
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

	// Test rule: max 4 parameters
	rule := core.Rule{
		ID:       "FUNC-MAX-PARAMS",
		Category: "formatting",
		Severity: "warning",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "params",
			"max":    4,
		},
		Message: "Functions should have at most 4 parameters",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/length-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if result.Passed {
		t.Error("Expected validation to fail for too many parameters")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations to be detected")
	}

	t.Logf("Found %d parameter violations", len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestLengthEngine_ValidFile_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := length.NewEngine()
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

	// Test rule: max line length 100
	rule := core.Rule{
		ID:       "FMT-LINE-100",
		Category: "formatting",
		Severity: "error",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    100,
		},
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/valid.js"),
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
