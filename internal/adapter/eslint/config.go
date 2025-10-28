package eslint

import (
	"encoding/json"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// ESLintConfig represents .eslintrc.json structure.
type ESLintConfig struct {
	Env   map[string]bool        `json:"env,omitempty"`
	Rules map[string]interface{} `json:"rules"`
	Extra map[string]interface{} `json:"-"` // For extensions
}

// generateConfig creates ESLint config from a Symphony rule.
func generateConfig(ruleInterface interface{}) ([]byte, error) {
	rule, ok := ruleInterface.(*core.Rule)
	if !ok {
		return nil, fmt.Errorf("expected *core.Rule, got %T", ruleInterface)
	}

	config := &ESLintConfig{
		Env: map[string]bool{
			"es2021":  true,
			"node":    true,
			"browser": true,
		},
		Rules: make(map[string]interface{}),
	}

	// Determine which ESLint rules to use based on engine type
	engine := rule.GetString("engine")

	switch engine {
	case "pattern":
		if err := addPatternRules(config, rule); err != nil {
			return nil, err
		}
	case "length":
		if err := addLengthRules(config, rule); err != nil {
			return nil, err
		}
	case "style":
		if err := addStyleRules(config, rule); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}

	return json.MarshalIndent(config, "", "  ")
}

// addPatternRules adds pattern validation rules.
func addPatternRules(config *ESLintConfig, rule *core.Rule) error {
	target := rule.GetString("target")
	pattern := rule.GetString("pattern")

	if pattern == "" {
		return fmt.Errorf("pattern is required for pattern engine")
	}

	switch target {
	case "identifier":
		// Use id-match rule for identifier patterns
		config.Rules["id-match"] = []interface{}{
			rule.Severity, // "error", "warn", "off"
			pattern,
			map[string]interface{}{
				"properties":       false,
				"classFields":      false,
				"onlyDeclarations": true,
			},
		}

	case "content":
		// Use no-restricted-syntax for content patterns
		config.Rules["no-restricted-syntax"] = []interface{}{
			rule.Severity,
			map[string]interface{}{
				"selector": fmt.Sprintf("Literal[value=/%s/]", pattern),
				"message":  rule.Message,
			},
		}

	case "import":
		// Use no-restricted-imports for import patterns
		config.Rules["no-restricted-imports"] = []interface{}{
			rule.Severity,
			map[string]interface{}{
				"patterns": []string{pattern},
			},
		}

	default:
		return fmt.Errorf("unsupported pattern target: %s", target)
	}

	return nil
}

// addLengthRules adds length constraint rules.
func addLengthRules(config *ESLintConfig, rule *core.Rule) error {
	scope := rule.GetString("scope")
	max := rule.GetInt("max")
	min := rule.GetInt("min")

	if max == 0 && min == 0 {
		return fmt.Errorf("max or min is required for length engine")
	}

	switch scope {
	case "line":
		// Use max-len rule
		opts := map[string]interface{}{
			"code": max,
		}
		if min > 0 {
			// TODO: ESLint doesn't have min-len, so we'd need custom rule
			// For now, just enforce max
			_ = min // Explicitly ignore min for now
		}
		config.Rules["max-len"] = []interface{}{rule.Severity, opts}

	case "file":
		// Use max-lines rule
		opts := map[string]interface{}{
			"max":            max,
			"skipBlankLines": true,
			"skipComments":   true,
		}
		config.Rules["max-lines"] = []interface{}{rule.Severity, opts}

	case "function":
		// Use max-lines-per-function rule
		opts := map[string]interface{}{
			"max":            max,
			"skipBlankLines": true,
			"skipComments":   true,
		}
		config.Rules["max-lines-per-function"] = []interface{}{rule.Severity, opts}

	case "params":
		// Use max-params rule
		config.Rules["max-params"] = []interface{}{rule.Severity, max}

	default:
		return fmt.Errorf("unsupported length scope: %s", scope)
	}

	return nil
}

// addStyleRules adds style formatting rules.
func addStyleRules(config *ESLintConfig, rule *core.Rule) error {
	// Get style properties from rule.Check
	indent := rule.GetInt("indent")
	quote := rule.GetString("quote")
	semi := rule.GetBool("semi")

	if indent > 0 {
		config.Rules["indent"] = []interface{}{rule.Severity, indent}
	}

	if quote != "" {
		config.Rules["quotes"] = []interface{}{rule.Severity, quote}
	}

	// Semi is boolean, but we need to handle it carefully
	// If explicitly set, add the rule
	if _, ok := rule.Check["semi"]; ok {
		if semi {
			config.Rules["semi"] = []interface{}{rule.Severity, "always"}
		} else {
			config.Rules["semi"] = []interface{}{rule.Severity, "never"}
		}
	}

	return nil
}

// MarshalConfig converts a config map to JSON bytes.
func MarshalConfig(config map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// MapSeverity converts severity string to ESLint severity level.
func MapSeverity(severity string) interface{} {
	switch severity {
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
