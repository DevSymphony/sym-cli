package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/length"
)

func TestLengthEngine_LineScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := length.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Max 80 characters per line
	rule := core.Rule{
		ID:       "FMT-LINE-80",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "line",
			"max":    80,
		},
		Message: "Line exceeds 80 characters",
	}

	badFile := filepath.Join("testdata", "javascript", "long-lines.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for long lines")
	}

	// Should find at least 2 long lines
	if len(result.Violations) < 2 {
		t.Errorf("Expected at least 2 violations, got %d", len(result.Violations))
	}
}

func TestLengthEngine_FileScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := length.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Max 50 lines per file
	rule := core.Rule{
		ID:       "FMT-FILE-50",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "file",
			"max":    50,
		},
		Message: "File exceeds 50 lines",
	}

	badFile := filepath.Join("testdata", "javascript", "long-file.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for long file")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations for long file")
	}
}

func TestLengthEngine_FunctionScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := length.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Max 30 lines per function
	rule := core.Rule{
		ID:       "FMT-FUNC-30",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "function",
			"max":    30,
		},
		Message: "Function exceeds 30 lines",
	}

	badFile := filepath.Join("testdata", "javascript", "long-function.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for long function")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations for long function")
	}
}

func TestLengthEngine_ParamsScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := length.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Max 4 parameters per function
	rule := core.Rule{
		ID:       "FMT-PARAMS-4",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "length",
			"scope":  "params",
			"max":    4,
		},
		Message: "Function has too many parameters (max 4)",
	}

	badFile := filepath.Join("testdata", "javascript", "many-params.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for too many params")
	}

	// Should find 2 functions with too many params
	if len(result.Violations) < 2 {
		t.Errorf("Expected at least 2 violations, got %d", len(result.Violations))
	}
}
