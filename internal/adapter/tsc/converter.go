package tsc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts rules to TypeScript compiler configuration using LLM
type Converter struct{}

// NewConverter creates a new TSC converter
func NewConverter() *Converter {
	return &Converter{}
}

// Name returns the linter name
func (c *Converter) Name() string {
	return "tsc"
}

// SupportedLanguages returns supported languages
func (c *Converter) SupportedLanguages() []string {
	return []string{"typescript", "ts", "tsx"}
}

// GetLLMDescription returns a description of TSC's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `TypeScript type checking ONLY (strict modes, noImplicitAny, strictNullChecks, type inference)
  - CAN: Type strictness, null checks, unused variable/parameter errors, implicit return checks
  - CANNOT: Code formatting, naming conventions, runtime behavior, business logic`
}

// ConvertRules converts type-checking rules to tsconfig.json using LLM
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*adapter.LinterConfig, error) {
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

	return &adapter.LinterConfig{
		Filename: "tsconfig.json",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertSingleRule converts a single user rule to TypeScript compiler option using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (map[string]interface{}, error) {
	systemPrompt := `You are a TypeScript compiler configuration expert. Convert natural language type-checking rules to tsconfig.json compiler options.

Return ONLY a JSON object (no markdown fences) with TypeScript compiler options.

Available TypeScript compiler options:
- strict: true/false (enable all strict checks)
- noImplicitAny: true/false (error on implicit any)
- strictNullChecks: true/false (strict null checking)
- strictFunctionTypes: true/false (strict function types)
- strictBindCallApply: true/false (strict bind/call/apply)
- noUnusedLocals: true/false (error on unused locals)
- noUnusedParameters: true/false (error on unused parameters)
- noImplicitReturns: true/false (error on implicit returns)
- noFallthroughCasesInSwitch: true/false (error on fallthrough)
- noUncheckedIndexedAccess: true/false (undefined in index signatures)
- allowUnreachableCode: true/false (allow unreachable code)
- allowUnusedLabels: true/false (allow unused labels)

If the rule is not about TypeScript type-checking, return empty object: {}

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
}`

	userPrompt := fmt.Sprintf("Convert this rule to TypeScript compiler configuration:\n\n%s", rule.Say)

	// Call LLM
	response, err := llmClient.Complete(ctx, systemPrompt, userPrompt)
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

	return config, nil
}
