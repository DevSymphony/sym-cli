package tsc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

// Converter converts rules to TypeScript compiler configuration using LLM
type Converter struct{}

// NewConverter creates a new TSC converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "tsc"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"typescript", "ts", "tsx"}
}

// GetLLMDescription returns a description of TSC's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `TypeScript type checking ONLY (strict modes, noImplicitAny, strictNullChecks, type inference)
  - CAN: Type strictness, null checks, unused variable/parameter errors, implicit return checks
  - CANNOT: Code formatting, naming conventions, runtime behavior, business logic`
}

// GetRoutingHints returns routing rules for LLM to decide when to use TSC
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For TypeScript type checking (noImplicitAny, strictNullChecks) → use tsc",
		"For TypeScript strict mode rules (strict, noUnusedLocals) → use tsc",
		"NEVER use tsc for naming conventions or formatting - use eslint/prettier instead",
	}
}

// tscRuleData holds TSC-specific conversion data (compiler options)
type tscRuleData struct {
	Options map[string]interface{}
}

// ConvertSingleRule converts ONE user rule to TSC compiler option.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by TSC (skip),
//	(nil, error) on actual conversion error.
//
// Note: Concurrency is handled by the main converter.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	config, err := c.convertToTSCOption(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	// Check if LLM returned empty config (rule cannot be enforced by TSC)
	if len(config) == 0 {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data: tscRuleData{
			Options: config,
		},
	}, nil
}

// BuildConfig assembles TypeScript configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Start with base TypeScript configuration
	compilerOpts := map[string]interface{}{
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
	}

	// Merge all rule options
	for _, r := range results {
		data, ok := r.Data.(tscRuleData)
		if !ok {
			continue
		}
		for key, value := range data.Options {
			compilerOpts[key] = value
		}
	}

	tsConfig := map[string]interface{}{
		"compilerOptions": compilerOpts,
	}

	content, err := json.MarshalIndent(tsConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &linter.LinterConfig{
		Filename: "tsconfig.json",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertToTSCOption converts a single user rule to TypeScript compiler option using LLM
func (c *Converter) convertToTSCOption(ctx context.Context, rule schema.UserRule, provider llm.Provider) (map[string]interface{}, error) {
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
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = linter.CleanJSONResponse(response)

	if response == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(response), &config); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	return config, nil
}
