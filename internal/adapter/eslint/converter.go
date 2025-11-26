package eslint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts rules to ESLint configuration using LLM
type Converter struct{}

// NewConverter creates a new ESLint converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "eslint"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"javascript", "js", "typescript", "ts", "jsx", "tsx"}
}

// GetLLMDescription returns a description of ESLint's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `ONLY native ESLint rules (no-console, no-unused-vars, eqeqeq, no-var, camelcase, new-cap, max-len, max-lines, no-eval, etc.)
  - CAN: Simple syntax checks, variable naming, console usage, basic patterns
  - CANNOT: Complex business logic, context-aware rules, file naming, advanced async patterns`
}

// GetRoutingHints returns routing rules for LLM to decide when to use ESLint
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For JavaScript/TypeScript naming rules (camelCase, PascalCase) → use eslint",
		"For JS/TS code quality (unused vars, no-console, no-eval) → use eslint",
		"For JS/TS best practices (eqeqeq, no-var, prefer-const) → use eslint",
	}
}

// ConvertRules converts user rules to ESLint configuration using LLM
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*adapter.LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Convert rules in parallel using goroutines
	type ruleResult struct {
		index    int
		ruleName string
		config   interface{}
		err      error
	}

	results := make(chan ruleResult, len(rules))
	var wg sync.WaitGroup

	// Process each rule in parallel
	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, r schema.UserRule) {
			defer wg.Done()

			ruleName, config, err := c.convertSingleRule(ctx, r, llmClient)
			results <- ruleResult{
				index:    idx,
				ruleName: ruleName,
				config:   config,
				err:      err,
			}
		}(i, rule)
	}

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	eslintRules := make(map[string]interface{})
	var errors []string
	skippedCount := 0

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("Rule %d: %v", result.index+1, result.err))
			fmt.Fprintf(os.Stderr, "⚠️  ESLint rule %d conversion error: %v\n", result.index+1, result.err)
			continue
		}

		if result.ruleName != "" {
			eslintRules[result.ruleName] = result.config
			fmt.Fprintf(os.Stderr, "✓ ESLint rule %d → %s\n", result.index+1, result.ruleName)
		} else {
			skippedCount++
			fmt.Fprintf(os.Stderr, "⊘ ESLint rule %d skipped (cannot be enforced by ESLint)\n", result.index+1)
		}
	}

	if skippedCount > 0 {
		fmt.Fprintf(os.Stderr, "ℹ️  %d rule(s) skipped for ESLint (will use llm-validator)\n", skippedCount)
	}

	if len(eslintRules) == 0 {
		return nil, fmt.Errorf("no rules converted successfully: %v", errors)
	}

	// Build ESLint configuration
	eslintConfig := map[string]interface{}{
		"env": map[string]bool{
			"es2021":  true,
			"node":    true,
			"browser": true,
		},
		"parserOptions": map[string]interface{}{
			"ecmaVersion": "latest",
			"sourceType":  "module",
		},
		"rules": eslintRules,
	}

	// Marshal to JSON
	content, err := json.MarshalIndent(eslintConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &adapter.LinterConfig{
		Filename: ".eslintrc.json",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertSingleRule converts a single user rule to ESLint rule using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (string, interface{}, error) {
	// Build list of valid ESLint rules for the prompt
	validRules := GetESLintRuleNames()
	sort.Strings(validRules)
	validRulesStr := strings.Join(validRules, ", ")

	systemPrompt := fmt.Sprintf(`You are an ESLint configuration expert. Convert natural language coding rules to ESLint rule configurations.

IMPORTANT: You MUST ONLY use rules from this exact list of valid ESLint rules:
%s

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "rule_name": "eslint-rule-name",
  "severity": "error|warn|off",
  "options": {...}
}

CRITICAL RULES:
1. ONLY use rule names from the list above - do NOT invent or guess rule names
2. If no rule from the list can enforce this requirement, return rule_name as empty string ""
3. Do NOT suggest plugin rules (e.g., @typescript-eslint/*, eslint-plugin-*)
4. When in doubt, return empty rule_name - it's better to skip than use wrong rule

Examples:

Input: "No console.log allowed"
Output:
{
  "rule_name": "no-console",
  "severity": "error",
  "options": null
}

Input: "Functions must not exceed 50 lines"
Output:
{
  "rule_name": "max-lines-per-function",
  "severity": "error",
  "options": {"max": 50, "skipBlankLines": true, "skipComments": true}
}

Input: "Use camelCase for variables"
Output:
{
  "rule_name": "camelcase",
  "severity": "error",
  "options": {"properties": "always"}
}

Input: "File names must be kebab-case"
Output:
{
  "rule_name": "",
  "severity": "off",
  "options": null
}
(Reason: No native ESLint rule for file naming)

Input: "No hardcoded API keys"
Output:
{
  "rule_name": "",
  "severity": "off",
  "options": null
}
(Reason: Requires plugin or semantic analysis)`, validRulesStr)

	userPrompt := fmt.Sprintf("Convert this rule to ESLint configuration:\n\n%s", rule.Say)
	if rule.Severity != "" {
		userPrompt += fmt.Sprintf("\nSeverity: %s", rule.Severity)
	}

	// Call LLM with power model + low reasoning
	response, err := llmClient.Request(systemPrompt, userPrompt).WithPower(llm.ReasoningMinimal).Execute(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if response == "" {
		return "", nil, fmt.Errorf("LLM returned empty response")
	}

	var result struct {
		RuleName string      `json:"rule_name"`
		Severity string      `json:"severity"`
		Options  interface{} `json:"options"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	// If rule_name is empty, this rule cannot be converted
	if result.RuleName == "" {
		return "", nil, nil
	}

	// VALIDATION: Check if the rule actually exists in our registry
	validation := ValidateESLintRule(result.RuleName, result.Options)
	if !validation.Valid {
		// Rule doesn't exist - skip it (will be handled by llm-validator)
		fmt.Printf("⚠️  Invalid ESLint rule '%s': %s\n", result.RuleName, validation.Message)
		return "", nil, nil
	}

	// Map user severity to ESLint severity if needed
	severity := mapSeverity(rule.Severity)
	if severity == "" {
		severity = result.Severity
	}

	// Build rule configuration using format helper for special rules
	config := formatESLintRuleConfig(result.RuleName, severity, result.Options)

	return result.RuleName, config, nil
}

// mapSeverity maps user severity to ESLint severity
func mapSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "error"
	case "warning", "warn":
		return "warn"
	case "info", "off":
		return "off"
	default:
		return "error"
	}
}

// formatESLintRuleConfig formats the rule configuration based on rule-specific requirements.
// Some ESLint rules have non-standard option formats that need special handling.
func formatESLintRuleConfig(ruleName string, severity string, options interface{}) interface{} {
	// Rules that need special formatting
	switch ruleName {
	case "id-match":
		// id-match requires: [severity, pattern, options]
		// where pattern is a string and options is an object
		if opts, ok := options.(map[string]interface{}); ok {
			if pattern, hasPattern := opts["pattern"].(string); hasPattern {
				// Remove pattern from options since it's a separate argument
				remainingOpts := make(map[string]interface{})
				for k, v := range opts {
					if k != "pattern" {
						remainingOpts[k] = v
					}
				}
				if len(remainingOpts) > 0 {
					return []interface{}{severity, pattern, remainingOpts}
				}
				return []interface{}{severity, pattern}
			}
		}

	case "no-restricted-imports":
		// no-restricted-imports can have complex options
		// Keep the default format for now
	}

	// Default format: [severity, options] or just severity
	if options != nil {
		return []interface{}{severity, options}
	}
	return severity
}
