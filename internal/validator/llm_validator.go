package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// ValidationError represents an error that occurred during validation
type ValidationError struct {
	RuleID  string
	Engine  string
	Message string
}

// ValidationResult represents the result of validating changes
type ValidationResult struct {
	Violations []Violation
	Errors     []ValidationError // Adapter/engine execution errors
	Checked    int
	Passed     int
	Failed     int
}

// LLMValidator validates code changes against LLM-based rules.
// This validator is specifically for Git diff validation.
// For regular file validation, use Validator which orchestrates all engines including LLM.
type LLMValidator struct {
	provider  llm.Provider
	policy    *schema.CodePolicy
	validator *Validator
}

// NewLLMValidator creates a new LLM validator
func NewLLMValidator(provider llm.Provider, policy *schema.CodePolicy) *LLMValidator {
	return &LLMValidator{
		provider:  provider,
		policy:    policy,
		validator: NewValidator(policy, false), // Use main validator for orchestration
	}
}

// Validate validates git changes against LLM-based rules.
// This method is for diff-based validation (pre-commit hooks, PR validation).
// For regular file validation, use validator.Validate() which orchestrates all engines.
// Concurrency is limited to CPU count to prevent CPU spike.
func (v *LLMValidator) Validate(ctx context.Context, changes []GitChange) (*ValidationResult, error) {
	result := &ValidationResult{
		Violations: make([]Violation, 0),
	}

	// Filter rules that use llm-validator engine
	llmRules := v.filterLLMRules()
	if len(llmRules) == 0 {
		return result, nil
	}

	// Check each change against LLM rules using goroutines for parallel processing
	// Limit concurrency to prevent resource exhaustion
	// Use CPU/4, minimum 2, maximum 4 to balance performance and stability
	var wg sync.WaitGroup
	var mu sync.Mutex

	maxConcurrent := runtime.NumCPU() / 4
	if maxConcurrent < 2 {
		maxConcurrent = 2
	}
	if maxConcurrent > 4 {
		maxConcurrent = 4
	}
	sem := make(chan struct{}, maxConcurrent)

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

		// Validate against each LLM rule in parallel with concurrency limit
		for _, rule := range llmRules {
			mu.Lock()
			result.Checked++
			mu.Unlock()

			wg.Add(1)
			go func(ch GitChange, lines []string, r schema.PolicyRule) {
				defer wg.Done()

				// Acquire semaphore
				sem <- struct{}{}
				defer func() { <-sem }()

				violation, err := v.CheckRule(ctx, ch, lines, r)
				if err != nil {
					// Log error but continue
					fmt.Printf("Warning: failed to check rule %s: %v\n", r.ID, err)
					return
				}

				mu.Lock()
				defer mu.Unlock()
				if violation != nil {
					result.Failed++
					result.Violations = append(result.Violations, *violation)
				} else {
					result.Passed++
				}
			}(change, addedLines, rule)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

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
	// Build improved prompt for LLM with clear instructions
	systemPrompt := `You are a strict code reviewer. Your job is to check if code changes violate a specific coding convention.

IMPORTANT INSTRUCTIONS:
1. Be CONSERVATIVE - only report violations when you are CERTAIN the code violates the rule
2. Do NOT report false positives - if unsure, report as NOT violating
3. Consider the context of the code when making your decision
4. Focus ONLY on the specific rule given - do not check other rules

You MUST respond with ONLY a valid JSON object (no markdown, no explanation outside JSON):
{
  "violates": false,
  "confidence": "high",
  "description": "",
  "suggestion": ""
}

JSON Field Definitions:
- violates: boolean - true ONLY if you are certain the code violates the rule
- confidence: "high" | "medium" | "low" - your confidence in the assessment
- description: string - brief explanation if violated (empty string if not violated)
- suggestion: string - how to fix if violated (empty string if not violated)

EXAMPLES:

Rule: "No console.log in production code"
Code: "console.log('debug');"
Response:
{"violates": true, "confidence": "high", "description": "console.log statement found", "suggestion": "Remove console.log or use a proper logging library"}

Rule: "Functions must not exceed 50 lines"
Code: (20 lines of code)
Response:
{"violates": false, "confidence": "high", "description": "", "suggestion": ""}

Rule: "Use const for variables that are never reassigned"
Code: "let x = 5; return x;"
Response:
{"violates": true, "confidence": "high", "description": "Variable 'x' is never reassigned but declared with 'let'", "suggestion": "Change 'let x' to 'const x'"}`

	codeSnippet := strings.Join(addedLines, "\n")

	// Truncate very long code to avoid token limits
	const maxCodeLength = 3000
	if len(codeSnippet) > maxCodeLength {
		codeSnippet = codeSnippet[:maxCodeLength] + "\n... (truncated)"
	}

	userPrompt := fmt.Sprintf(`File: %s

=== RULE TO CHECK ===
%s

=== CODE TO REVIEW ===
%s

Analyze the code and determine if it violates the rule. Respond with JSON only.`, change.FilePath, rule.Desc, codeSnippet)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := v.provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return nil, err
	}

	// Parse response with improved parsing
	result := parseValidationResponse(response)

	// Only report high-confidence violations
	if !result.Violates || result.Confidence == "low" {
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
	Confidence  string
	Description string
	Suggestion  string
}

// jsonValidationResponse is the structure for JSON parsing
type jsonValidationResponse struct {
	Violates    bool   `json:"violates"`
	Confidence  string `json:"confidence"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

func parseValidationResponse(response string) validationResponse {
	// Default to no violation (conservative approach)
	result := validationResponse{
		Violates:    false,
		Confidence:  "low",
		Description: "",
		Suggestion:  "",
	}

	// Clean up response - remove markdown fences if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to find JSON object in response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		// No valid JSON found, return default (no violation)
		return result
	}

	jsonStr := response[startIdx : endIdx+1]

	// Parse JSON properly using encoding/json
	var parsed jsonValidationResponse
	if err := parseJSON(jsonStr, &parsed); err != nil {
		// Fallback to string-based parsing for edge cases
		return parseValidationResponseFallback(response)
	}

	result.Violates = parsed.Violates
	result.Confidence = parsed.Confidence
	result.Description = parsed.Description
	result.Suggestion = parsed.Suggestion

	// Default confidence to "medium" if not specified
	if result.Confidence == "" {
		result.Confidence = "medium"
	}

	// Provide default description if violation detected but no description given
	if result.Violates && result.Description == "" {
		result.Description = "Rule violation detected"
	}

	return result
}

// parseJSON parses JSON string into the target struct using encoding/json
func parseJSON(jsonStr string, target interface{}) error {
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return nil
}

// parseValidationResponseFallback is used when JSON parsing fails
func parseValidationResponseFallback(response string) validationResponse {
	result := validationResponse{
		Violates:    false,
		Confidence:  "low",
		Description: "",
		Suggestion:  "",
	}

	lower := strings.ToLower(response)

	// Check if explicitly no violation
	if strings.Contains(lower, `"violates": false`) ||
		strings.Contains(lower, `"violates":false`) ||
		strings.Contains(lower, "does not violate") {
		return result
	}

	// Check if violates
	if strings.Contains(lower, `"violates": true`) ||
		strings.Contains(lower, `"violates":true`) {
		result.Violates = true
		result.Confidence = "medium" // Lower confidence for fallback parsing

		if desc := extractJSONField(response, "description"); desc != "" {
			result.Description = desc
		} else {
			result.Description = "Rule violation detected"
		}

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
