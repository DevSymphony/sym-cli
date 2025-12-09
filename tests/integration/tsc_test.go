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

func TestTSC_ValidateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "tsc"))
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
	adp, err := linter.Global().GetLinter("tsc")
	if err != nil {
		t.Skipf("TSC adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("TSC not available: %v", err)
	}

	// 5. Create GitChange
	testFile := filepath.Join(testdataDir, "Test.ts")
	require.FileExists(t, testFile, "Test.ts should exist")

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
		t.Logf("  - %s:%d:%d: [%s] %s (tool: %s)",
			viol.File, viol.Line, viol.Column, viol.RuleID, viol.Message, viol.ToolName)
	}

	// 8. Assertions - TSC should detect strict null check violations
	assert.GreaterOrEqual(t, len(result.Violations), 2, "Should detect at least 2 type violations")

	// Check that ToolName is tsc
	for _, viol := range result.Violations {
		assert.Equal(t, "tsc", viol.ToolName, "ToolName should be tsc")
	}

	// Check that RuleID is from policy
	for _, viol := range result.Violations {
		assert.Equal(t, "tsc-strictnull", viol.RuleID,
			"RuleID should be 'tsc-strictnull' from policy, got: %s", viol.RuleID)
	}
}

func TestTSC_StrictNullChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "tsc"))
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

	adp, err := linter.Global().GetLinter("tsc")
	if err != nil {
		t.Skipf("TSC adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("TSC not available: %v", err)
	}

	// Verify config has strictNullChecks enabled
	configPath := filepath.Join(testdataDir, ".sym", "tsconfig.json")
	configData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(configData), `"strictNullChecks": true`, "Config should have strictNullChecks enabled")

	testFile := filepath.Join(testdataDir, "Test.ts")
	changes := []git.Change{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Check for undefined/null type errors in messages
	hasTypeError := false
	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "undefined") || strings.Contains(viol.Message, "not assignable") {
			hasTypeError = true
			break
		}
	}
	assert.True(t, hasTypeError, "Should detect undefined type assignment violations")
}

func TestTSC_TypeErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "tsc"))
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

	adp, err := linter.Global().GetLinter("tsc")
	if err != nil {
		t.Skipf("TSC adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("TSC not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.ts")
	changes := []git.Change{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Check that TSC reports type errors
	typeErrorCount := 0
	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "Type") && strings.Contains(viol.Message, "not assignable") {
			typeErrorCount++
			t.Logf("Type error at line %d: %s", viol.Line, viol.Message)
		}
	}

	assert.GreaterOrEqual(t, typeErrorCount, 2, "Should detect at least 2 type assignment errors")
}

func TestTSC_ToolNameAndRuleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "tsc"))
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

	adp, err := linter.Global().GetLinter("tsc")
	if err != nil {
		t.Skipf("TSC adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("TSC not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.ts")
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
		assert.Equal(t, "tsc", viol.ToolName,
			"ToolName should be 'tsc', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy
	for _, viol := range result.Violations {
		assert.Equal(t, "tsc-strictnull", viol.RuleID,
			"RuleID should be 'tsc-strictnull' from policy, got '%s'", viol.RuleID)
	}

	// Verify Severity comes from policy
	for _, viol := range result.Violations {
		assert.Equal(t, "error", viol.Severity,
			"Severity should be 'error' from policy, got '%s'", viol.Severity)
	}
}
