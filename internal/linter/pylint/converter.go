package pylint

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

// Converter converts rules to Pylint configuration using LLM
type Converter struct{}

// NewConverter creates a new Pylint converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "pylint"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"python", "py"}
}

// GetLLMDescription returns a description of Pylint's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `Python code quality via Pylint (PEP 8 style, docstrings, complexity, error handling, imports)
  - CAN: Naming conventions (snake_case, PascalCase), line length limits, docstring requirements,
         cyclomatic complexity, function arguments/locals limits, import ordering,
         bare except blocks, unused variables/imports, dangerous default values, eval/exec usage
  - CANNOT: Business logic validation, runtime behavior checks, type correctness beyond basic inference,
         async/await pattern validation, complex decorator analysis`
}

// GetRoutingHints returns routing rules for LLM to decide when to use Pylint
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For Python naming rules (snake_case, PascalCase) → use pylint",
		"For Python code quality (docstrings, complexity, unused vars) → use pylint",
		"For Python style (line length, import order) → use pylint",
		"For Python error handling (bare except, broad except) → use pylint",
	}
}

// pylintRuleData holds Pylint-specific conversion data
type pylintRuleData struct {
	Symbol  string
	Options map[string]interface{}
}

// ConvertSingleRule converts ONE user rule to Pylint rule.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by Pylint (skip),
//	(nil, error) on actual conversion error.
//
// Note: Concurrency is handled by the main converter.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	symbol, options, err := c.convertToPylintRule(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	if symbol == "" {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data: pylintRuleData{
			Symbol:  symbol,
			Options: options,
		},
	}, nil
}

// BuildConfig assembles Pylint configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	enabledRules := make([]string, 0)
	options := make(map[string]map[string]interface{})

	for _, r := range results {
		data, ok := r.Data.(pylintRuleData)
		if !ok {
			continue
		}

		enabledRules = append(enabledRules, data.Symbol)

		if len(data.Options) > 0 {
			// Group options by section
			for key, value := range data.Options {
				section := getOptionSection(key)
				if _, ok := options[section]; !ok {
					options[section] = make(map[string]interface{})
				}
				options[section][key] = value
			}
		}
	}

	if len(enabledRules) == 0 {
		return nil, nil
	}

	content := c.generatePylintRC(enabledRules, options)

	return &linter.LinterConfig{
		Filename: ".pylintrc",
		Content:  []byte(content),
		Format:   "ini",
	}, nil
}

// convertToPylintRule converts a single user rule to Pylint rule using LLM
func (c *Converter) convertToPylintRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (string, map[string]interface{}, error) {
	systemPrompt := `You are a Pylint configuration expert. Convert natural language Python coding rules to Pylint rule configurations.

Return ONLY a JSON object (no markdown fences) with this structure:
{
  "symbol": "pylint-rule-symbol",
  "message_id": "C0116",
  "options": {"key": "value", ...}
}

Common Pylint rules:
- Naming: invalid-name (C0103), disallowed-name (C0104)
  Options: variable-rgx, function-rgx, class-rgx, const-rgx, argument-rgx
- Docstrings: missing-module-docstring (C0114), missing-class-docstring (C0115), missing-function-docstring (C0116)
- Length: line-too-long (C0301), too-many-lines (C0302)
  Options: max-line-length, max-module-lines
- Imports: multiple-imports (C0410), wrong-import-order (C0411), unused-import (W0611)
- Error handling: bare-except (W0702), broad-except (W0703)
  Options: overgeneral-exceptions
- Complexity: too-many-branches (R0912), too-many-arguments (R0913), too-many-locals (R0914), too-many-statements (R0915), too-many-nested-blocks (R1702)
  Options: max-branches, max-args, max-locals, max-statements, max-nested-blocks
- Security: dangerous-default-value (W0102), exec-used (W0122), eval-used (W0123)
- Unused: unused-variable (W0612), unused-argument (W0613)

If the rule cannot be expressed in Pylint, return:
{
  "symbol": "",
  "message_id": "",
  "options": null
}

Examples:

Input: "All functions must have docstrings"
Output:
{
  "symbol": "missing-function-docstring",
  "message_id": "C0116",
  "options": null
}

Input: "Lines must not exceed 120 characters"
Output:
{
  "symbol": "line-too-long",
  "message_id": "C0301",
  "options": {"max-line-length": 120}
}

Input: "Functions should have at most 5 arguments"
Output:
{
  "symbol": "too-many-arguments",
  "message_id": "R0913",
  "options": {"max-args": 5}
}

Input: "Don't use bare except blocks"
Output:
{
  "symbol": "bare-except",
  "message_id": "W0702",
  "options": null
}`

	userPrompt := fmt.Sprintf("Convert this rule to Pylint configuration:\n\n%s", rule.Say)
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
		Symbol    string                 `json:"symbol"`
		MessageID string                 `json:"message_id"`
		Options   map[string]interface{} `json:"options"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	// If symbol is empty, this rule cannot be converted
	if result.Symbol == "" {
		return "", nil, nil
	}

	return result.Symbol, result.Options, nil
}

// generatePylintRC generates .pylintrc content in INI format
func (c *Converter) generatePylintRC(enabledRules []string, options map[string]map[string]interface{}) string {
	var sb strings.Builder

	// Header
	sb.WriteString("[MASTER]\n")
	sb.WriteString("# Generated by Symphony CLI\n\n")

	// Messages control section
	sb.WriteString("[MESSAGES CONTROL]\n")
	sb.WriteString("enable=" + strings.Join(enabledRules, ",") + "\n\n")

	// FORMAT section
	if formatOpts, ok := options["FORMAT"]; ok && len(formatOpts) > 0 {
		sb.WriteString("[FORMAT]\n")
		for key, value := range formatOpts {
			sb.WriteString(fmt.Sprintf("%s=%v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// BASIC section (naming)
	if basicOpts, ok := options["BASIC"]; ok && len(basicOpts) > 0 {
		sb.WriteString("[BASIC]\n")
		for key, value := range basicOpts {
			sb.WriteString(fmt.Sprintf("%s=%v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// DESIGN section (complexity)
	if designOpts, ok := options["DESIGN"]; ok && len(designOpts) > 0 {
		sb.WriteString("[DESIGN]\n")
		for key, value := range designOpts {
			sb.WriteString(fmt.Sprintf("%s=%v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// EXCEPTIONS section
	if exceptOpts, ok := options["EXCEPTIONS"]; ok && len(exceptOpts) > 0 {
		sb.WriteString("[EXCEPTIONS]\n")
		for key, value := range exceptOpts {
			sb.WriteString(fmt.Sprintf("%s=%v\n", key, value))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// getOptionSection returns the Pylint config section for a given option
func getOptionSection(option string) string {
	formatOptions := map[string]bool{
		"max-line-length":    true,
		"max-module-lines":   true,
		"indent-string":      true,
		"indent-after-paren": true,
	}

	basicOptions := map[string]bool{
		"variable-rgx":        true,
		"function-rgx":        true,
		"class-rgx":           true,
		"const-rgx":           true,
		"argument-rgx":        true,
		"attr-rgx":            true,
		"method-rgx":          true,
		"module-rgx":          true,
		"good-names":          true,
		"bad-names":           true,
		"include-naming-hint": true,
	}

	designOptions := map[string]bool{
		"max-args":           true,
		"max-locals":         true,
		"max-returns":        true,
		"max-branches":       true,
		"max-statements":     true,
		"max-parents":        true,
		"max-attributes":     true,
		"min-public-methods": true,
		"max-public-methods": true,
		"max-bool-expr":      true,
		"max-nested-blocks":  true,
	}

	exceptOptions := map[string]bool{
		"overgeneral-exceptions": true,
	}

	if formatOptions[option] {
		return "FORMAT"
	}
	if basicOptions[option] {
		return "BASIC"
	}
	if designOptions[option] {
		return "DESIGN"
	}
	if exceptOptions[option] {
		return "EXCEPTIONS"
	}

	return "MASTER"
}
