package integration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// ============================================================================
// JavaScript Pattern Tests
// ============================================================================

func TestValidator_JavaScript_Pattern_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load policy from testdata
	policy := loadPolicyFromTestdata(t, "testdata/javascript/pattern/code-policy.json")
	require.Equal(t, 4, len(policy.Rules), "Should have 4 rules: 3 naming + 1 security")

	// Create validator
	v := createTestValidator(t, policy)

	// Test naming violations
	t.Run("NamingViolations", func(t *testing.T) {
		filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/pattern/naming-violations.js")
		result, err := v.Validate(filePath)
		require.NoError(t, err)
		assertViolationsDetected(t, result)
	})

	// Test security violations
	t.Run("SecurityViolations", func(t *testing.T) {
		filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/pattern/security-violations.js")
		result, err := v.Validate(filePath)
		require.NoError(t, err)
		assertViolationsDetected(t, result)
	})
}

func TestValidator_JavaScript_Pattern_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/pattern/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/pattern/valid.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// JavaScript Length Tests
// ============================================================================

func TestValidator_JavaScript_Length_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/length/code-policy.json")
	require.Equal(t, 3, len(policy.Rules), "Should have 3 rules: line/function/params length")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/length/length-violations.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_JavaScript_Length_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/length/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/length/valid.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// JavaScript Style Tests
// ============================================================================

func TestValidator_JavaScript_Style_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/style/code-policy.json")
	require.GreaterOrEqual(t, len(policy.Rules), 3, "Should have at least 3 style rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/style/style-violations.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_JavaScript_Style_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/style/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/style/valid.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// JavaScript AST Tests
// ============================================================================

func TestValidator_JavaScript_AST_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/ast/code-policy.json")
	require.Equal(t, 3, len(policy.Rules), "Should have 3 AST rules for naming")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/ast/naming-violations.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_JavaScript_AST_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/javascript/ast/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/javascript/ast/valid.js")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// TypeScript TypeChecker Tests
// ============================================================================

func TestValidator_TypeScript_TypeChecker_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/typescript/typechecker/code-policy.json")
	require.Equal(t, 3, len(policy.Rules), "Should have 3 type checking rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/typescript/typechecker/type-errors.ts")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_TypeScript_TypeChecker_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/typescript/typechecker/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/typescript/typechecker/valid.ts")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// Java Pattern Tests
// ============================================================================

func TestValidator_Java_Pattern_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/pattern/code-policy.json")
	require.Equal(t, 5, len(policy.Rules), "Should have 5 naming rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/pattern/NamingViolations.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_Java_Pattern_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/pattern/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/pattern/ValidNaming.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// Java Length Tests
// ============================================================================

func TestValidator_Java_Length_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/length/code-policy.json")
	require.Equal(t, 3, len(policy.Rules), "Should have 3 length rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/length/LengthViolations.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_Java_Length_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/length/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/length/ValidLength.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// Java Style Tests
// ============================================================================

func TestValidator_Java_Style_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/style/code-policy.json")
	require.GreaterOrEqual(t, len(policy.Rules), 5, "Should have at least 5 style rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/style/StyleViolations.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_Java_Style_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/style/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/style/ValidStyle.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}

// ============================================================================
// Java AST Tests
// ============================================================================

func TestValidator_Java_AST_Violations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/ast/code-policy.json")
	require.Equal(t, 4, len(policy.Rules), "Should have 4 AST rules")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/ast/AstViolations.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertViolationsDetected(t, result)
}

func TestValidator_Java_AST_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policy := loadPolicyFromTestdata(t, "testdata/java/ast/code-policy.json")
	v := createTestValidator(t, policy)

	filePath := filepath.Join(getTestdataDir(t), "testdata/java/ast/ValidAst.java")
	result, err := v.Validate(filePath)
	require.NoError(t, err)
	assertNoPolicyViolations(t, result)
}
