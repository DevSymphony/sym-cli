package linters

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// PMDConverter converts rules to PMD ruleset XML configuration
type PMDConverter struct {
	verbose bool
}

// NewPMDConverter creates a new PMD converter
func NewPMDConverter(verbose bool) *PMDConverter {
	return &PMDConverter{
		verbose: verbose,
	}
}

// Name returns the linter name
func (c *PMDConverter) Name() string {
	return "pmd"
}

// SupportedLanguages returns supported languages
func (c *PMDConverter) SupportedLanguages() []string {
	return []string{"java"}
}

// SupportedCategories returns supported rule categories
func (c *PMDConverter) SupportedCategories() []string {
	return []string{
		"naming",
		"complexity",
		"design",
		"performance",
		"security",
		"error_handling",
		"best_practices",
		"code_style",
	}
}

// PMDRuleset represents the root PMD ruleset
type PMDRuleset struct {
	XMLName     xml.Name `xml:"ruleset"`
	Name        string   `xml:"name,attr"`
	XMLNS       string   `xml:"xmlns,attr"`
	XMLNSXSI    string   `xml:"xmlns:xsi,attr"`
	XSISchema   string   `xml:"xsi:schemaLocation,attr"`
	Description string   `xml:"description"`
	Rules       []PMDRule `xml:"rule"`
}

// PMDRule represents a single PMD rule reference
type PMDRule struct {
	XMLName    xml.Name        `xml:"rule"`
	Ref        string          `xml:"ref,attr,omitempty"`
	Name       string          `xml:"name,attr,omitempty"`
	Message    string          `xml:"message,attr,omitempty"`
	Class      string          `xml:"class,attr,omitempty"`
	Priority   int             `xml:"priority,omitempty"`
	Properties []PMDProperty   `xml:"properties>property,omitempty"`
	Comment    string          `xml:",comment"`
}

// PMDProperty represents a property in PMD rule
type PMDProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
}

// Convert converts a user rule with intent to PMD rule
func (c *PMDConverter) Convert(userRule *schema.UserRule, intent *llm.RuleIntent) (*LinterRule, error) {
	if userRule == nil {
		return nil, fmt.Errorf("user rule is nil")
	}

	if intent == nil {
		return nil, fmt.Errorf("rule intent is nil")
	}

	priority := c.mapSeverityToPriority(userRule.Severity)

	var rules []PMDRule
	var err error

	switch intent.Engine {
	case "pattern":
		rules, err = c.convertPatternRule(intent, priority)
	case "length":
		rules, err = c.convertLengthRule(intent, priority)
	case "style":
		rules, err = c.convertStyleRule(intent, priority)
	case "ast":
		rules, err = c.convertASTRule(intent, priority)
	default:
		// Return empty config with comment for unsupported rules
		return &LinterRule{
			ID:       userRule.ID,
			Severity: userRule.Severity,
			Config:   make(map[string]any),
			Comment:  fmt.Sprintf("Unsupported rule (engine: %s): %s", intent.Engine, userRule.Say),
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to convert rule: %w", err)
	}

	// Store rules in config map
	config := map[string]any{
		"rules": rules,
	}

	return &LinterRule{
		ID:       userRule.ID,
		Severity: userRule.Severity,
		Config:   config,
		Comment:  userRule.Say,
	}, nil
}

// GenerateConfig generates PMD ruleset XML configuration from rules
func (c *PMDConverter) GenerateConfig(rules []*LinterRule) (*LinterConfig, error) {
	ruleset := PMDRuleset{
		Name:        "Symphony Convention Rules",
		XMLNS:       "http://pmd.sourceforge.net/ruleset/2.0.0",
		XMLNSXSI:    "http://www.w3.org/2001/XMLSchema-instance",
		XSISchema:   "http://pmd.sourceforge.net/ruleset/2.0.0 https://pmd.sourceforge.io/ruleset_2_0_0.xsd",
		Description: "Generated PMD ruleset from Symphony user policy",
		Rules:       []PMDRule{},
	}

	// Collect all PMD rules
	for _, rule := range rules {
		if rulesInterface, ok := rule.Config["rules"]; ok {
			if pmdRules, ok := rulesInterface.([]PMDRule); ok {
				for _, pmdRule := range pmdRules {
					// Add comment if available
					if rule.Comment != "" {
						pmdRule.Comment = " " + rule.Comment + " "
					}
					ruleset.Rules = append(ruleset.Rules, pmdRule)
				}
			}
		}
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(ruleset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PMD ruleset: %w", err)
	}

	// Add XML header
	xmlHeader := `<?xml version="1.0"?>
`
	content := []byte(xmlHeader + string(output))

	return &LinterConfig{
		Format:   "xml",
		Filename: "pmd-ruleset.xml",
		Content:  content,
	}, nil
}

// convertPatternRule converts pattern engine rules to PMD rules
func (c *PMDConverter) convertPatternRule(intent *llm.RuleIntent, priority int) ([]PMDRule, error) {
	rules := []PMDRule{}

	switch intent.Target {
	case "class":
		// Class naming convention
		if caseStyle, ok := intent.Params["case"].(string); ok && strings.ToLower(caseStyle) == "pascalcase" {
			rules = append(rules, PMDRule{
				Ref:      "category/java/codestyle.xml/ClassNamingConventions",
				Priority: priority,
			})
		}

	case "method", "function":
		// Method naming convention
		if caseStyle, ok := intent.Params["case"].(string); ok && strings.ToLower(caseStyle) == "camelcase" {
			rules = append(rules, PMDRule{
				Ref:      "category/java/codestyle.xml/MethodNamingConventions",
				Priority: priority,
			})
		}

	case "variable":
		// Variable naming convention
		rules = append(rules, PMDRule{
			Ref:      "category/java/codestyle.xml/LocalVariableNamingConventions",
			Priority: priority,
		})

	case "import", "dependency":
		// Import restrictions
		rules = append(rules, PMDRule{
			Ref:      "category/java/codestyle.xml/UnnecessaryImport",
			Priority: priority,
		})
		rules = append(rules, PMDRule{
			Ref:      "category/java/codestyle.xml/DuplicateImports",
			Priority: priority,
		})
	}

	return rules, nil
}

// convertLengthRule converts length engine rules to PMD rules
func (c *PMDConverter) convertLengthRule(intent *llm.RuleIntent, priority int) ([]PMDRule, error) {
	rules := []PMDRule{}

	max := c.getIntParam(intent.Params, "max")

	switch intent.Scope {
	case "method", "function":
		if max > 0 {
			rules = append(rules, PMDRule{
				Ref:      "category/java/design.xml/ExcessiveMethodLength",
				Priority: priority,
				Properties: []PMDProperty{
					{Name: "minimum", Value: fmt.Sprintf("%d", max), Type: "Integer"},
				},
			})
		}

	case "class":
		if max > 0 {
			rules = append(rules, PMDRule{
				Ref:      "category/java/design.xml/ExcessiveClassLength",
				Priority: priority,
				Properties: []PMDProperty{
					{Name: "minimum", Value: fmt.Sprintf("%d", max), Type: "Integer"},
				},
			})
		}

	case "params", "parameters":
		if max > 0 {
			rules = append(rules, PMDRule{
				Ref:      "category/java/design.xml/ExcessiveParameterList",
				Priority: priority,
				Properties: []PMDProperty{
					{Name: "minimum", Value: fmt.Sprintf("%d", max), Type: "Integer"},
				},
			})
		}
	}

	return rules, nil
}

// convertStyleRule converts style engine rules to PMD rules
func (c *PMDConverter) convertStyleRule(intent *llm.RuleIntent, priority int) ([]PMDRule, error) {
	rules := []PMDRule{}

	// PMD has limited style rules compared to Checkstyle
	// Add some common code style rules
	rules = append(rules, PMDRule{
		Ref:      "category/java/codestyle.xml/UnnecessaryModifier",
		Priority: priority,
	})

	rules = append(rules, PMDRule{
		Ref:      "category/java/codestyle.xml/UselessParentheses",
		Priority: priority,
	})

	return rules, nil
}

// convertASTRule converts AST engine rules to PMD rules
func (c *PMDConverter) convertASTRule(intent *llm.RuleIntent, priority int) ([]PMDRule, error) {
	rules := []PMDRule{}

	// Cyclomatic complexity
	if complexity := c.getIntParam(intent.Params, "complexity"); complexity > 0 {
		rules = append(rules, PMDRule{
			Ref:      "category/java/design.xml/CyclomaticComplexity",
			Priority: priority,
			Properties: []PMDProperty{
				{
					Name:  "methodReportLevel",
					Value: fmt.Sprintf("%d", complexity),
					Type:  "Integer",
				},
			},
		})
	}

	// Nesting depth
	if depth := c.getIntParam(intent.Params, "depth"); depth > 0 {
		rules = append(rules, PMDRule{
			Ref:      "category/java/design.xml/AvoidDeeplyNestedIfStmts",
			Priority: priority,
			Properties: []PMDProperty{
				{
					Name:  "problemDepth",
					Value: fmt.Sprintf("%d", depth),
					Type:  "Integer",
				},
			},
		})
	}

	// Cognitive complexity
	if cognitiveComplexity := c.getIntParam(intent.Params, "cognitiveComplexity"); cognitiveComplexity > 0 {
		rules = append(rules, PMDRule{
			Ref:      "category/java/design.xml/CognitiveComplexity",
			Priority: priority,
			Properties: []PMDProperty{
				{
					Name:  "reportLevel",
					Value: fmt.Sprintf("%d", cognitiveComplexity),
					Type:  "Integer",
				},
			},
		})
	}

	return rules, nil
}

// mapSeverityToPriority maps Symphony severity to PMD priority (1-5, lower is more severe)
func (c *PMDConverter) mapSeverityToPriority(severity string) int {
	switch strings.ToLower(severity) {
	case "error":
		return 1
	case "warning", "warn":
		return 3
	case "info":
		return 5
	default:
		return 1
	}
}

// getIntParam safely extracts an integer parameter
func (c *PMDConverter) getIntParam(params map[string]any, key string) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			var i int
			fmt.Sscanf(v, "%d", &i)
			return i
		}
	}
	return 0
}

func init() {
	// Register PMD converter on package initialization
	Register(NewPMDConverter(false))
}
