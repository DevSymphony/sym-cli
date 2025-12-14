package eslint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

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

// eslintRuleData holds ESLint-specific conversion data
type eslintRuleData struct {
	RuleName string      `json:"ruleName"`
	Config   interface{} `json:"config"`
}

// ConvertSingleRule converts ONE user rule to ESLint rule configuration.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by ESLint (skip),
//	(nil, error) on actual conversion error.
//
// Note: Concurrency is handled by the main converter.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	ruleName, config, err := c.convertToESLintRule(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	// If rule_name is empty, this rule cannot be converted by ESLint
	if ruleName == "" {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data: eslintRuleData{
			RuleName: ruleName,
			Config:   config,
		},
	}, nil
}

// BuildConfig assembles ESLint configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	eslintRules := make(map[string]interface{})
	for _, r := range results {
		data, ok := r.Data.(eslintRuleData)
		if !ok {
			continue
		}
		eslintRules[data.RuleName] = data.Config
	}

	if len(eslintRules) == 0 {
		return nil, nil
	}

	eslintConfig := map[string]interface{}{
		"env": map[string]bool{
			"es2021":  true,
			"node":    true,
			"browser": true,
		},
		"parser": "@typescript-eslint/parser",
		"parserOptions": map[string]interface{}{
			"ecmaVersion": "latest",
			"sourceType":  "module",
		},
		"rules": eslintRules,
	}

	content, err := json.MarshalIndent(eslintConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &linter.LinterConfig{
		Filename: ".eslintrc.json",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertToESLintRule converts a single user rule to ESLint rule using LLM
func (c *Converter) convertToESLintRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (string, interface{}, error) {
	systemPrompt := `You are an ESLint configuration expert. Convert natural language coding rules to ESLint rule configurations.

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "rule_name": "eslint-rule-name",
  "severity": "error|warn|off",
  "options": {...}
}

Available native ESLint rules:
- Console/Debug: no-console, no-debugger, no-alert
- Variables: no-unused-vars, no-undef, no-var, prefer-const
- Naming: camelcase, new-cap, id-length, id-match
- Code Quality: eqeqeq, no-eval, no-implied-eval, no-new-func
- Complexity: complexity, max-depth, max-nested-callbacks
- Length/Size: max-len, max-lines, max-lines-per-function, max-params, max-statements
- Style: indent, quotes, semi, comma-dangle, brace-style
- Imports: no-restricted-imports, no-duplicate-imports
- Best Practices: curly, dot-notation, no-else-return, no-empty, no-empty-function, no-magic-numbers, no-throw-literal, no-useless-return, require-await

CRITICAL RULES:
1. ONLY use native ESLint rules - do NOT invent or guess rule names
2. If no rule can enforce this requirement, return rule_name as empty string ""
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
(Reason: Requires plugin or semantic analysis)`

	userPrompt := fmt.Sprintf("Convert this rule to ESLint configuration:\n\n%s", rule.Say)
	if rule.Severity != "" {
		userPrompt += fmt.Sprintf("\nSeverity: %s", rule.Severity)
	}

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return "", nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = linter.CleanJSONResponse(response)

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
