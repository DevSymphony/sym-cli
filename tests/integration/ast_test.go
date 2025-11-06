package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/DevSymphony/sym-cli/internal/engine/ast"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestASTEngine_AsyncWithTry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := ast.NewEngine()

	config := core.EngineConfig{
		WorkDir: "../../tests",
		Timeout: 30 * time.Second,
		Debug:   true,
	}

	ctx := context.Background()
	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint init failed: %v", err)
	}
	defer engine.Close()

	// Rule: Async functions must have try-catch
	rule := core.Rule{
		ID:       "AST-ASYNC-TRY",
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "FunctionDeclaration",
			"where": map[string]interface{}{
				"async": true,
			},
			"has": []interface{}{"TryStatement"},
		},
		Message: "Async functions must use try-catch for error handling",
	}

	// Test bad file (async without try-catch)
	badFile := filepath.Join("testdata", "javascript", "async-without-try.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	t.Logf("Bad file result: passed=%v, violations=%d", result.Passed, len(result.Violations))

	if result.Passed {
		t.Error("Expected validation to fail for async functions without try-catch")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations for async functions without try-catch")
	}

	// Should find 3 violations (3 async functions without try-catch)
	if len(result.Violations) < 3 {
		t.Errorf("Expected at least 3 violations, got %d", len(result.Violations))
	}

	// Verify violation details
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
		if v.Severity != "error" {
			t.Errorf("Violation severity = %s, want error", v.Severity)
		}
	}
}

func TestASTEngine_AsyncWithTryGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := ast.NewEngine()

	config := core.EngineConfig{
		WorkDir: "../../tests",
		Timeout: 30 * time.Second,
		Debug:   true,
	}

	ctx := context.Background()
	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint init failed: %v", err)
	}
	defer engine.Close()

	rule := core.Rule{
		ID:       "AST-ASYNC-TRY",
		Category: "error_handling",
		Severity: "error",
		Check: map[string]interface{}{
			"engine": "ast",
			"node":   "FunctionDeclaration",
			"where": map[string]interface{}{
				"async": true,
			},
			"has": []interface{}{"TryStatement"},
		},
		Message: "Async functions must use try-catch for error handling",
	}

	// Test good file (async with try-catch)
	goodFile := filepath.Join("testdata", "javascript", "async-with-try.js")
	result, err := engine.Validate(ctx, rule, []string{goodFile})
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	t.Logf("Good file result: passed=%v, violations=%d", result.Passed, len(result.Violations))

	if !result.Passed {
		t.Errorf("Expected validation to pass for async functions with try-catch")
		for i, v := range result.Violations {
			t.Logf("Unexpected violation %d: %s", i+1, v.String())
		}
	}

	if len(result.Violations) > 0 {
		t.Errorf("Expected no violations, got %d", len(result.Violations))
	}
}
