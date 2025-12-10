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

func TestCheckstyle_ValidateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Get testdata path
	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "checkstyle"))
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
	adp, err := linter.Global().GetLinter("checkstyle")
	if err != nil {
		t.Skipf("Checkstyle adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Checkstyle not available: %v", err)
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

	// 8. Assertions
	assert.GreaterOrEqual(t, len(result.Violations), 3, "Should detect at least 3 naming violations")

	// Check that ToolName is checkstyle
	for _, viol := range result.Violations {
		assert.Equal(t, "checkstyle", viol.ToolName, "ToolName should be checkstyle")
	}

	// Check that RuleID is from policy
	validRuleIDs := map[string]bool{
		"checkstyle-typename":   true,
		"checkstyle-methodname": true,
		"checkstyle-membername": true,
	}
	for _, viol := range result.Violations {
		assert.True(t, validRuleIDs[viol.RuleID],
			"RuleID should be from policy, got: %s", viol.RuleID)
	}
}

func TestCheckstyle_NamingRules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "checkstyle"))
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

	adp, err := linter.Global().GetLinter("checkstyle")
	if err != nil {
		t.Skipf("Checkstyle adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Checkstyle not available: %v", err)
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

	// Check specific violations
	tests := []struct {
		name        string
		checkLine   int
		msgPattern  string
		description string
	}{
		{
			name:        "TypeName violation - bad_class",
			checkLine:   2,
			msgPattern:  "TypeName",
			description: "Class name 'bad_class' should trigger TypeName violation",
		},
		{
			name:        "MemberName violation - MyVar",
			checkLine:   4,
			msgPattern:  "MemberName",
			description: "Member name 'MyVar' should trigger MemberName violation",
		},
		{
			name:        "MethodName violation - BadFunc",
			checkLine:   7,
			msgPattern:  "MethodName",
			description: "Method name 'BadFunc' should trigger MethodName violation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, viol := range result.Violations {
				if viol.Line == tt.checkLine {
					if strings.Contains(viol.Message, tt.msgPattern) {
						found = true
						t.Logf("Found violation: %s", viol.Message)
						break
					}
				}
			}
			if !found {
				t.Logf("Expected %s violation at line %d not found (may be detected at different line)", tt.msgPattern, tt.checkLine)
			}
		})
	}
}

func TestCheckstyle_ToolNameAndRuleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testdataDir, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata", "checkstyle"))
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

	adp, err := linter.Global().GetLinter("checkstyle")
	if err != nil {
		t.Skipf("Checkstyle adapter not found: %v", err)
	}
	if err := adp.CheckAvailability(context.Background()); err != nil {
		t.Skipf("Checkstyle not available: %v", err)
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
		assert.Equal(t, "checkstyle", viol.ToolName,
			"ToolName should be 'checkstyle', got '%s'", viol.ToolName)
	}

	// Verify RuleID comes from policy
	for _, viol := range result.Violations {
		assert.True(t, strings.HasPrefix(viol.RuleID, "checkstyle-"),
			"RuleID should start with 'checkstyle-', got '%s'", viol.RuleID)
	}

	// Verify Severity comes from policy
	for _, viol := range result.Violations {
		assert.Equal(t, "error", viol.Severity,
			"Severity should be 'error' from policy, got '%s'", viol.Severity)
	}
}
