package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCP_GetConventionsByCategory tests MCP server's ability to query conventions by category
func TestMCP_GetConventionsByCategory(t *testing.T) {
	// Load JavaScript policy
	policyPath := filepath.Join(".sym", "js-code-policy.json")

	// Skip if policy file doesn't exist (e.g., in CI environment)
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		t.Skipf("Policy file not found: %s (skipping in CI)", policyPath)
	}

	policy, err := loadPolicy(policyPath)
	require.NoError(t, err, "Failed to load JavaScript policy")
	require.NotEmpty(t, policy.Rules, "Policy should have rules")

	t.Logf("Loaded policy with %d rules", len(policy.Rules))

	// Test 1: Get security conventions
	t.Run("security_conventions", func(t *testing.T) {
		securityRules := filterRulesByCategory(policy.Rules, "security")

		assert.Greater(t, len(securityRules), 0, "Should have security rules")
		t.Logf("Found %d security rules", len(securityRules))

		// Verify we got the expected security rules
		expectedSecurityRules := []string{"SEC-001", "SEC-002", "SEC-003"}
		foundRules := make(map[string]bool)

		for _, rule := range securityRules {
			foundRules[rule.ID] = true
			t.Logf("  - %s: %s (severity: %s)", rule.ID, rule.Message, rule.Severity)
		}

		for _, expectedID := range expectedSecurityRules {
			assert.True(t, foundRules[expectedID], "Should find rule %s", expectedID)
		}
	})

	// Test 2: Get style conventions
	t.Run("style_conventions", func(t *testing.T) {
		styleRules := filterRulesByCategory(policy.Rules, "style")

		assert.Greater(t, len(styleRules), 0, "Should have style rules")
		t.Logf("Found %d style rules", len(styleRules))

		// Verify style rules
		expectedStyleRules := []string{"STYLE-001", "STYLE-002", "STYLE-003"}
		foundRules := make(map[string]bool)

		for _, rule := range styleRules {
			foundRules[rule.ID] = true
			t.Logf("  - %s: %s", rule.ID, rule.Message)
		}

		for _, expectedID := range expectedStyleRules {
			assert.True(t, foundRules[expectedID], "Should find rule %s", expectedID)
		}
	})

	// Test 3: Get error handling conventions
	t.Run("error_handling_conventions", func(t *testing.T) {
		errorRules := filterRulesByCategory(policy.Rules, "error_handling")

		assert.Greater(t, len(errorRules), 0, "Should have error handling rules")
		t.Logf("Found %d error handling rules", len(errorRules))

		// Verify error handling rules
		expectedErrorRules := []string{"ERR-001", "ERR-002"}
		foundRules := make(map[string]bool)

		for _, rule := range errorRules {
			foundRules[rule.ID] = true
			t.Logf("  - %s: %s", rule.ID, rule.Message)
		}

		for _, expectedID := range expectedErrorRules {
			assert.True(t, foundRules[expectedID], "Should find rule %s", expectedID)
		}
	})

	// Test 4: Filter by severity
	t.Run("filter_by_severity", func(t *testing.T) {
		errorLevelRules := filterRulesBySeverity(policy.Rules, "error")
		warningLevelRules := filterRulesBySeverity(policy.Rules, "warning")
		infoLevelRules := filterRulesBySeverity(policy.Rules, "info")

		t.Logf("Error rules: %d", len(errorLevelRules))
		t.Logf("Warning rules: %d", len(warningLevelRules))
		t.Logf("Info rules: %d", len(infoLevelRules))

		assert.Greater(t, len(errorLevelRules), 0, "Should have error-level rules")
		assert.Greater(t, len(warningLevelRules), 0, "Should have warning-level rules")
	})

	// Test 5: Combined filtering (category + severity)
	t.Run("combined_filter_security_errors", func(t *testing.T) {
		securityRules := filterRulesByCategory(policy.Rules, "security")
		securityErrors := filterRulesBySeverity(securityRules, "error")

		t.Logf("Security error rules: %d", len(securityErrors))
		assert.Greater(t, len(securityErrors), 0, "Should have security error rules")

		for _, rule := range securityErrors {
			assert.Equal(t, "security", rule.Category)
			assert.Equal(t, "error", rule.Severity)
			t.Logf("  - %s: %s", rule.ID, rule.Message)
		}
	})
}

// TestMCP_ValidateAIGeneratedCode tests validation of AI-generated code against conventions
func TestMCP_ValidateAIGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping MCP validation test")
	}

	// Load policy
	policyPath := filepath.Join(".sym", "js-code-policy.json")
	policy, err := loadPolicy(policyPath)
	require.NoError(t, err)

	// Create LLM client
	client := llm.NewClient(
		apiKey,
		llm.WithModel("gpt-4o"),
		llm.WithTimeout(30*time.Second),
	)

	// Create validator
	v := validator.NewLLMValidator(client, policy)
	ctx := context.Background()

	// Test 1: Validate BAD code (should find multiple violations)
	t.Run("validate_bad_code", func(t *testing.T) {
		t.Log("Reading bad example code...")
		badCode, err := os.ReadFile(filepath.Join("examples", "bad-example.js"))
		require.NoError(t, err, "Failed to read bad-example.js")

		// Format as git diff with + prefix for each line
		lines := strings.Split(string(badCode), "\n")
		var diffLines []string
		for _, line := range lines {
			diffLines = append(diffLines, "+"+line)
		}
		formattedDiff := strings.Join(diffLines, "\n")

		changes := []validator.GitChange{
			{
				FilePath: "examples/bad-example.js",
				Diff:     formattedDiff,
			},
		}

		t.Log("Validating bad code against conventions...")
		result, err := v.Validate(ctx, changes)
		require.NoError(t, err, "Validation should not error")

		t.Logf("Validation completed: checked=%d, violations=%d",
			result.Checked, len(result.Violations))

		// Should find multiple violations
		assert.Greater(t, len(result.Violations), 0,
			"Should detect violations in bad code")

		// Log all violations
		for i, violation := range result.Violations {
			t.Logf("  Violation %d: [%s] %s - %s",
				i+1, violation.Severity, violation.RuleID, violation.Message)
		}

		// Check for specific critical violations
		foundSecurityViolation := false
		foundErrorHandlingViolation := false

		for _, v := range result.Violations {
			if v.Severity == "error" {
				foundSecurityViolation = true
			}
			// Check if we caught error handling issues
			if contains(v.RuleID, "ERR-") {
				foundErrorHandlingViolation = true
			}
		}

		assert.True(t, foundSecurityViolation || foundErrorHandlingViolation,
			"Should detect at least one critical violation")
	})

	// Test 2: Validate GOOD code (should pass or have minimal violations)
	t.Run("validate_good_code", func(t *testing.T) {
		t.Log("Reading good example code...")
		goodCode, err := os.ReadFile(filepath.Join("examples", "good-example.js"))
		require.NoError(t, err, "Failed to read good-example.js")

		// Format as git diff with + prefix for each line
		lines := strings.Split(string(goodCode), "\n")
		var diffLines []string
		for _, line := range lines {
			diffLines = append(diffLines, "+"+line)
		}
		formattedDiff := strings.Join(diffLines, "\n")

		changes := []validator.GitChange{
			{
				FilePath: "examples/good-example.js",
				Diff:     formattedDiff,
			},
		}

		t.Log("Validating good code against conventions...")
		result, err := v.Validate(ctx, changes)
		require.NoError(t, err)

		t.Logf("Validation completed: checked=%d, violations=%d",
			result.Checked, len(result.Violations))

		if len(result.Violations) == 0 {
			t.Log("✓ Good code passed all checks!")
		} else {
			t.Logf("Good code has %d violations:", len(result.Violations))
			for i, v := range result.Violations {
				t.Logf("  Violation %d: [%s] %s - %s",
					i+1, v.Severity, v.RuleID, v.Message)
			}
		}

		// Good code should have significantly fewer violations than bad code
		// We'll run bad code validation for comparison if needed
	})

	// Test 3: Category-specific validation
	t.Run("validate_security_only", func(t *testing.T) {
		t.Log("Testing security-focused validation...")

		// Filter policy to only security rules
		securityPolicy := &schema.CodePolicy{
			Version: policy.Version,
			Rules:   filterRulesByCategory(policy.Rules, "security"),
		}

		securityValidator := validator.NewLLMValidator(client, securityPolicy)

		// Code with security violation (format as git diff with + prefix)
		codeWithSecurityIssue := `+const apiKey = "sk-1234567890abcdef"; // Hardcoded secret
+fetch('/api/data', {
+  headers: { 'Authorization': 'Bearer ' + apiKey }
+});`

		changes := []validator.GitChange{
			{
				FilePath: "test-security.js",
				Diff:     codeWithSecurityIssue,
			},
		}

		result, err := securityValidator.Validate(ctx, changes)
		require.NoError(t, err)

		t.Logf("Security validation: checked=%d, violations=%d",
			result.Checked, len(result.Violations))

		// Should detect hardcoded API key
		assert.Greater(t, len(result.Violations), 0,
			"Should detect security violation")

		for _, v := range result.Violations {
			t.Logf("  - [%s] %s: %s", v.Severity, v.RuleID, v.Message)
			assert.Equal(t, "security", extractCategory(v.RuleID),
				"Should only report security violations")
		}
	})

	// Test 4: Incremental validation (simulating AI fixing violations)
	t.Run("iterative_fix_workflow", func(t *testing.T) {
		t.Log("Testing iterative fix workflow...")

		// Iteration 1: Code with hardcoded secret (format with + prefix)
		iteration1 := `+const apiKey = "sk-test123";`

		result1, err := v.Validate(ctx, []validator.GitChange{
			{FilePath: "test.js", Diff: iteration1},
		})
		require.NoError(t, err)
		violations1 := len(result1.Violations)
		t.Logf("Iteration 1: %d violations", violations1)

		// Iteration 2: AI fixes the issue (format with + prefix)
		iteration2 := `+const apiKey = process.env.API_KEY;`

		result2, err := v.Validate(ctx, []validator.GitChange{
			{FilePath: "test.js", Diff: iteration2},
		})
		require.NoError(t, err)
		violations2 := len(result2.Violations)
		t.Logf("Iteration 2: %d violations", violations2)

		// Should have fewer violations after fix
		if violations1 > 0 {
			assert.LessOrEqual(t, violations2, violations1,
				"Fixed code should have fewer or equal violations")
			t.Logf("✓ Iterative fix successful: %d -> %d violations",
				violations1, violations2)
		}
	})
}

// TestMCP_EndToEndWorkflow tests the complete workflow with MCP integration
func TestMCP_EndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	t.Log("========== MCP INTEGRATION E2E WORKFLOW ==========")

	// Step 1: Load conventions from policy (simulating MCP query)
	t.Log("STEP 1: Loading conventions via MCP")
	policyPath := filepath.Join(".sym", "js-code-policy.json")
	policy, err := loadPolicy(policyPath)
	require.NoError(t, err)
	t.Logf("✓ Loaded %d conventions", len(policy.Rules))

	// Step 2: Query conventions by category (MCP tool call)
	t.Log("STEP 2: Querying security conventions")
	securityConventions := filterRulesByCategory(policy.Rules, "security")
	t.Logf("✓ Retrieved %d security conventions", len(securityConventions))
	for _, rule := range securityConventions {
		t.Logf("  - %s: %s", rule.ID, rule.Message)
	}

	// Step 3: AI generates code (simulated)
	t.Log("STEP 3: AI generates code based on conventions")
	generatedCode := `+// AI-generated authentication handler
+const authenticateUser = async (username, password) => {
+  const apiKey = process.env.API_KEY;  // Following SEC-001
+
+  try {
+    const response = await fetch('/api/auth', {
+      method: 'POST',
+      headers: { 'X-API-Key': apiKey },
+      body: JSON.stringify({ username, password })
+    });
+
+    if (!response.ok) {
+      throw new Error('Authentication failed');
+    }
+
+    return await response.json();
+  } catch (error) {
+    console.error('Auth error:', error);  // Following ERR-002
+    throw error;
+  }
+};`
	t.Log("✓ Code generated with convention awareness")

	// Step 4: Validate generated code
	t.Log("STEP 4: Validating AI-generated code")
	client := llm.NewClient(apiKey, llm.WithModel("gpt-4o"))
	v := validator.NewLLMValidator(client, policy)

	result, err := v.Validate(context.Background(), []validator.GitChange{
		{FilePath: "auth.js", Diff: generatedCode},
	})
	require.NoError(t, err)

	t.Logf("✓ Validation completed: %d violations found", len(result.Violations))

	if len(result.Violations) == 0 {
		t.Log("✓ AI-generated code follows all conventions!")
	} else {
		t.Log("⚠ Violations detected:")
		for _, v := range result.Violations {
			t.Logf("  - [%s] %s", v.Severity, v.Message)
		}
	}

	t.Log("====================================")
}

// Helper functions (MCP-specific)

func extractCategory(ruleID string) string {
	// Extract category from rule ID (e.g., "SEC-001" -> "security")
	categoryMap := map[string]string{
		"SEC":   "security",
		"STYLE": "style",
		"ERR":   "error_handling",
		"ARCH":  "architecture",
		"PERF":  "performance",
	}

	for prefix, category := range categoryMap {
		if len(ruleID) >= len(prefix) && ruleID[:len(prefix)] == prefix {
			return category
		}
	}
	return ""
}
