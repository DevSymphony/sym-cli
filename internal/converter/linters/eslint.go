package linters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// ESLintConverter converts rules to ESLint configuration
type ESLintConverter struct {
	verbose bool
}

// NewESLintConverter creates a new ESLint converter
func NewESLintConverter(verbose bool) *ESLintConverter {
	return &ESLintConverter{
		verbose: verbose,
	}
}

// Name returns the linter name
func (c *ESLintConverter) Name() string {
	return "eslint"
}

// SupportedLanguages returns supported languages
func (c *ESLintConverter) SupportedLanguages() []string {
	return []string{"javascript", "typescript", "js", "ts", "jsx", "tsx"}
}

// SupportedCategories returns supported rule categories
func (c *ESLintConverter) SupportedCategories() []string {
	return []string{
		"naming",
		"formatting",
		"style",
		"length",
		"security",
		"error_handling",
		"dependency",
		"import",
	}
}

// Convert converts a user rule with intent to ESLint rule
func (c *ESLintConverter) Convert(userRule *schema.UserRule, intent *llm.RuleIntent) (*LinterRule, error) {
	if userRule == nil {
		return nil, fmt.Errorf("user rule is nil")
	}

	if intent == nil {
		return nil, fmt.Errorf("rule intent is nil")
	}

	severity := c.mapSeverity(userRule.Severity)

	// Map based on engine type
	var config map[string]any
	var err error

	switch intent.Engine {
	case "pattern":
		config, err = c.convertPatternRule(intent, severity)
	case "length":
		config, err = c.convertLengthRule(intent, severity)
	case "style":
		config, err = c.convertStyleRule(intent, severity)
	case "ast":
		config, err = c.convertASTRule(intent, severity)
	default:
		// Custom or unsupported engine - create generic comment
		return &LinterRule{
			ID:       userRule.ID,
			Severity: severity,
			Config:   make(map[string]any),
			Comment:  fmt.Sprintf("Unsupported rule (engine: %s): %s", intent.Engine, userRule.Say),
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to convert rule: %w", err)
	}

	return &LinterRule{
		ID:       userRule.ID,
		Severity: severity,
		Config:   config,
		Comment:  userRule.Say,
	}, nil
}

// GenerateConfig generates ESLint configuration from rules
func (c *ESLintConverter) GenerateConfig(rules []*LinterRule) (*LinterConfig, error) {
	eslintConfig := map[string]any{
		"env": map[string]bool{
			"es2021":  true,
			"node":    true,
			"browser": true,
		},
		"rules": make(map[string]any),
	}

	rulesMap := eslintConfig["rules"].(map[string]any)

	// Merge all rule configs
	for _, rule := range rules {
		for ruleID, ruleConfig := range rule.Config {
			rulesMap[ruleID] = ruleConfig
		}
	}

	// Add comments as a separate field (not part of standard ESLint config)
	comments := make(map[string]string)
	for _, rule := range rules {
		if rule.Comment != "" {
			for ruleID := range rule.Config {
				comments[ruleID] = rule.Comment
			}
		}
	}

	if len(comments) > 0 {
		eslintConfig["_comments"] = comments
	}

	content, err := json.MarshalIndent(eslintConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ESLint config: %w", err)
	}

	return &LinterConfig{
		Format:   "json",
		Filename: ".eslintrc.json",
		Content:  content,
	}, nil
}

// convertPatternRule converts pattern engine rules to ESLint rules
func (c *ESLintConverter) convertPatternRule(intent *llm.RuleIntent, severity string) (map[string]any, error) {
	config := make(map[string]any)

	switch intent.Target {
	case "identifier", "variable", "function", "class":
		// Use id-match for identifier patterns
		if len(intent.Patterns) > 0 {
			pattern := intent.Patterns[0]
			config["id-match"] = []any{
				severity,
				pattern,
				map[string]any{
					"properties":       false,
					"classFields":      false,
					"onlyDeclarations": true,
				},
			}
		} else if caseStyle, ok := intent.Params["case"].(string); ok {
			// Convert case style to regex
			pattern := c.caseToRegex(caseStyle)
			config["id-match"] = []any{
				severity,
				pattern,
				map[string]any{
					"properties":       false,
					"classFields":      false,
					"onlyDeclarations": true,
				},
			}
		}

	case "content":
		// Use no-restricted-syntax for content patterns
		if len(intent.Patterns) > 0 {
			pattern := intent.Patterns[0]
			config["no-restricted-syntax"] = []any{
				severity,
				map[string]any{
					"selector": fmt.Sprintf("Literal[value=/%s/]", pattern),
					"message":  "Forbidden pattern detected",
				},
			}
		}

	case "import", "dependency":
		// Use no-restricted-imports
		if len(intent.Patterns) > 0 {
			config["no-restricted-imports"] = []any{
				severity,
				map[string]any{
					"patterns": intent.Patterns,
				},
			}
		}

	default:
		return nil, fmt.Errorf("unsupported pattern target: %s", intent.Target)
	}

	return config, nil
}

// convertLengthRule converts length engine rules to ESLint rules
func (c *ESLintConverter) convertLengthRule(intent *llm.RuleIntent, severity string) (map[string]any, error) {
	config := make(map[string]any)

	max := c.getIntParam(intent.Params, "max")
	min := c.getIntParam(intent.Params, "min")

	switch intent.Scope {
	case "line":
		if max > 0 {
			config["max-len"] = []any{
				severity,
				map[string]any{
					"code": max,
				},
			}
		}

	case "file":
		if max > 0 {
			config["max-lines"] = []any{
				severity,
				map[string]any{
					"max":            max,
					"skipBlankLines": true,
					"skipComments":   true,
				},
			}
		}

	case "function", "method":
		if max > 0 {
			config["max-lines-per-function"] = []any{
				severity,
				map[string]any{
					"max":            max,
					"skipBlankLines": true,
					"skipComments":   true,
				},
			}
		}

	case "params", "parameters":
		if max > 0 {
			config["max-params"] = []any{severity, max}
		}

	default:
		return nil, fmt.Errorf("unsupported length scope: %s", intent.Scope)
	}

	// Note: ESLint doesn't have min-len, so we ignore min for now
	_ = min

	return config, nil
}

// convertStyleRule converts style engine rules to ESLint rules
func (c *ESLintConverter) convertStyleRule(intent *llm.RuleIntent, severity string) (map[string]any, error) {
	config := make(map[string]any)

	// Indent
	if indent := c.getIntParam(intent.Params, "indent"); indent > 0 {
		config["indent"] = []any{severity, indent}
	}

	// Quote style
	if quote, ok := intent.Params["quote"].(string); ok {
		config["quotes"] = []any{severity, quote}
	}

	// Semicolons
	if semi, ok := intent.Params["semi"].(bool); ok {
		if semi {
			config["semi"] = []any{severity, "always"}
		} else {
			config["semi"] = []any{severity, "never"}
		}
	}

	// Trailing comma
	if trailingComma, ok := intent.Params["trailingComma"].(string); ok {
		config["comma-dangle"] = []any{severity, trailingComma}
	}

	return config, nil
}

// convertASTRule converts AST engine rules to ESLint rules
func (c *ESLintConverter) convertASTRule(intent *llm.RuleIntent, severity string) (map[string]any, error) {
	config := make(map[string]any)

	// Cyclomatic complexity
	if complexity := c.getIntParam(intent.Params, "complexity"); complexity > 0 {
		config["complexity"] = []any{severity, complexity}
	}

	// Max depth
	if depth := c.getIntParam(intent.Params, "depth"); depth > 0 {
		config["max-depth"] = []any{severity, depth}
	}

	// Max nested callbacks
	if callbacks := c.getIntParam(intent.Params, "callbacks"); callbacks > 0 {
		config["max-nested-callbacks"] = []any{severity, callbacks}
	}

	return config, nil
}

// mapSeverity maps Symphony severity to ESLint severity
func (c *ESLintConverter) mapSeverity(severity string) string {
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

// caseToRegex converts case style name to regex pattern
func (c *ESLintConverter) caseToRegex(caseStyle string) string {
	switch strings.ToLower(caseStyle) {
	case "pascalcase":
		return "^[A-Z][a-zA-Z0-9]*$"
	case "camelcase":
		return "^[a-z][a-zA-Z0-9]*$"
	case "snake_case":
		return "^[a-z][a-z0-9_]*$"
	case "screaming_snake_case":
		return "^[A-Z][A-Z0-9_]*$"
	case "kebab-case":
		return "^[a-z][a-z0-9-]*$"
	default:
		return "^[a-zA-Z][a-zA-Z0-9]*$"
	}
}

// getIntParam safely extracts an integer parameter
func (c *ESLintConverter) getIntParam(params map[string]any, key string) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			var i int
			_, _ = fmt.Sscanf(v, "%d", &i)
			return i
		}
	}
	return 0
}

func init() {
	// Register ESLint converter on package initialization
	Register(NewESLintConverter(false))
}
