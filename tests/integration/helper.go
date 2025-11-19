package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/policy"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestdataDir returns the path to the testdata directory
func getTestdataDir(t *testing.T) string {
	t.Helper()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Go up two levels from tests/integration to project root
	projectRoot := filepath.Join(cwd, "../..")

	return projectRoot
}

// loadPolicyFromTestdata loads a code-policy.json from testdata directory
func loadPolicyFromTestdata(t *testing.T, relativePath string) *schema.CodePolicy {
	t.Helper()
	loader := policy.NewLoader(false)
	policyPath := filepath.Join(getTestdataDir(t), relativePath)
	pol, err := loader.LoadCodePolicy(policyPath)
	require.NoError(t, err, "Failed to load policy from %s", relativePath)
	require.NotNil(t, pol, "Policy should not be nil")
	return pol
}

// createTestValidator creates a validator with given policy and registers cleanup
func createTestValidator(t *testing.T, pol *schema.CodePolicy) *validator.Validator {
	t.Helper()
	v := validator.NewValidator(pol, testing.Verbose())
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Logf("Warning: failed to close validator: %v", err)
		}
	})
	return v
}

// assertViolationsDetected asserts that violations are found and logs them
func assertViolationsDetected(t *testing.T, result *validator.Result) {
	t.Helper()
	assert.False(t, result.Passed, "Should detect violations")
	assert.Greater(t, len(result.Violations), 0, "Should have violations")

	// Log violations for debugging
	if len(result.Violations) > 0 {
		t.Logf("Found %d violation(s):", len(result.Violations))
		for i, v := range result.Violations {
			t.Logf("  %d. [%s] %s at %s:%d:%d (severity: %s)",
				i+1, v.RuleID, v.Message, v.File, v.Line, v.Column, v.Severity)
		}
	}
}

// assertNoPolicyViolations asserts that no violations are found
func assertNoPolicyViolations(t *testing.T, result *validator.Result) {
	t.Helper()
	if !result.Passed || len(result.Violations) > 0 {
		// Log violations if any for debugging
		if len(result.Violations) > 0 {
			t.Logf("Unexpected violations found:")
			for i, v := range result.Violations {
				t.Logf("  %d. [%s] %s at %s:%d:%d",
					i+1, v.RuleID, v.Message, v.File, v.Line, v.Column)
			}
		}
	}
	assert.True(t, result.Passed, "Should pass validation")
	assert.Equal(t, 0, len(result.Violations), "Should have no violations")
}
