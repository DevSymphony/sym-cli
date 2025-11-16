package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// ValidationResult represents the result of validating changes
type ValidationResult struct {
	Violations []Violation
	Checked    int
	Passed     int
	Failed     int
}

// LLMValidator validates code changes against LLM-based rules.
// This validator is specifically for Git diff validation.
// For regular file validation, use Validator which orchestrates all engines including LLM.
type LLMValidator struct {
	client    *llm.Client
	policy    *schema.CodePolicy
	validator *Validator
}

// NewLLMValidator creates a new LLM validator
func NewLLMValidator(client *llm.Client, policy *schema.CodePolicy) *LLMValidator {
	return &LLMValidator{
		client:    client,
		policy:    policy,
		validator: NewValidator(policy, false), // Use main validator for orchestration
	}
}

// Validate validates git changes against LLM-based rules.
// This method is for diff-based validation (pre-commit hooks, PR validation).
// For regular file validation, use validator.Validate() which orchestrates all engines.
func (v *LLMValidator) Validate(ctx context.Context, changes []GitChange) (*ValidationResult, error) {
	result := &ValidationResult{
		Violations: make([]Violation, 0),
	}

	// Filter rules that use llm-validator engine
	llmRules := v.filterLLMRules()
	if len(llmRules) == 0 {
		return result, nil
	}

	// Check each change against LLM rules
	for _, change := range changes {
		if change.Status == "D" {
			continue // Skip deleted files
		}

		addedLines := ExtractAddedLines(change.Diff)
		// If no git diff format detected, treat entire diff as code to validate
		if len(addedLines) == 0 && strings.TrimSpace(change.Diff) != "" {
			addedLines = strings.Split(change.Diff, "\n")
		}

		if len(addedLines) == 0 {
			continue
		}

		// Validate against each LLM rule
		for _, rule := range llmRules {
			result.Checked++

			violation, err := v.CheckRule(ctx, change, addedLines, rule)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to check rule %s: %v\n", rule.ID, err)
				continue
			}

			if violation != nil {
				result.Failed++
				result.Violations = append(result.Violations, *violation)
			} else {
				result.Passed++
			}
		}
	}

	return result, nil
}

// filterLLMRules filters rules that use llm-validator engine
func (v *LLMValidator) filterLLMRules() []schema.PolicyRule {
	llmRules := make([]schema.PolicyRule, 0)

	for _, rule := range v.policy.Rules {
		if !rule.Enabled {
			continue
		}

		engine, ok := rule.Check["engine"].(string)
		if ok && engine == "llm-validator" {
			llmRules = append(llmRules, rule)
		}
	}

	return llmRules
}

// CheckRule checks if code violates a specific rule using LLM
// This is the single source of truth for LLM-based validation logic
func (v *LLMValidator) CheckRule(ctx context.Context, change GitChange, addedLines []string, rule schema.PolicyRule) (*Violation, error) {
	// Build prompt for LLM
	systemPrompt := `You are a code reviewer. Check if the code changes violate the given coding convention.

Respond with JSON only:
{
  "violates": true/false,
  "description": "explanation of violation if any",
  "suggestion": "how to fix it if violated"
}`

	codeSnippet := strings.Join(addedLines, "\n")
	userPrompt := fmt.Sprintf(`File: %s

Coding Convention:
%s

Code Changes:
%s

Does this code violate the convention?`, change.FilePath, rule.Desc, codeSnippet)

	// Call LLM
	response, err := v.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	result := parseValidationResponse(response)
	if !result.Violates {
		return nil, nil
	}

	message := result.Description
	if result.Suggestion != "" {
		message += fmt.Sprintf(" | Suggestion: %s", result.Suggestion)
	}

	return &Violation{
		RuleID:   rule.ID,
		Severity: rule.Severity,
		Message:  message,
		File:     change.FilePath,
	}, nil
}

type validationResponse struct {
	Violates    bool
	Description string
	Suggestion  string
}

func parseValidationResponse(response string) validationResponse {
	// Default to no violation
	result := validationResponse{
		Violates:    false,
		Description: "",
		Suggestion:  "",
	}

	lower := strings.ToLower(response)

	// Check if no violation
	if strings.Contains(lower, `"violates": false`) ||
		strings.Contains(lower, `"violates":false`) ||
		strings.Contains(lower, "does not violate") {
		return result
	}

	// Check if violates
	if strings.Contains(lower, `"violates": true`) ||
		strings.Contains(lower, `"violates":true`) {
		result.Violates = true

		// Extract description
		if desc := extractJSONField(response, "description"); desc != "" {
			result.Description = desc
		} else {
			result.Description = "Rule violation detected"
		}

		// Extract suggestion
		if sugg := extractJSONField(response, "suggestion"); sugg != "" {
			result.Suggestion = sugg
		}
	}

	return result
}

func extractJSONField(response, field string) string {
	// Look for "field": "value"
	key := fmt.Sprintf(`"%s"`, field)
	idx := strings.Index(response, key)
	if idx == -1 {
		return ""
	}

	// Find : after field name
	colonIdx := strings.Index(response[idx:], ":") + idx
	if colonIdx <= idx {
		return ""
	}

	// Find opening quote
	openIdx := strings.Index(response[colonIdx:], `"`) + colonIdx
	if openIdx <= colonIdx {
		return ""
	}

	// Find closing quote (skip escaped quotes)
	closeIdx := openIdx + 1
	for closeIdx < len(response) {
		if response[closeIdx] == '"' && (closeIdx == openIdx+1 || response[closeIdx-1] != '\\') {
			return response[openIdx+1 : closeIdx]
		}
		closeIdx++
	}

	return ""
}
