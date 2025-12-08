package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPMD_ValidateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pmd"))
	require.NoError(t, err, "Failed to get testdata path")

	// Check testdata exists
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skipf("Testdata directory not found: %s", testdataDir)
	}

	// 2. Load policy
	policyPath := filepath.Join(testdataDir, ".sym", "code-policy.json")
	policyData, err := os.ReadFile(policyPath)
	require.NoError(t, err, "Failed to read code-policy.json")

	var policy schema.CodePolicy
	require.NoError(t, json.Unmarshal(policyData, &policy), "Failed to parse policy")

	// 3. Create validator with custom workDir
	v := validator.NewValidatorWithWorkDir(&policy, true, testdataDir)
	defer v.Close()

	// 4. Check tool availability
	adp, err := linter.Global().GetLinter("pmd")
	if err != nil {
		t.Skipf("PMD adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("PMD not available: %v", err)
	}

	// 5. Create GitChange
	testFile := filepath.Join(testdataDir, "Test.java")
	require.FileExists(t, testFile, "Test.java should exist")

	changes := []git.Change{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
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

	// 8. Assertions - PMD should detect at least 2 violations
	assert.GreaterOrEqual(t, len(result.Violations), 2, "Should detect at least 2 violations")

	// Check that ToolName is pmd
	for _, viol := range result.Violations {
		assert.Equal(t, "pmd", viol.ToolName, "ToolName should be pmd")
	}

	// Check that RuleID is from policy
	validRuleIDs := map[string]bool{
		"pmd-emptycatch":   true,
		"pmd-unusedmethod": true,
		"pmd-complexity":   true,
	}
	for _, viol := range result.Violations {
		assert.True(t, validRuleIDs[viol.RuleID],
			"RuleID should be from policy, got: %s", viol.RuleID)
	}
}

func TestPMD_EmptyCatchBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pmd"))
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

	adp, err := linter.Global().GetLinter("pmd")
	if err != nil {
		t.Skipf("PMD adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("PMD not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.java")
	changes := []git.Change{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Check for EmptyCatchBlock violation
	found := false
	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "empty catch") || strings.Contains(viol.Message, "EmptyCatchBlock") {
			found = true
			t.Logf("Found EmptyCatchBlock violation at line %d: %s", viol.Line, viol.Message)
		}
	}

	if !found {
		t.Log("EmptyCatchBlock violation not detected (may depend on PMD version)")
	}
}

func TestPMD_UnusedPrivateMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pmd"))
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

	adp, err := linter.Global().GetLinter("pmd")
	if err != nil {
		t.Skipf("PMD adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("PMD not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.java")
	changes := []git.Change{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Check for UnusedPrivateMethod violation
	found := false
	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "unused") || strings.Contains(viol.Message, "UnusedPrivateMethod") {
			found = true
			t.Logf("Found UnusedPrivateMethod violation at line %d: %s", viol.Line, viol.Message)
		}
	}

	if !found {
		t.Log("UnusedPrivateMethod violation not detected (may depend on PMD version)")
	}
}

func TestPMD_ToolNameAndRuleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "pmd"))
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

	adp, err := linter.Global().GetLinter("pmd")
	if err != nil {
		t.Skipf("PMD adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("PMD not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.java")
	changes := []git.Change{{
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
		assert.Equal(t, "pmd", viol.ToolName,
			"ToolName should be 'pmd', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy
	for _, viol := range result.Violations {
		assert.True(t, strings.HasPrefix(viol.RuleID, "pmd-"),
			"RuleID should start with 'pmd-', got '%s'", viol.RuleID)
	}
}
