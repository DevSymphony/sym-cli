package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/validator"
)

// TestValidator_RBAC_AdminFullAccess verifies that admin role has full write access
func TestValidator_RBAC_AdminFullAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "alice") // alice is admin

	// Admin should be able to modify any file
	testFiles := []string{
		"testdata/rbac/src/components/Button.js",
		"testdata/rbac/src/core/engine.js",
		"testdata/rbac/src/api/client.js",
		"testdata/rbac/tests/test.js",
		"testdata/rbac/config/settings.json",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(getTestdataDir(t), file)
		changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

		ctx := context.Background()
		result, err := v.ValidateChanges(ctx, changes)

		if err != nil {
			t.Fatalf("ValidateChanges failed for %s: %v", file, err)
		}

		assertNoRBACViolation(t, result)
		t.Logf("✓ Admin successfully modified: %s", file)
	}
}

// TestValidator_RBAC_DeveloperAllowedFiles verifies that developer can modify allowed files
func TestValidator_RBAC_DeveloperAllowedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Developer should be able to modify src/components/* and tests/*
	testFiles := []string{
		"testdata/rbac/src/components/Button.js",
		"testdata/rbac/tests/test.js",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(getTestdataDir(t), file)
		changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

		ctx := context.Background()
		result, err := v.ValidateChanges(ctx, changes)

		if err != nil {
			t.Fatalf("ValidateChanges failed for %s: %v", file, err)
		}

		assertNoRBACViolation(t, result)
		t.Logf("✓ Developer successfully modified: %s", file)
	}
}

// TestValidator_RBAC_DeveloperDeniedCoreFiles verifies that developer cannot modify core files
func TestValidator_RBAC_DeveloperDeniedCoreFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Developer should NOT be able to modify src/core/*
	filePath := filepath.Join(getTestdataDir(t), "testdata/rbac/src/core/engine.js")
	changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	assertRBACViolation(t, result, "charlie")
	t.Logf("✓ Developer correctly blocked from modifying core file")
}

// TestValidator_RBAC_DeveloperDeniedAPIFiles verifies that developer cannot modify API files
func TestValidator_RBAC_DeveloperDeniedAPIFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Developer should NOT be able to modify src/api/*
	filePath := filepath.Join(getTestdataDir(t), "testdata/rbac/src/api/client.js")
	changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	assertRBACViolation(t, result, "charlie")
	t.Logf("✓ Developer correctly blocked from modifying API file")
}

// TestValidator_RBAC_ViewerDenied verifies that viewer role has no write access
func TestValidator_RBAC_ViewerDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "frank") // frank is viewer

	// Viewer should NOT be able to modify any file
	testFiles := []string{
		"testdata/rbac/src/components/Button.js",
		"testdata/rbac/tests/test.js",
		"testdata/rbac/config/settings.json",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(getTestdataDir(t), file)
		changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

		ctx := context.Background()
		result, err := v.ValidateChanges(ctx, changes)

		if err != nil {
			t.Fatalf("ValidateChanges failed for %s: %v", file, err)
		}

		assertRBACViolation(t, result, "frank")
		t.Logf("✓ Viewer correctly blocked from modifying: %s", file)
	}
}

// TestValidator_RBAC_MixedPermissions verifies mixed file permissions in a single commit
func TestValidator_RBAC_MixedPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Mix of allowed and denied files
	allowedFile := filepath.Join(getTestdataDir(t), "testdata/rbac/src/components/Button.js")
	deniedFile := filepath.Join(getTestdataDir(t), "testdata/rbac/src/core/engine.js")

	changes := []validator.GitChange{
		createGitChangeFromFile(t, allowedFile),
		createGitChangeFromFile(t, deniedFile),
	}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	// Should have RBAC violation for the denied file
	assertRBACViolation(t, result, "charlie")
	t.Logf("✓ Mixed permissions correctly detected RBAC violation")
}

// TestValidator_RBAC_DeletedFilesSkipped verifies that deleted files don't trigger RBAC
func TestValidator_RBAC_DeletedFilesSkipped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Deleted files should not trigger RBAC violations
	filePath := filepath.Join(getTestdataDir(t), "testdata/rbac/src/core/engine.js")

	changes := []validator.GitChange{
		{
			FilePath: filePath,
			Status:   "D", // Deleted
			Diff:     "--- a/src/core/engine.js\n+++ /dev/null\n",
		},
	}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	assertNoRBACViolation(t, result)
	t.Logf("✓ Deleted files correctly skipped RBAC validation")
}

// TestValidator_RBAC_Disabled verifies that RBAC can be disabled
func TestValidator_RBAC_Disabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupRBACEnvironment(t, "charlie") // charlie is developer

	// Load a policy with RBAC disabled
	pol := loadPolicyFromTestdata(t, "testdata/rbac/code-policy.json")

	// Disable RBAC for this test
	if pol.Enforce.RBACConfig != nil {
		pol.Enforce.RBACConfig.Enabled = false
	}

	v := createTestValidator(t, pol)

	// Developer should be able to modify core files when RBAC is disabled
	filePath := filepath.Join(getTestdataDir(t), "testdata/rbac/src/core/engine.js")
	changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	assertNoRBACViolation(t, result)
	t.Logf("✓ RBAC disabled - no violations detected")
}

// TestValidator_RBAC_UnknownUser verifies behavior with unknown users
func TestValidator_RBAC_UnknownUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "unknown_user") // User not in roles.json

	// Unknown user should have no permissions (default to viewer-like behavior)
	filePath := filepath.Join(getTestdataDir(t), "testdata/rbac/src/components/Button.js")
	changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	if err != nil {
		t.Fatalf("ValidateChanges failed: %v", err)
	}

	// Unknown user should be denied (no role = no permissions)
	assertRBACViolation(t, result, "unknown_user")
	t.Logf("✓ Unknown user correctly denied access")
}

// TestValidator_RBAC_GlobPatternMatching verifies glob pattern matching works correctly
func TestValidator_RBAC_GlobPatternMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	v := createRBACTestValidator(t, "charlie") // charlie is developer

	// Test various glob patterns
	testCases := []struct {
		file        string
		shouldAllow bool
		description string
	}{
		{
			file:        "testdata/rbac/src/components/Button.js",
			shouldAllow: true,
			description: "src/** allows src/components/*",
		},
		{
			file:        "testdata/rbac/src/core/engine.js",
			shouldAllow: false,
			description: "src/core/** deny overrides src/** allow",
		},
		{
			file:        "testdata/rbac/src/api/client.js",
			shouldAllow: false,
			description: "src/api/** deny overrides src/** allow",
		},
		{
			file:        "testdata/rbac/tests/test.js",
			shouldAllow: true,
			description: "tests/** is allowed",
		},
	}

	for _, tc := range testCases {
		filePath := filepath.Join(getTestdataDir(t), tc.file)
		changes := []validator.GitChange{createGitChangeFromFile(t, filePath)}

		ctx := context.Background()
		result, err := v.ValidateChanges(ctx, changes)

		if err != nil {
			t.Fatalf("ValidateChanges failed for %s: %v", tc.file, err)
		}

		if tc.shouldAllow {
			assertNoRBACViolation(t, result)
			t.Logf("✓ Glob pattern test passed: %s (allowed)", tc.description)
		} else {
			assertRBACViolation(t, result, "charlie")
			t.Logf("✓ Glob pattern test passed: %s (denied)", tc.description)
		}
	}
}
