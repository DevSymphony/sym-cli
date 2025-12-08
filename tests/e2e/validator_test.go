package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/git"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ValidatorWithPolicy tests the full flow of LLM validator
func TestE2E_ValidatorWithPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Skip if no API key (this is an integration test)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping E2E test")
	}

	// Load policy
	policy, err := loadPolicy(".sym/code-policy.json")
	require.NoError(t, err, "Failed to load policy")
	require.NotEmpty(t, policy.Rules, "Policy should have rules")

	// Create LLM provider
	cfg := llm.LoadConfig()
	provider, err := llm.New(cfg)
	require.NoError(t, err, "LLM provider creation should succeed")

	// Create validator
	v := validator.NewValidator(policy, false)
	v.SetLLMProvider(provider)
	defer v.Close()

	// Create a test change (simulating git diff output)
	changes := []git.Change{
		{
			FilePath: "tests/scenario/bad_code.go",
			Diff:     `+const APIKey = "sk-1234567890abcdefghijklmnopqrstuvwxyz"`,
		},
	}

	// Run validation
	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	// Assertions
	require.NoError(t, err, "Validation should not error")
	assert.NotNil(t, result)
	assert.Greater(t, result.Checked, 0, "Should have checked some rules")

	// We expect violations for hardcoded API key
	assert.Greater(t, len(result.Violations), 0, "Should find violations in bad code")

	// Check that we found the hardcoded API key violation
	foundAPIKeyViolation := false
	for _, v := range result.Violations {
		if v.Severity == "error" {
			foundAPIKeyViolation = true
			t.Logf("Found violation: %s - %s", v.RuleID, v.Message)
		}
	}
	assert.True(t, foundAPIKeyViolation, "Should detect hardcoded API key")
}

// TestE2E_ValidatorWithGoodCode tests validation against compliant code
func TestE2E_ValidatorWithGoodCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping E2E test")
	}

	// Load policy
	policy, err := loadPolicy(".sym/code-policy.json")
	require.NoError(t, err)

	// Create LLM provider
	cfg := llm.LoadConfig()
	provider, err := llm.New(cfg)
	require.NoError(t, err, "LLM provider creation should succeed")

	// Create validator
	v := validator.NewValidator(policy, false)
	v.SetLLMProvider(provider)
	defer v.Close()

	// Create a test change with good code
	changes := []git.Change{
		{
			FilePath: "tests/scenario/good_code.go",
			Diff:     `+var APIKey = os.Getenv("OPENAI_API_KEY")`,
		},
	}

	// Run validation
	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Good code should have fewer or no violations
	t.Logf("Violations found: %d", len(result.Violations))
	// Note: LLM might still flag some issues, so we just log the count
}

// TestE2E_GitChangeExtraction tests git diff extraction
// Note: extractAddedLines is now internal, testing via ValidateChanges
func TestE2E_GitChangeExtraction(t *testing.T) {
	// This test verifies that git diff format is correctly processed
	// by the validator (internally uses extractAddedLines)
	t.Skip("extractAddedLines is now internal - tested via ValidateChanges")
}

// TestE2E_PolicyParsing tests policy file parsing
func TestE2E_PolicyParsing(t *testing.T) {
	policyPath := ".sym/code-policy.json"

	// Skip if policy file doesn't exist (e.g., in CI environment)
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		t.Skipf("Policy file not found: %s (skipping in CI)", policyPath)
	}

	policy, err := loadPolicy(policyPath)
	require.NoError(t, err, "Should parse policy file")

	// Verify policy structure
	assert.Equal(t, "1.0.0", policy.Version)
	assert.NotEmpty(t, policy.Rules)
	assert.Greater(t, len(policy.Rules), 5, "Should have multiple rules")

	// Check for specific rules
	hasSecurityRule := false
	hasArchitectureRule := false

	for _, rule := range policy.Rules {
		if rule.Category == "security" {
			hasSecurityRule = true
		}
		if rule.Category == "architecture" {
			hasArchitectureRule = true
		}
	}

	assert.True(t, hasSecurityRule, "Should have security rules")
	assert.True(t, hasArchitectureRule, "Should have architecture rules")
}

// TestE2E_ValidatorFilter tests that only appropriate rules are checked
func TestE2E_ValidatorFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping E2E test")
	}

	policy, err := loadPolicy(".sym/code-policy.json")
	require.NoError(t, err)

	// Create LLM provider
	cfg := llm.LoadConfig()
	provider, err := llm.New(cfg)
	require.NoError(t, err, "LLM provider creation should succeed")

	// Create validator
	v := validator.NewValidator(policy, false)
	v.SetLLMProvider(provider)
	defer v.Close()

	// Test with Go file
	changes := []git.Change{
		{
			FilePath: "test.go",
			Diff:     "+const x = 1",
		},
	}

	ctx := context.Background()
	result, err := v.ValidateChanges(ctx, changes)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should have checked rules applicable to Go
	assert.Greater(t, result.Checked, 0, "Should check Go rules")
}

// Helper function to load policy
func loadPolicy(path string) (*schema.CodePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var policy schema.CodePolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}
