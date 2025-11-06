package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/engine/pattern"
)

func TestPatternEngine_BadNaming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create engine
	engine := pattern.NewEngine()

	// Init (this will try to install ESLint if not found)
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Define rule: class names must be PascalCase
	rule := core.Rule{
		ID:       "NAMING-CLASS-PASCAL",
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
		Message: "Class names must be PascalCase",
	}

	// Validate bad file
	badFile := filepath.Join("testdata", "javascript", "bad-naming.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: %+v", result)

	// Note: This test requires ESLint to be installed
	// If not installed, Init will try to install it
	// In CI, we should pre-install ESLint
}

func TestPatternEngine_GoodCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	rule := core.Rule{
		ID:       "NAMING-CLASS-PASCAL",
		Category: "naming",
		Severity: "error",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "identifier",
			"pattern": "^[A-Z][a-zA-Z0-9]*$",
		},
	}

	goodFile := filepath.Join("testdata", "javascript", "good-code.js")
	result, err := engine.Validate(ctx, rule, []string{goodFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Good file should pass (though ESLint id-match might still flag some things)
	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
}

func TestPatternEngine_ContentPattern_Secrets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Detect hardcoded secrets
	rule := core.Rule{
		ID:       "SEC-NO-HARDCODED-SECRETS",
		Category: "security",
		Severity: "error",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "content",
			"pattern": "(api[_-]?key|secret|password).*=.*[\"'][^\"']+[\"']",
		},
		Message: "Hardcoded secrets detected",
	}

	badFile := filepath.Join("testdata", "javascript", "hardcoded-secrets.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for hardcoded secrets")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violations for hardcoded secrets")
	}
}

func TestPatternEngine_ImportPattern_RestrictedModules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	engine := pattern.NewEngine()
	ctx := context.Background()
	config := core.EngineConfig{
		WorkDir: "../../tests",
		Debug:   true,
	}

	if err := engine.Init(ctx, config); err != nil {
		t.Skipf("ESLint not available: %v", err)
	}
	defer engine.Close()

	// Rule: Restrict lodash imports
	rule := core.Rule{
		ID:       "DEP-NO-LODASH",
		Category: "dependency",
		Severity: "warning",
		Check: map[string]interface{}{
			"engine":  "pattern",
			"target":  "import",
			"pattern": "lodash",
		},
		Message: "Lodash imports are restricted, use native alternatives",
	}

	badFile := filepath.Join("testdata", "javascript", "bad-imports.js")
	result, err := engine.Validate(ctx, rule, []string{badFile})

	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	t.Logf("Result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for i, v := range result.Violations {
		t.Logf("Violation %d: %s", i+1, v.String())
	}

	if result.Passed {
		t.Error("Expected validation to fail for lodash imports")
	}

	// Should find at least 2 lodash imports
	if len(result.Violations) < 2 {
		t.Errorf("Expected at least 2 violations, got %d", len(result.Violations))
	}
}
