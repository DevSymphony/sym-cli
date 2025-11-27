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

func TestPrettier_ValidateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "prettier"))
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
	adp, err := adapterRegistry.Global().GetAdapter("prettier")
	if err != nil {
		t.Skipf("Prettier adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Prettier not available: %v", err)
	}

	// 5. Create GitChange
	testFile := filepath.Join(testdataDir, "Test.ts")
	require.FileExists(t, testFile, "Test.ts should exist")

	changes := []validator.GitChange{{
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

	// 8. Assertions - Prettier should detect formatting issues
	assert.GreaterOrEqual(t, len(result.Violations), 1, "Should detect at least 1 formatting violation")

	// Check that ToolName is prettier
	for _, viol := range result.Violations {
		assert.Equal(t, "prettier", viol.ToolName, "ToolName should be prettier")
	}

	// Check that RuleID is from policy
	validRuleIDs := map[string]bool{
		"prettier-quotes":     true,
		"prettier-indent":     true,
		"prettier-printwidth": true,
	}
	for _, viol := range result.Violations {
		assert.True(t, validRuleIDs[viol.RuleID],
			"RuleID should be from policy, got: %s", viol.RuleID)
	}
}

func TestPrettier_FormattingCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "prettier"))
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

	adp, err := adapterRegistry.Global().GetAdapter("prettier")
	if err != nil {
		t.Skipf("Prettier adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Prettier not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.ts")
	changes := []validator.GitChange{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// Prettier reports files that need formatting
	// The Test.ts file has:
	// - Double quotes (should be single quotes)
	// - 4-space indentation (should be 2)
	// - Lines exceeding printWidth

	if len(result.Violations) == 0 {
		t.Log("No formatting violations found (file may already be formatted)")
	} else {
		t.Logf("Found %d formatting issue(s)", len(result.Violations))
		for _, viol := range result.Violations {
			t.Logf("  File needs formatting: %s", viol.File)
		}
	}

	// Prettier should detect that the file needs formatting
	assert.GreaterOrEqual(t, len(result.Violations), 1, "Test.ts should need formatting")
}

func TestPrettier_QuotesAndIndent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "prettier"))
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

	adp, err := adapterRegistry.Global().GetAdapter("prettier")
	if err != nil {
		t.Skipf("Prettier adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Prettier not available: %v", err)
	}

	// Read the test file to verify it has violations
	testFile := filepath.Join(testdataDir, "Test.ts")
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Verify the file contains double quotes (which should be single)
	assert.Contains(t, string(content), `"https://`, "Test file should contain double quotes")

	changes := []validator.GitChange{{
		FilePath: testFile,
		Status:   "M",
		Diff:     "",
	}}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)
	require.NoError(t, err)

	// The file has double quotes but config requires single quotes
	// Prettier should report this file needs formatting
	assert.GreaterOrEqual(t, len(result.Violations), 1, "Should detect formatting issues with quotes")
}

func TestPrettier_ToolNameAndRuleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "prettier"))
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

	adp, err := adapterRegistry.Global().GetAdapter("prettier")
	if err != nil {
		t.Skipf("Prettier adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Prettier not available: %v", err)
	}

	testFile := filepath.Join(testdataDir, "Test.ts")
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
		assert.Equal(t, "prettier", viol.ToolName,
			"ToolName should be 'prettier', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy
	for _, viol := range result.Violations {
		assert.True(t, strings.HasPrefix(viol.RuleID, "prettier-"),
			"RuleID should start with 'prettier-', got '%s'", viol.RuleID)
	}

	// Verify Severity comes from policy (warning for prettier rules)
	for _, viol := range result.Violations {
		assert.Equal(t, "warning", viol.Severity,
			"Severity should be 'warning' from policy, got '%s'", viol.Severity)
	}
}
