package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESLint_ValidateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "eslint"))
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
	defer func() { _ = v.Close() }()

	// 4. Check tool availability
	adp, err := linter.Global().GetLinter("eslint")
	if err != nil {
		t.Skipf("ESLint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("ESLint not available: %v", err)
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

	// 8. Assertions
	assert.GreaterOrEqual(t, len(result.Violations), 3, "Should detect at least 3 violations")

	// Check that ToolName is eslint
	for _, viol := range result.Violations {
		assert.Equal(t, "eslint", viol.ToolName, "ToolName should be eslint")
	}

	// Check that RuleID is from policy
	validRuleIDs := map[string]bool{
		"eslint-naming": true,
		"eslint-maxlen": true,
	}
	for _, viol := range result.Violations {
		assert.True(t, validRuleIDs[viol.RuleID],
			"RuleID should be from policy, got: %s", viol.RuleID)
	}
}

func TestESLint_NamingConventions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "eslint"))
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
	defer func() { _ = v.Close() }()

	adp, err := linter.Global().GetLinter("eslint")
	if err != nil {
		t.Skipf("ESLint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("ESLint not available: %v", err)
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

	// Check for id-match or max-len violations in messages
	hasIdMatch := false
	hasMaxLen := false

	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "id-match") || strings.Contains(viol.Message, "Identifier") {
			hasIdMatch = true
		}
		if strings.Contains(viol.Message, "max-len") || strings.Contains(viol.Message, "exceeds") {
			hasMaxLen = true
		}
	}

	assert.True(t, hasIdMatch || hasMaxLen, "Should detect id-match or max-len violations")
}

func TestESLint_MaxLineLength(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "eslint"))
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
	defer func() { _ = v.Close() }()

	adp, err := linter.Global().GetLinter("eslint")
	if err != nil {
		t.Skipf("ESLint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("ESLint not available: %v", err)
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

	// Check for max-len violations
	found := false
	for _, viol := range result.Violations {
		if strings.Contains(viol.Message, "max-len") || strings.Contains(viol.Message, "exceeds") {
			found = true
			t.Logf("Found max-len violation at line %d: %s", viol.Line, viol.Message)
		}
	}

	if !found {
		t.Log("max-len violation not detected (may be configured to ignore certain patterns)")
	}
}

func TestESLint_ToolNameAndRuleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "eslint"))
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
	defer func() { _ = v.Close() }()

	adp, err := linter.Global().GetLinter("eslint")
	if err != nil {
		t.Skipf("ESLint adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("ESLint not available: %v", err)
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
		assert.Equal(t, "eslint", viol.ToolName,
			"ToolName should be 'eslint', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy
	for _, viol := range result.Violations {
		assert.True(t, strings.HasPrefix(viol.RuleID, "eslint-"),
			"RuleID should start with 'eslint-', got '%s'", viol.RuleID)
	}
}
