package linters

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// CheckstyleConverter converts rules to Checkstyle XML configuration
type CheckstyleConverter struct {
	verbose bool
}

// NewCheckstyleConverter creates a new Checkstyle converter
func NewCheckstyleConverter(verbose bool) *CheckstyleConverter {
	return &CheckstyleConverter{
		verbose: verbose,
	}
}

// Name returns the linter name
func (c *CheckstyleConverter) Name() string {
	return "checkstyle"
}

// SupportedLanguages returns supported languages
func (c *CheckstyleConverter) SupportedLanguages() []string {
	return []string{"java"}
}

// SupportedCategories returns supported rule categories
func (c *CheckstyleConverter) SupportedCategories() []string {
	return []string{
		"naming",
		"formatting",
		"style",
		"length",
		"complexity",
		"whitespace",
		"javadoc",
		"imports",
	}
}

// CheckstyleModule represents a Checkstyle module in XML
type CheckstyleModule struct {
	XMLName    xml.Name            `xml:"module"`
	Name       string              `xml:"name,attr"`
	Properties []CheckstyleProperty `xml:"property,omitempty"`
	Modules    []CheckstyleModule  `xml:"module,omitempty"`
	Comment    string              `xml:",comment"`
}

// CheckstyleProperty represents a property in Checkstyle XML
type CheckstyleProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

// CheckstyleConfig represents the root Checkstyle configuration
type CheckstyleConfig struct {
	XMLName xml.Name         `xml:"module"`
	Name    string           `xml:"name,attr"`
	Modules []CheckstyleModule `xml:"module"`
}

// Convert converts a user rule with intent to Checkstyle module
func (c *CheckstyleConverter) Convert(userRule *schema.UserRule, intent *llm.RuleIntent) (*LinterRule, error) {
	if userRule == nil {
		return nil, fmt.Errorf("user rule is nil")
	}

	if intent == nil {
		return nil, fmt.Errorf("rule intent is nil")
	}

	severity := c.mapSeverity(userRule.Severity)

	var modules []CheckstyleModule
	var err error

	switch intent.Engine {
	case "pattern":
		modules, err = c.convertPatternRule(intent, severity)
	case "length":
		modules, err = c.convertLengthRule(intent, severity)
	case "style":
		modules, err = c.convertStyleRule(intent, severity)
	case "ast":
		modules, err = c.convertASTRule(intent, severity)
	default:
		// Return empty config with comment for unsupported rules
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

	// Store modules in config map
	config := map[string]any{
		"modules": modules,
	}

	return &LinterRule{
		ID:       userRule.ID,
		Severity: severity,
		Config:   config,
		Comment:  userRule.Say,
	}, nil
}

// GenerateConfig generates Checkstyle XML configuration from rules
func (c *CheckstyleConverter) GenerateConfig(rules []*LinterRule) (*LinterConfig, error) {
	rootModule := CheckstyleConfig{
		Name:    "Checker",
		Modules: []CheckstyleModule{},
	}

	// TreeWalker module for most rules
	treeWalker := CheckstyleModule{
		Name:    "TreeWalker",
		Modules: []CheckstyleModule{},
	}

	// Collect all modules from rules
	for _, rule := range rules {
		if modulesInterface, ok := rule.Config["modules"]; ok {
			if modules, ok := modulesInterface.([]CheckstyleModule); ok {
				for _, module := range modules {
					// Add comment if available
					if rule.Comment != "" {
						module.Comment = " " + rule.Comment + " "
					}
					treeWalker.Modules = append(treeWalker.Modules, module)
				}
			}
		}
	}

	// Add TreeWalker to root if it has modules
	if len(treeWalker.Modules) > 0 {
		rootModule.Modules = append(rootModule.Modules, treeWalker)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(rootModule, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Checkstyle config: %w", err)
	}

	// Add XML header and DOCTYPE
	xmlHeader := `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
    "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
    "https://checkstyle.org/dtds/configuration_1_3.dtd">
`
	content := []byte(xmlHeader + string(output))

	return &LinterConfig{
		Format:   "xml",
		Filename: "checkstyle.xml",
		Content:  content,
	}, nil
}

// convertPatternRule converts pattern engine rules to Checkstyle modules
func (c *CheckstyleConverter) convertPatternRule(intent *llm.RuleIntent, severity string) ([]CheckstyleModule, error) {
	modules := []CheckstyleModule{}

	switch intent.Target {
	case "identifier", "variable", "class", "method", "function":
		// Naming conventions
		if caseStyle, ok := intent.Params["case"].(string); ok {
			format := c.caseToRegex(caseStyle)

			switch intent.Target {
			case "class":
				modules = append(modules, CheckstyleModule{
					Name: "TypeName",
					Properties: []CheckstyleProperty{
						{Name: "format", Value: format},
						{Name: "severity", Value: severity},
					},
				})

			case "method", "function":
				modules = append(modules, CheckstyleModule{
					Name: "MethodName",
					Properties: []CheckstyleProperty{
						{Name: "format", Value: format},
						{Name: "severity", Value: severity},
					},
				})

			case "variable":
				modules = append(modules, CheckstyleModule{
					Name: "LocalVariableName",
					Properties: []CheckstyleProperty{
						{Name: "format", Value: format},
						{Name: "severity", Value: severity},
					},
				})

			default:
				// Generic member name
				modules = append(modules, CheckstyleModule{
					Name: "MemberName",
					Properties: []CheckstyleProperty{
						{Name: "format", Value: format},
						{Name: "severity", Value: severity},
					},
				})
			}
		} else if len(intent.Patterns) > 0 {
			// Use the first pattern
			pattern := intent.Patterns[0]
			modules = append(modules, CheckstyleModule{
				Name: "MemberName",
				Properties: []CheckstyleProperty{
					{Name: "format", Value: pattern},
					{Name: "severity", Value: severity},
				},
			})
		}

	case "import", "dependency":
		// Import control
		if len(intent.Patterns) > 0 {
			for _, pattern := range intent.Patterns {
				modules = append(modules, CheckstyleModule{
					Name: "IllegalImport",
					Properties: []CheckstyleProperty{
						{Name: "illegalPkgs", Value: pattern},
						{Name: "severity", Value: severity},
					},
				})
			}
		}
	}

	return modules, nil
}

// convertLengthRule converts length engine rules to Checkstyle modules
func (c *CheckstyleConverter) convertLengthRule(intent *llm.RuleIntent, severity string) ([]CheckstyleModule, error) {
	modules := []CheckstyleModule{}

	max := c.getIntParam(intent.Params, "max")

	switch intent.Scope {
	case "line":
		if max > 0 {
			modules = append(modules, CheckstyleModule{
				Name: "LineLength",
				Properties: []CheckstyleProperty{
					{Name: "max", Value: fmt.Sprintf("%d", max)},
					{Name: "severity", Value: severity},
				},
			})
		}

	case "file":
		if max > 0 {
			modules = append(modules, CheckstyleModule{
				Name: "FileLength",
				Properties: []CheckstyleProperty{
					{Name: "max", Value: fmt.Sprintf("%d", max)},
					{Name: "severity", Value: severity},
				},
			})
		}

	case "method", "function":
		if max > 0 {
			modules = append(modules, CheckstyleModule{
				Name: "MethodLength",
				Properties: []CheckstyleProperty{
					{Name: "max", Value: fmt.Sprintf("%d", max)},
					{Name: "severity", Value: severity},
				},
			})
		}

	case "params", "parameters":
		if max > 0 {
			modules = append(modules, CheckstyleModule{
				Name: "ParameterNumber",
				Properties: []CheckstyleProperty{
					{Name: "max", Value: fmt.Sprintf("%d", max)},
					{Name: "severity", Value: severity},
				},
			})
		}
	}

	return modules, nil
}

// convertStyleRule converts style engine rules to Checkstyle modules
func (c *CheckstyleConverter) convertStyleRule(intent *llm.RuleIntent, severity string) ([]CheckstyleModule, error) {
	modules := []CheckstyleModule{}

	// Indentation
	if indent := c.getIntParam(intent.Params, "indent"); indent > 0 {
		modules = append(modules, CheckstyleModule{
			Name: "Indentation",
			Properties: []CheckstyleProperty{
				{Name: "basicOffset", Value: fmt.Sprintf("%d", indent)},
				{Name: "braceAdjustment", Value: "0"},
				{Name: "caseIndent", Value: fmt.Sprintf("%d", indent)},
				{Name: "severity", Value: severity},
			},
		})
	}

	// Whitespace around operators
	modules = append(modules, CheckstyleModule{
		Name: "WhitespaceAround",
		Properties: []CheckstyleProperty{
			{Name: "severity", Value: severity},
		},
	})

	return modules, nil
}

// convertASTRule converts AST engine rules to Checkstyle modules
func (c *CheckstyleConverter) convertASTRule(intent *llm.RuleIntent, severity string) ([]CheckstyleModule, error) {
	modules := []CheckstyleModule{}

	// Cyclomatic complexity
	if complexity := c.getIntParam(intent.Params, "complexity"); complexity > 0 {
		modules = append(modules, CheckstyleModule{
			Name: "CyclomaticComplexity",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", complexity)},
				{Name: "severity", Value: severity},
			},
		})
	}

	// Nesting depth
	if depth := c.getIntParam(intent.Params, "depth"); depth > 0 {
		modules = append(modules, CheckstyleModule{
			Name: "NestedIfDepth",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", depth)},
				{Name: "severity", Value: severity},
			},
		})
	}

	return modules, nil
}

// mapSeverity maps Symphony severity to Checkstyle severity
func (c *CheckstyleConverter) mapSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "error"
	case "warning", "warn":
		return "warning"
	case "info":
		return "info"
	default:
		return "error"
	}
}

// caseToRegex converts case style name to regex pattern (Java conventions)
func (c *CheckstyleConverter) caseToRegex(caseStyle string) string {
	switch strings.ToLower(caseStyle) {
	case "pascalcase":
		return "^[A-Z][a-zA-Z0-9]*$"
	case "camelcase":
		return "^[a-z][a-zA-Z0-9]*$"
	case "snake_case":
		return "^[a-z][a-z0-9_]*$"
	case "screaming_snake_case":
		return "^[A-Z][A-Z0-9_]*$"
	default:
		return "^[a-zA-Z][a-zA-Z0-9]*$"
	}
}

// getIntParam safely extracts an integer parameter
func (c *CheckstyleConverter) getIntParam(params map[string]any, key string) int {
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
	// Register Checkstyle converter on package initialization
	Register(NewCheckstyleConverter(false))
}
