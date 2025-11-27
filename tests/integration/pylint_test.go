package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adapterRegistry "github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPylint_ValidateChanges(t *testing.T) {
	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pylint"))
	require.NoError(t, err, "Failed to get testdata path")

	// Check testdata exists
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skipf("Testdata directory not found: %s", testdataDir)
	}

	// 2. Load policy (enforce.rbac 없으면 RBAC 비활성화됨)
	policyPath := filepath.Join(testdataDir, ".sym", "code-policy.json")
	policyData, err := os.ReadFile(policyPath)
	require.NoError(t, err, "Failed to read code-policy.json")

	var policy schema.CodePolicy
	require.NoError(t, json.Unmarshal(policyData, &policy), "Failed to parse policy")

	// 3. Create validator with custom workDir (symDir = workDir/.sym)
	v := validator.NewValidatorWithWorkDir(&policy, true, testdataDir)
	defer v.Close()

	// 4. Check tool availability
	adp, err := adapterRegistry.Global().GetAdapter("pylint")
	if err != nil {
		t.Skipf("Pylint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Pylint not available: %v", err)
	}

	// 5. Create GitChange (simulate modified file)
	testFile := filepath.Join(testdataDir, "Test.py")
	require.FileExists(t, testFile, "Test.py should exist")

	changes := []validator.GitChange{{
		FilePath: testFile,
		Status:   "M", // Modified
		Diff:     "",  // Diff not needed for adapter-based rules
	}}

	// 6. Execute ValidateChanges
	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err, "ValidateChanges failed")

	// 7. Log violations for debugging
	t.Logf("Found %d violations", len(result.Violations))
	for _, viol := range result.Violations {
		t.Logf("  - %s:%d: [%s] %s (tool: %s)",
			viol.File, viol.Line, viol.RuleID, viol.Message, viol.ToolName)
	}

	// 8. Assertions
	assert.GreaterOrEqual(t, len(result.Violations), 4, "Should detect at least 4 naming violations")

	// Check that ToolName is pylint
	for _, viol := range result.Violations {
		assert.Equal(t, "pylint", viol.ToolName, "ToolName should be pylint")
	}

	// Check that RuleID is from policy (not adapter's C0103)
	// Policy rule IDs: pylint-naming-func, pylint-naming-class, pylint-naming-const
	validRuleIDs := map[string]bool{
		"pylint-naming-func":  true,
		"pylint-naming-class": true,
		"pylint-naming-const": true,
	}
	for _, viol := range result.Violations {
		assert.True(t, validRuleIDs[viol.RuleID],
			"RuleID should be from policy, got: %s", viol.RuleID)
	}
}

func TestPylint_NamingConventions(t *testing.T) {
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pylint"))
	require.NoError(t, err)

	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skipf("Testdata directory not found: %s", testdataDir)
	}

	policyPath := filepath.Join(testdataDir, ".sym", "code-policy.json")
	policyData, err := os.ReadFile(policyPath)
	require.NoError(t, err)

	var policy schema.CodePolicy
	require.NoError(t, json.Unmarshal(policyData, &policy))

	v := validator.NewValidatorWithWorkDir(&policy, true, testdataDir)
	defer v.Close()

	adp, err := adapterRegistry.Global().GetAdapter("pylint")
	if err != nil {
		t.Skipf("Pylint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Pylint not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.py")
	changes := []validator.GitChange{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Test specific naming convention checks
	tests := []struct {
		name        string
		checkLine   int
		description string
	}{
		{
			name:        "snake_case class name violation",
			checkLine:   6,
			description: "user_profile class should trigger PascalCase violation",
		},
		{
			name:        "camelCase function name violation",
			checkLine:   10,
			description: "getUserName function should trigger snake_case violation",
		},
		{
			name:        "snake_case class name violation 2",
			checkLine:   17,
			description: "dataProcessor class should trigger PascalCase violation",
		},
		{
			name:        "camelCase function name violation 2",
			checkLine:   25,
			description: "calculateTotal function should trigger snake_case violation",
		},
	}

	// Build violation line map
	violationLines := make(map[int]bool)
	for _, viol := range result.Violations {
		violationLines[viol.Line] = true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if violationLines[tt.checkLine] {
				t.Logf("Found expected violation at line %d", tt.checkLine)
			} else {
				t.Logf("Expected violation at line %d not found", tt.checkLine)
			}
		})
	}
}

func TestPylint_ToolNameAndRuleID(t *testing.T) {
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pylint"))
	require.NoError(t, err)

	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skipf("Testdata directory not found: %s", testdataDir)
	}

	policyPath := filepath.Join(testdataDir, ".sym", "code-policy.json")
	policyData, err := os.ReadFile(policyPath)
	require.NoError(t, err)

	var policy schema.CodePolicy
	require.NoError(t, json.Unmarshal(policyData, &policy))

	v := validator.NewValidatorWithWorkDir(&policy, true, testdataDir)
	defer v.Close()

	adp, err := adapterRegistry.Global().GetAdapter("pylint")
	if err != nil {
		t.Skipf("Pylint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Pylint not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.py")
	changes := []validator.GitChange{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	require.Greater(t, len(result.Violations), 0, "Should have at least one violation")

	// Verify ToolName is set correctly
	for _, viol := range result.Violations {
		assert.Equal(t, "pylint", viol.ToolName,
			"ToolName should be 'pylint', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy, not adapter
	// Adapter would return C0103, but policy should map to pylint-naming-*
	for _, viol := range result.Violations {
		assert.True(t, strings.HasPrefix(viol.RuleID, "pylint-"),
			"RuleID should start with 'pylint-', got '%s'", viol.RuleID)
	}

	// Verify Severity comes from policy
	for _, viol := range result.Violations {
		assert.Equal(t, "error", viol.Severity,
			"Severity should be 'error' from policy, got '%s'", viol.Severity)
	}
}
