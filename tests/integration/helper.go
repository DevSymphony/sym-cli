package integration

import (
	"os"
	"os/exec"
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

// getToolsDir returns the path to tools directory for test
func getToolsDir(t *testing.T) string {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	return filepath.Join(home, ".symphony", "tools")
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

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repo root so Validator.workDir is set correctly
	repoRoot := getTestdataDir(t)
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	// Create validator (it will use repo root as workDir)
	v := validator.NewValidator(pol, testing.Verbose())

	// Change back to original directory
	if err := os.Chdir(originalDir); err != nil {
		t.Fatalf("Failed to change back to original directory: %v", err)
	}

	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Logf("Warning: failed to close validator: %v", err)
		}
	})
	return v
}

// createGitChangeFromFile creates a GitChange from a test file using git diff
func createGitChangeFromFile(t *testing.T, filePath string) validator.GitChange {
	t.Helper()

	// Use git diff --no-index to generate a unified diff
	// This treats the file as newly added (comparing /dev/null to the file)
	cmd := exec.Command("git", "diff", "--no-index", "/dev/null", filePath)
	output, err := cmd.CombinedOutput()

	// git diff --no-index returns exit code 1 when there are differences (which is expected)
	// We only care about the output, not the exit code
	if err != nil {
		// Check if it's just the expected exit code 1
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected - there are differences between /dev/null and the file
		} else {
			// This is an actual error
			t.Fatalf("Failed to generate diff for %s: %v", filePath, err)
		}
	}

	// Use absolute path for FilePath
	// The adapters (checkstyle/pmd) will handle path resolution correctly
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		// If relative, make it absolute
		cwd, _ := os.Getwd()
		absPath, _ = filepath.Abs(filepath.Join(cwd, filePath))
	}

	return validator.GitChange{
		FilePath: absPath,
		Status:   "A", // Treat as Added file
		Diff:     string(output),
	}
}

// assertViolationsDetected asserts that violations are found and logs them
func assertViolationsDetected(t *testing.T, result *validator.ValidationResult) {
	t.Helper()
	assert.Greater(t, len(result.Violations), 0, "Should have violations")
	assert.Greater(t, result.Failed, 0, "Should have failed checks")

	// Log violations and metrics for debugging
	if len(result.Violations) > 0 {
		t.Logf("Found %d violation(s): Checked=%d, Passed=%d, Failed=%d",
			len(result.Violations), result.Checked, result.Passed, result.Failed)
		for i, v := range result.Violations {
			t.Logf("  %d. [%s] %s at %s:%d:%d (severity: %s)",
				i+1, v.RuleID, v.Message, v.File, v.Line, v.Column, v.Severity)
		}
	}
}

// assertNoPolicyViolations asserts that no violations are found
func assertNoPolicyViolations(t *testing.T, result *validator.ValidationResult) {
	t.Helper()
	if len(result.Violations) > 0 {
		// Log violations if any for debugging
		t.Logf("Unexpected violations found: Checked=%d, Passed=%d, Failed=%d",
			result.Checked, result.Passed, result.Failed)
		for i, v := range result.Violations {
			t.Logf("  %d. [%s] %s at %s:%d:%d",
				i+1, v.RuleID, v.Message, v.File, v.Line, v.Column)
		}
	}
	assert.Equal(t, 0, len(result.Violations), "Should have no violations")
	assert.Equal(t, 0, result.Failed, "Should have no failed checks")
	if result.Checked > 0 {
		assert.Equal(t, result.Checked, result.Passed, "All checks should pass")
	}
}

// setupRBACEnvironment sets up the environment for RBAC testing
// by setting the GIT_AUTHOR_NAME environment variable
func setupRBACEnvironment(t *testing.T, username string) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", username)
	t.Logf("Set GIT_AUTHOR_NAME=%s for RBAC testing", username)
}

// createRBACTestValidator creates a validator with RBAC testdata policy
// and sets up the git user environment
func createRBACTestValidator(t *testing.T, username string) *validator.Validator {
	t.Helper()
	setupRBACEnvironment(t, username)
	pol := loadPolicyFromTestdata(t, "testdata/rbac/code-policy.json")
	return createTestValidator(t, pol)
}

// assertRBACViolation asserts that an RBAC violation occurred
// for the expected user
func assertRBACViolation(t *testing.T, result *validator.ValidationResult, expectedUser string) {
	t.Helper()

	// Find RBAC violations (RuleID is "rbac-permission-denied")
	rbacViolations := []validator.Violation{}
	for _, v := range result.Violations {
		if v.RuleID == "rbac-permission-denied" {
			rbacViolations = append(rbacViolations, v)
		}
	}

	assert.Greater(t, len(rbacViolations), 0, "Should have RBAC violations")

	// Log RBAC violations for debugging
	if len(rbacViolations) > 0 {
		t.Logf("Found %d RBAC violation(s) for user '%s':", len(rbacViolations), expectedUser)
		for i, v := range rbacViolations {
			t.Logf("  %d. %s (file: %s)", i+1, v.Message, v.File)
		}
	}
}

// assertNoRBACViolation asserts that no RBAC violations occurred
func assertNoRBACViolation(t *testing.T, result *validator.ValidationResult) {
	t.Helper()

	// Check for RBAC violations (RuleID is "rbac-permission-denied")
	rbacViolations := []validator.Violation{}
	for _, v := range result.Violations {
		if v.RuleID == "rbac-permission-denied" {
			rbacViolations = append(rbacViolations, v)
		}
	}

	if len(rbacViolations) > 0 {
		t.Logf("Unexpected RBAC violations found:")
		for i, v := range rbacViolations {
			t.Logf("  %d. %s (file: %s)", i+1, v.Message, v.File)
		}
	}

	assert.Equal(t, 0, len(rbacViolations), "Should have no RBAC violations")
}
