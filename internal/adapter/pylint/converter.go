package pylint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

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

// ConvertRules converts user rules to Pylint configuration using LLM.
// Returns ConversionResult with per-rule success/failure tracking for fallback support.
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, provider llm.Provider) (*adapter.ConversionResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	// Convert rules in parallel using goroutines
	type ruleResult struct {
		index   int
		ruleID  string
		symbol  string
		options map[string]interface{}
		err     error
	}

	results := make(chan ruleResult, len(rules))
	var wg sync.WaitGroup

	// Process each rule in parallel
	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, r schema.UserRule) {
			defer wg.Done()

			symbol, options, err := c.convertSingleRule(ctx, r, provider)
			results <- ruleResult{
				index:   idx,
				ruleID:  r.ID,
				symbol:  symbol,
				options: options,
				err:     err,
			}
		}(i, rule)
	}

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with per-rule tracking
	enabledRules := make([]string, 0)
	options := make(map[string]map[string]interface{})
	successRuleIDs := make([]string, 0)
	failedRuleIDs := make([]string, 0)

	for result := range results {
		if result.err != nil {
			failedRuleIDs = append(failedRuleIDs, result.ruleID)
			fmt.Fprintf(os.Stderr, "⚠️  Pylint rule %s conversion error: %v\n", result.ruleID, result.err)
			continue
		}

		if result.symbol != "" {
			enabledRules = append(enabledRules, result.symbol)
			successRuleIDs = append(successRuleIDs, result.ruleID)
			if len(result.options) > 0 {
				// Group options by section
				for key, value := range result.options {
					section := getOptionSection(key)
					if _, ok := options[section]; !ok {
						options[section] = make(map[string]interface{})
					}
					options[section][key] = value
				}
			}
			fmt.Fprintf(os.Stderr, "✓ Pylint rule %s → %s\n", result.ruleID, result.symbol)
		} else {
			// Skipped = cannot be enforced by this linter, fallback to llm-validator
			failedRuleIDs = append(failedRuleIDs, result.ruleID)
			fmt.Fprintf(os.Stderr, "⊘ Pylint rule %s skipped (cannot be enforced by Pylint)\n", result.ruleID)
		}
	}

	// Build result with tracking info
	convResult := &adapter.ConversionResult{
		SuccessRules: successRuleIDs,
		FailedRules:  failedRuleIDs,
	}

	// Generate config only if at least one rule succeeded
	if len(enabledRules) > 0 {
		content := c.generatePylintRC(enabledRules, options)
		convResult.Config = &adapter.LinterConfig{
			Filename: ".pylintrc",
			Content:  []byte(content),
			Format:   "ini",
		}
	}

	return convResult, nil
}

// convertSingleRule converts a single user rule to Pylint rule using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (string, map[string]interface{}, error) {
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
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

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
