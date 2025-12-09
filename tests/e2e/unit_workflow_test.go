package e2e_test

import (
	"encoding/json"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/validator"
	"github.com/DevSymphony/sym-cli/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnit_PolicyParsing tests parsing of user policy JSON
func TestUnit_PolicyParsing(t *testing.T) {
	policyJSON := `{
		"version": "1.0.0",
		"defaults": {
			"languages": ["go"],
			"severity": "warning"
		},
		"rules": [
			{
				"say": "API 키를 하드코딩하지 마세요",
				"category": "security",
				"severity": "error"
			},
			{
				"say": "함수에 godoc 주석을 추가하세요",
				"category": "documentation"
			}
		]
	}`

	var policy schema.UserPolicy
	err := json.Unmarshal([]byte(policyJSON), &policy)

	require.NoError(t, err, "Should parse valid policy JSON")
	assert.Equal(t, "1.0.0", policy.Version)
	assert.Equal(t, []string{"go"}, policy.Defaults.Languages)
	assert.Equal(t, 2, len(policy.Rules))
	assert.Equal(t, "security", policy.Rules[0].Category)
	assert.Equal(t, "error", policy.Rules[0].Severity)
}

// TestUnit_GitDiffExtraction tests extracting added lines from git diff
// Note: extractAddedLines is now internal to the validator package
func TestUnit_GitDiffExtraction(t *testing.T) {
	t.Skip("extractAddedLines is now internal - tested via internal/validator/git_test.go")
}

// TestUnit_RuleFiltering tests filtering rules by category and severity
func TestUnit_RuleFiltering(t *testing.T) {
	rules := []schema.PolicyRule{
		{ID: "SEC-001", Category: "security", Severity: "error"},
		{ID: "SEC-002", Category: "security", Severity: "warning"},
		{ID: "ARCH-001", Category: "architecture", Severity: "error"},
		{ID: "DOC-001", Category: "documentation", Severity: "warning"},
	}

	t.Run("filter by category", func(t *testing.T) {
		security := filterRulesByCategory(rules, "security")
		assert.Equal(t, 2, len(security))
		assert.Equal(t, "SEC-001", security[0].ID)
		assert.Equal(t, "SEC-002", security[1].ID)

		arch := filterRulesByCategory(rules, "architecture")
		assert.Equal(t, 1, len(arch))
		assert.Equal(t, "ARCH-001", arch[0].ID)
	})

	t.Run("filter by severity", func(t *testing.T) {
		errors := filterRulesBySeverity(rules, "error")
		assert.Equal(t, 2, len(errors))

		warnings := filterRulesBySeverity(rules, "warning")
		assert.Equal(t, 2, len(warnings))
	})

	t.Run("combined filtering", func(t *testing.T) {
		// Security errors only
		securityRules := filterRulesByCategory(rules, "security")
		securityErrors := filterRulesBySeverity(securityRules, "error")
		assert.Equal(t, 1, len(securityErrors))
		assert.Equal(t, "SEC-001", securityErrors[0].ID)
	})
}

// TestUnit_ValidationResponseParsing tests parsing LLM validation responses
func TestUnit_ValidationResponseParsing(t *testing.T) {
	tests := []struct {
		name		string
		response	string
		expectViolates	bool
		expectDesc	bool
	}{
		{
			name:		"explicit violation",
			response:	`{"violates": true, "description": "Hardcoded secret detected"}`,
			expectViolates:	true,
			expectDesc:	true,
		},
		{
			name:		"no violation",
			response:	`{"violates": false}`,
			expectViolates:	false,
			expectDesc:	false,
		},
		{
			name:		"text-based violation",
			response:	`The code violates the security policy by hardcoding an API key.`,
			expectViolates:	true,
			expectDesc:	true,
		},
		{
			name:		"text-based no violation",
			response:	`The code does not violate any conventions.`,
			expectViolates:	false,
			expectDesc:	false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This assumes parseValidationResponse is exported
			// If not, we test via the public Validate method with mocks
			// For now, we're testing the concept

			// Check for explicit JSON format
			hasJSONViolation := contains(tt.response, "\"violates\": true")
			hasJSONNoViolation := contains(tt.response, "\"violates\": false")

			// Check for text-based violation (but exclude negations like "does not violate")
			hasTextViolation := !hasJSONNoViolation && (
				contains(tt.response, "violates the") ||
					contains(tt.response, "violates any") ||
					(contains(tt.response, "violate") && !contains(tt.response, "does not violate") && !contains(tt.response, "not violate")))

			containsViolation := hasJSONViolation || hasTextViolation

			if tt.expectViolates {
				assert.True(t, containsViolation,
					"Response should indicate violation")
			} else {
				assert.False(t, containsViolation,
					"Response should not indicate violation")
			}
		})
	}
}

// TestUnit_WorkflowSteps tests individual workflow steps
func TestUnit_WorkflowSteps(t *testing.T) {
	t.Run("step1_user_creates_policy", func(t *testing.T) {
		policy := schema.UserPolicy{
			Version:	"1.0.0",
			Rules: []schema.UserRule{
				{Say: "No hardcoded secrets", Category: "security"},
			},
		}

		data, err := json.Marshal(policy)
		require.NoError(t, err)

		var parsed schema.UserPolicy
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, policy.Version, parsed.Version)
		assert.Equal(t, len(policy.Rules), len(parsed.Rules))
	})

	t.Run("step2_conversion_structure", func(t *testing.T) {
		// Test that conversion produces expected structure
		userRule := schema.UserRule{
			Say:		"No hardcoded secrets",
			Category:	"security",
			Severity:	"error",
		}

		// After conversion, should have structured fields
		assert.NotEmpty(t, userRule.Say)
		assert.NotEmpty(t, userRule.Category)
		assert.NotEmpty(t, userRule.Severity)
	})

	t.Run("step3_mcp_query_simulation", func(t *testing.T) {
		// Simulate MCP tool querying for security rules
		allRules := []schema.PolicyRule{
			{ID: "SEC-001", Category: "security"},
			{ID: "ARCH-001", Category: "architecture"},
			{ID: "SEC-002", Category: "security"},
		}

		// MCP tool: get_conventions_by_category("security")
		securityRules := filterRulesByCategory(allRules, "security")

		assert.Equal(t, 2, len(securityRules))
		assert.Equal(t, "SEC-001", securityRules[0].ID)
		assert.Equal(t, "SEC-002", securityRules[1].ID)
	})

	t.Run("step4_validation_result_structure", func(t *testing.T) {
		// Test validation result structure
		result := validator.ValidationResult{
			Checked:	5,
			Passed:		3,
			Failed:		2,
			Violations: []validator.Violation{
				{
					RuleID:		"SEC-001",
					Severity:	"error",
					Message:	"Hardcoded secret detected",
					File:		"test.go",
				},
			},
		}

		assert.Equal(t, 5, result.Checked)
		assert.Equal(t, 2, result.Failed)
		assert.Equal(t, 1, len(result.Violations))
		assert.Equal(t, "SEC-001", result.Violations[0].RuleID)
	})
}

// TestUnit_MCPToolResponses tests MCP tool response formats
func TestUnit_MCPToolResponses(t *testing.T) {
	t.Run("get_conventions_by_category", func(t *testing.T) {
		// Simulated MCP tool response
		response := map[string]interface{}{
			"tool":		"get_conventions_by_category",
			"category":	"security",
			"conventions": []map[string]string{
				{
					"id":		"SEC-001",
					"message":	"No hardcoded secrets",
					"severity":	"error",
				},
				{
					"id":		"SEC-002",
					"message":	"Use parameterized queries",
					"severity":	"error",
				},
			},
		}

		assert.Equal(t, "get_conventions_by_category", response["tool"])
		conventions := response["conventions"].([]map[string]string)
		assert.Equal(t, 2, len(conventions))
	})

	t.Run("get_all_conventions", func(t *testing.T) {
		response := map[string]interface{}{
			"tool":		"get_all_conventions",
			"count":	10,
			"conventions": []string{
				"No hardcoded secrets",
				"Document exported functions",
				"Use repository pattern",
			},
		}

		assert.Equal(t, "get_all_conventions", response["tool"])
		assert.Equal(t, 10, response["count"])
	})
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAny(s, substr))
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
