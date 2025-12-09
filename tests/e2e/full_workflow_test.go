package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DevSymphony/sym-cli/internal/converter"
	"github.com/DevSymphony/sym-cli/internal/util/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FullWorkflow tests the complete workflow:
// 1. User provides natural language conventions (user-policy.json)
// 2. Convert command transforms it into structured policy
// 3. LLM coding tool queries conventions via MCP
// 4. Generated code is validated against conventions
func TestE2E_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping E2E test")
	}

	// Setup test directory
	testDir := t.TempDir()
	t.Logf("Test directory: %s", testDir)

	// ========== STEP 1: User creates natural language policy ==========
	t.Log("STEP 1: Creating user policy with natural language conventions")

	userPolicy := schema.UserPolicy{
		Version: "1.0.0",
		Defaults: &schema.UserDefaults{
			Languages: []string{"go"},
			Severity:  "warning",
		},
		Rules: []schema.UserRule{
			{
				Say:      "API 키나 비밀번호를 코드에 하드코딩하면 안됩니다. 환경변수를 사용하세요",
				Category: "security",
				Severity: "error",
			},
			{
				Say:      "모든 exported 함수는 godoc 주석이 있어야 합니다",
				Category: "documentation",
				Severity: "warning",
			},
			{
				Say:      "에러를 반환하는 함수를 호출할 때는 반드시 에러를 체크해야 합니다",
				Category: "error_handling",
				Severity: "warning",
			},
		},
	}

	userPolicyPath := filepath.Join(testDir, "user-policy.json")
	userPolicyData, err := json.MarshalIndent(userPolicy, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(userPolicyPath, userPolicyData, 0644)
	require.NoError(t, err)
	t.Logf("✓ Created user policy: %s", userPolicyPath)

	// ========== STEP 2: Convert natural language to structured policy ==========
	t.Log("STEP 2: Converting user policy using LLM")

	cfg := llm.LoadConfig()
	provider, err := llm.New(cfg)
	require.NoError(t, err, "LLM provider creation should succeed")

	outputDir := filepath.Join(testDir, ".sym")
	conv := converter.NewConverter(provider, outputDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := conv.Convert(ctx, &userPolicy)
	require.NoError(t, err, "Conversion should succeed")

	// Load the generated code policy
	codePolicyPath := filepath.Join(outputDir, "code-policy.json")
	codePolicyData, err := os.ReadFile(codePolicyPath)
	require.NoError(t, err, "Should be able to read generated code policy")

	var convertedPolicy schema.CodePolicy
	err = json.Unmarshal(codePolicyData, &convertedPolicy)
	require.NoError(t, err, "Conversion should succeed")
	require.NotNil(t, convertedPolicy)

	t.Logf("✓ Converted %d rules", len(convertedPolicy.Rules))

	// Verify conversion produced structured rules
	assert.Greater(t, len(convertedPolicy.Rules), 0, "Should have converted rules")
	for i, rule := range convertedPolicy.Rules {
		t.Logf("  Rule %d: %s (category: %s)", i+1, rule.ID, rule.Category)
	}

	// Files are already written by converter
	for _, filePath := range result.GeneratedFiles {
		t.Logf("✓ Generated: %s", filePath)
	}

	// ========== STEP 3: LLM coding tool queries conventions via MCP ==========
	t.Log("STEP 3: Simulating LLM tool querying conventions")

	// Simulate MCP tool call: get_conventions_by_category
	securityRules := filterRulesByCategory(convertedPolicy.Rules, "security")
	require.Greater(t, len(securityRules), 0, "Should have security rules")

	t.Logf("✓ Found %d security rules via MCP query", len(securityRules))
	for _, rule := range securityRules {
		t.Logf("  - %s: %s", rule.ID, rule.Message)
	}

	// Simulate LLM tool generating code based on conventions
	t.Log("STEP 3b: LLM generates code (simulated)")

	// Case A: Generated code that VIOLATES conventions
	badGeneratedCode := `package main

import "fmt"

const APIKey = "sk-1234567890abcdef"  // Hardcoded secret - VIOLATION!

func ProcessData(data string) {
	fmt.Println(APIKey)
}
`
	badCodePath := filepath.Join(testDir, "generated_bad.go")
	err = os.WriteFile(badCodePath, []byte(badGeneratedCode), 0644)
	require.NoError(t, err)
	t.Logf("✓ Generated bad code: %s", badCodePath)

	// Case B: Generated code that FOLLOWS conventions
	goodGeneratedCode := `package main

import (
	"fmt"
	"os"
)

// ProcessData processes the given data string according to security guidelines.
// It uses environment variables for sensitive configuration.
func ProcessData(data string) error {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return fmt.Errorf("API_KEY not set")
	}

	fmt.Println(data)
	return nil
}
`
	goodCodePath := filepath.Join(testDir, "generated_good.go")
	err = os.WriteFile(goodCodePath, []byte(goodGeneratedCode), 0644)
	require.NoError(t, err)
	t.Logf("✓ Generated good code: %s", goodCodePath)

	// ========== STEP 4: Validate generated code ==========
	t.Log("STEP 4: Validating generated code against conventions")

	llmValidator := validator.NewValidator(&convertedPolicy, false)
	llmValidator.SetLLMProvider(provider)
	defer llmValidator.Close()

	// Validate BAD code
	t.Log("STEP 4a: Validating BAD code (should find violations)")
	badChanges := []git.Change{
		{
			FilePath: badCodePath,
			Diff:     badGeneratedCode,
		},
	}

	badResult, err := llmValidator.ValidateChanges(ctx, badChanges)
	require.NoError(t, err, "Validation should not error")

	t.Logf("✓ Validation completed: checked=%d, violations=%d",
		badResult.Checked, len(badResult.Violations))

	// Should find violations in bad code
	assert.Greater(t, len(badResult.Violations), 0, "Should detect violations in bad code")

	for i, v := range badResult.Violations {
		t.Logf("  Violation %d: [%s] %s - %s", i+1, v.Severity, v.RuleID, v.Message)
	}

	// Verify specific violations
	foundHardcodedSecret := false
	for _, v := range badResult.Violations {
		if v.Severity == "error" {
			foundHardcodedSecret = true
			t.Logf("✓ Detected hardcoded secret violation")
		}
	}
	assert.True(t, foundHardcodedSecret, "Should detect hardcoded API key")

	// Validate GOOD code
	t.Log("STEP 4b: Validating GOOD code (should pass or have fewer violations)")
	goodChanges := []git.Change{
		{
			FilePath: goodCodePath,
			Diff:     goodGeneratedCode,
		},
	}

	goodResult, err := llmValidator.ValidateChanges(ctx, goodChanges)
	require.NoError(t, err)

	t.Logf("✓ Validation completed: checked=%d, violations=%d",
		goodResult.Checked, len(goodResult.Violations))

	// Good code should have significantly fewer violations
	assert.Less(t, len(goodResult.Violations), len(badResult.Violations),
		"Good code should have fewer violations than bad code")

	if len(goodResult.Violations) == 0 {
		t.Log("✓ Good code passed all checks!")
	} else {
		t.Logf("⚠ Good code has %d minor violations:", len(goodResult.Violations))
		for i, v := range goodResult.Violations {
			t.Logf("  Violation %d: [%s] %s", i+1, v.Severity, v.Message)
		}
	}

	// ========== VERIFICATION: Complete workflow success ==========
	t.Log("========== WORKFLOW SUMMARY ==========")
	t.Logf("✓ Step 1: User policy created (%d rules)", len(userPolicy.Rules))
	t.Logf("✓ Step 2: Converted to structured policy (%d rules)", len(convertedPolicy.Rules))
	t.Logf("✓ Step 3: MCP query returned %d security rules", len(securityRules))
	t.Logf("✓ Step 4a: Bad code validation found %d violations", len(badResult.Violations))
	t.Logf("✓ Step 4b: Good code validation found %d violations", len(goodResult.Violations))
	t.Log("=====================================")
}

// TestE2E_MCPToolIntegration tests MCP tool interactions
func TestE2E_MCPToolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// Create a policy with multiple categories
	policy := &schema.CodePolicy{
		Version: "1.0.0",
		Rules: []schema.PolicyRule{
			{
				ID:       "SEC-001",
				Category: "security",
				Severity: "error",
				Message:  "No hardcoded secrets",
			},
			{
				ID:       "SEC-002",
				Category: "security",
				Severity: "error",
				Message:  "No SQL injection",
			},
			{
				ID:       "ARCH-001",
				Category: "architecture",
				Severity: "warning",
				Message:  "Use repository pattern",
			},
			{
				ID:       "DOC-001",
				Category: "documentation",
				Severity: "warning",
				Message:  "Document exported functions",
			},
		},
	}

	// Test MCP tool: get_conventions_by_category
	t.Run("get_security_conventions", func(t *testing.T) {
		securityRules := filterRulesByCategory(policy.Rules, "security")
		assert.Equal(t, 2, len(securityRules))
		assert.Equal(t, "SEC-001", securityRules[0].ID)
		assert.Equal(t, "SEC-002", securityRules[1].ID)
	})

	t.Run("get_architecture_conventions", func(t *testing.T) {
		archRules := filterRulesByCategory(policy.Rules, "architecture")
		assert.Equal(t, 1, len(archRules))
		assert.Equal(t, "ARCH-001", archRules[0].ID)
	})

	t.Run("get_all_error_level_conventions", func(t *testing.T) {
		errorRules := filterRulesBySeverity(policy.Rules, "error")
		assert.Equal(t, 2, len(errorRules))
	})
}

// TestE2E_CodeGenerationFeedbackLoop tests the feedback loop:
// Generate code -> Validate -> Fix violations -> Validate again
func TestE2E_CodeGenerationFeedbackLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	policy := &schema.CodePolicy{
		Version: "1.0.0",
		Rules: []schema.PolicyRule{
			{
				ID:       "SEC-001",
				Enabled:  true,
				Category: "security",
				Severity: "error",
				Message:  "No hardcoded API keys",
				Desc:     "API keys should not be hardcoded in source code",
				Check: map[string]any{
					"engine": "llm-validator",
					"desc":   "API keys should not be hardcoded in source code",
				},
			},
		},
	}

	cfg := llm.LoadConfig()
	provider, err := llm.New(cfg)
	require.NoError(t, err, "LLM provider creation should succeed")
	v := validator.NewValidator(policy, false)
	v.SetLLMProvider(provider)
	defer v.Close()
	ctx := context.Background()

	// Iteration 1: Bad code
	t.Log("Iteration 1: Validating initial code with violations")
	iteration1 := `+const APIKey = "sk-test123"`

	result1, err := v.ValidateChanges(ctx, []git.Change{
		{FilePath: "test.go", Diff: iteration1},
	})
	require.NoError(t, err)

	violations1 := len(result1.Violations)
	t.Logf("Iteration 1: %d violations found", violations1)
	assert.Greater(t, violations1, 0, "Should find violations in iteration 1")

	// Iteration 2: Fixed code (simulating LLM fixing the issue)
	t.Log("Iteration 2: Validating fixed code")
	iteration2 := `+apiKey := os.Getenv("API_KEY")`

	result2, err := v.ValidateChanges(ctx, []git.Change{
		{FilePath: "test.go", Diff: iteration2},
	})
	require.NoError(t, err)

	violations2 := len(result2.Violations)
	t.Logf("Iteration 2: %d violations found", violations2)

	// Fixed code should have fewer violations
	assert.Less(t, violations2, violations1, "Fixed code should have fewer violations")
	t.Logf("✓ Feedback loop successful: %d -> %d violations", violations1, violations2)
}

// Helper functions

func filterRulesByCategory(rules []schema.PolicyRule, category string) []schema.PolicyRule {
	var filtered []schema.PolicyRule
	for _, rule := range rules {
		if rule.Category == category {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func filterRulesBySeverity(rules []schema.PolicyRule, severity string) []schema.PolicyRule {
	var filtered []schema.PolicyRule
	for _, rule := range rules {
		if rule.Severity == severity {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}
