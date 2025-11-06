package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/style"
)

func TestStyleEngine_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := style.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Prettier/ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: 2-space indentation, single quotes, semicolons
	rule := core.Rule{
		ID:       "STYLE-STANDARD",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
			"semi":   true,
		},
		Message: "Code style violations detected",
	}

	badFile := filepath.Join("testdata", "javascript", "bad-style.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for bad style")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations for style issues")
	}
}

func TestStyleEngine_GoodStyle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := style.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("Prettier/ESLint not available: %v", err)
	}
	defer engine.Close()

	rule := core.Rule{
		ID:       "STYLE-STANDARD",
		Category: "formatting",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine": "style",
			"indent": 2,
			"quote":  "single",
			"semi":   true,
		},
		Message: "Code style violations detected",
	}

	goodFile := filepath.Join("testdata", "javascript", "good-style.js")
	result, err := engine.Validate(ctx, rule, []string{goodFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))

	// Good file should have minimal or no violations
	if len(result.Violations) > 5 {
		t.Errorf("Expected few violations for good style file, got %d", len(result.Violations))
		for i, v := range result.Violations {
			t.Logf("Violation %d: %s", i+1, v.String())
		}
	}
}
