package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/ast"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestASTEngine_CallExpression_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := ast.NewEngine()
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
	defer engine.Close()

	// Test rule: detect console.log calls
	rule := core.Rule{
		ID:       "AST-NO-CONSOLE-LOG",
		Category: "custom",
		Severity: "warning",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":   "ast",
			"language": "javascript",
			"node":     "CallExpression",
			"where": map[string]interface{}{
				"callee.object.name": "console",
				"callee.property.name": "log",
			},
		},
		Message: "Avoid using console.log in production code",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/valid.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	t.Logf("AST validation result: passed=%v, violations=%d", result.Passed, len(result.Violations))
	for _, v := range result.Violations {
		t.Logf("  %s", v.String())
	}
}

func TestASTEngine_ClassDeclaration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := ast.NewEngine()
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
	defer engine.Close()

	// Test rule: detect class declarations
	rule := core.Rule{
		ID:       "AST-CLASS-EXISTS",
		Category: "custom",
		Severity: "info",
		When: &core.Selector{
			Languages: []string{"javascript"},
		},
		Check: map[string]interface{}{
			"engine":   "ast",
			"language": "javascript",
			"node":     "ClassDeclaration",
		},
		Message: "Class declaration found",
	}

	files := []string{
		filepath.Join(workDir, "testdata/javascript/valid.js"),
		filepath.Join(workDir, "testdata/javascript/naming-violations.js"),
	}

	result, err := engine.Validate(ctx, rule, files)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Should find class declarations in both files
	t.Logf("Found %d class declarations", len(result.Violations))
}
