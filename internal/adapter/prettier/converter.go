package prettier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts rules to Prettier configuration using LLM
type Converter struct{}

// NewConverter creates a new Prettier converter
func NewConverter() *Converter {
	return &Converter{}
}

// Name returns the linter name
func (c *Converter) Name() string {
	return "prettier"
}

// SupportedLanguages returns supported languages
func (c *Converter) SupportedLanguages() []string {
	return []string{"javascript", "js", "typescript", "ts", "jsx", "tsx"}
}

// GetLLMDescription returns a description of Prettier's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `Code formatting ONLY (quotes, semicolons, indentation, line length, trailing commas)
  - CAN: String quotes, semicolons, tab width, trailing commas, print width, arrow function parens
  - CANNOT: Code logic, naming conventions, unused variables, type checking`
}

// ConvertRules converts formatting rules to Prettier config using LLM
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*adapter.LinterConfig, error) {
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

	return &adapter.LinterConfig{
		Filename: ".prettierrc",
		Content:  content,
		Format:   "json",
	}, nil
}

// convertSingleRule converts a single user rule to Prettier config using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (map[string]interface{}, error) {
	systemPrompt := `You are a Prettier configuration expert. Convert natural language formatting rules to Prettier configuration options.

Return ONLY a JSON object (no markdown fences) with Prettier options.

Available Prettier options:
- semi: true/false (use semicolons)
- singleQuote: true/false (use single quotes)
- tabWidth: number (spaces per indentation level)
- useTabs: true/false (use tabs instead of spaces)
- trailingComma: "none"/"es5"/"all" (trailing commas)
- printWidth: number (line length)
- arrowParens: "always"/"avoid" (arrow function parentheses)
- bracketSpacing: true/false (spaces in object literals)
- endOfLine: "lf"/"crlf"/"auto"

If the rule is not about formatting, return empty object: {}

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
}`

	userPrompt := fmt.Sprintf("Convert this rule to Prettier configuration:\n\n%s", rule.Say)

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
