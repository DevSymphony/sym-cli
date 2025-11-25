package linters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// PrettierLinterConverter converts rules to Prettier configuration
type PrettierLinterConverter struct{}

// NewPrettierLinterConverter creates a new Prettier converter
func NewPrettierLinterConverter() *PrettierLinterConverter {
	return &PrettierLinterConverter{}
}

// Name returns the linter name
func (c *PrettierLinterConverter) Name() string {
	return "prettier"
}

// SupportedLanguages returns supported languages
func (c *PrettierLinterConverter) SupportedLanguages() []string {
	return []string{"javascript", "js", "typescript", "ts", "jsx", "tsx"}
}

// ConvertRules converts formatting rules to Prettier config using LLM
func (c *PrettierLinterConverter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Start with default Prettier configuration
	prettierConfig := map[string]interface{}{
		"semi":          true,
		"singleQuote":   false,
		"tabWidth":      2,
		"useTabs":       false,
		"trailingComma": "es5",
		"printWidth":    80,
		"arrowParens":   "always",
	}

	// Use LLM to infer settings from rules
	for _, rule := range rules {
		config, err := c.convertSingleRule(ctx, rule, llmClient)
		if err != nil {
			continue // Skip rules that cannot be converted
		}

		// Merge LLM-generated config
		for key, value := range config {
			prettierConfig[key] = value
		}
	}

	content, err := json.MarshalIndent(prettierConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &LinterConfig{
		Filename: ".prettierrc",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertSingleRule converts a single user rule to Prettier config using LLM
func (c *PrettierLinterConverter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (map[string]interface{}, error) {
	// Build list of valid Prettier options for the prompt
	validOptions := GetPrettierOptionNames()
	validOptionsStr := strings.Join(validOptions, ", ")

	systemPrompt := fmt.Sprintf(`You are a Prettier configuration expert. Convert natural language formatting rules to Prettier configuration options.

IMPORTANT: You MUST ONLY use options from this exact list of valid Prettier options:
%s

Return ONLY a JSON object (no markdown fences) with Prettier options.
If the rule cannot be expressed with Prettier options, return empty object: {}

CRITICAL RULES:
1. ONLY use option names from the list above
2. Do NOT invent new options
3. If no option can enforce this rule, return {}

Examples:

Input: "Use single quotes for strings"
Output:
{
  "singleQuote": true
}

Input: "No semicolons"
Output:
{
  "semi": false
}

Input: "Use 4 spaces for indentation"
Output:
{
  "tabWidth": 4,
  "useTabs": false
}

Input: "Maximum line length is 120 characters"
Output:
{
  "printWidth": 120
}

Input: "Sort imports alphabetically"
Output:
{}
(Reason: No native Prettier option for this)`, validOptionsStr)

	userPrompt := fmt.Sprintf("Convert this rule to Prettier configuration:\n\n%s", rule.Say)

	// Call LLM with minimal reasoning (fast, simple conversion task)
	response, err := llmClient.CompleteMinimal(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(response), &config); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// VALIDATION: Filter out invalid options
	validConfig := make(map[string]interface{})
	for key, value := range config {
		validation := ValidatePrettierOption(key, value)
		if validation.Valid {
			validConfig[key] = value
		} else {
			fmt.Printf("⚠️  Invalid Prettier option '%s': %s\n", key, validation.Message)
		}
	}

	return validConfig, nil
}

// TSCLinterConverter converts rules to TypeScript compiler configuration
type TSCLinterConverter struct{}

// NewTSCLinterConverter creates a new TSC converter
func NewTSCLinterConverter() *TSCLinterConverter {
	return &TSCLinterConverter{}
}

// Name returns the linter name
func (c *TSCLinterConverter) Name() string {
	return "tsc"
}

// SupportedLanguages returns supported languages
func (c *TSCLinterConverter) SupportedLanguages() []string {
	return []string{"typescript", "ts", "tsx"}
}

// ConvertRules converts type-checking rules to tsconfig.json using LLM
func (c *TSCLinterConverter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Start with strict TypeScript configuration
	tsConfig := map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"target":                           "ES2020",
			"module":                           "commonjs",
			"lib":                              []string{"ES2020"},
			"strict":                           true,
			"esModuleInterop":                  true,
			"skipLibCheck":                     true,
			"forceConsistentCasingInFileNames": true,
			"resolveJsonModule":                true,
			"moduleResolution":                 "node",
			"noImplicitAny":                    true,
			"strictNullChecks":                 true,
			"strictFunctionTypes":              true,
			"noUnusedLocals":                   false,
			"noUnusedParameters":               false,
		},
	}

	compilerOpts := tsConfig["compilerOptions"].(map[string]interface{})

	// Use LLM to infer settings from rules
	for _, rule := range rules {
		config, err := c.convertSingleRule(ctx, rule, llmClient)
		if err != nil {
			continue // Skip rules that cannot be converted
		}

		// Merge LLM-generated compiler options
		for key, value := range config {
			compilerOpts[key] = value
		}
	}

	content, err := json.MarshalIndent(tsConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &LinterConfig{
		Filename: "tsconfig.json",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertSingleRule converts a single user rule to TypeScript compiler option using LLM
func (c *TSCLinterConverter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (map[string]interface{}, error) {
	// Build list of valid TSC options for the prompt
	validOptions := GetTSCOptionNames()
	validOptionsStr := strings.Join(validOptions, ", ")

	systemPrompt := fmt.Sprintf(`You are a TypeScript compiler configuration expert. Convert natural language type-checking rules to tsconfig.json compiler options.

IMPORTANT: You MUST ONLY use options from this exact list of valid TypeScript compiler options:
%s

Return ONLY a JSON object (no markdown fences) with TypeScript compiler options.
If the rule cannot be expressed with TypeScript compiler options, return empty object: {}

CRITICAL RULES:
1. ONLY use option names from the list above
2. Do NOT invent new options
3. If no option can enforce this rule, return {}

Examples:

Input: "No implicit any types allowed"
Output:
{
  "noImplicitAny": true
}

Input: "Check for null and undefined strictly"
Output:
{
  "strictNullChecks": true
}

Input: "Report unused variables"
Output:
{
  "noUnusedLocals": true,
  "noUnusedParameters": true
}

Input: "Enable all strict type checks"
Output:
{
  "strict": true
}

Input: "Functions must have return type annotations"
Output:
{}
(Reason: No native TSC option for this - requires plugin)`, validOptionsStr)

	userPrompt := fmt.Sprintf("Convert this rule to TypeScript compiler configuration:\n\n%s", rule.Say)

	// Call LLM with minimal reasoning (fast, simple conversion task)
	response, err := llmClient.CompleteMinimal(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(response), &config); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// VALIDATION: Filter out invalid options
	validConfig := make(map[string]interface{})
	for key, value := range config {
		validation := ValidateTSCOption(key, value)
		if validation.Valid {
			validConfig[key] = value
		} else {
			fmt.Printf("⚠️  Invalid TSC option '%s': %s\n", key, validation.Message)
		}
	}

	return validConfig, nil
}
