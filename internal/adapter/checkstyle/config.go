package checkstyle

import (
	"encoding/xml"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// CheckstyleModule represents a Checkstyle module in XML.
type CheckstyleModule struct {
	XMLName    xml.Name             `xml:"module"`
	Name       string               `xml:"name,attr"`
	Properties []CheckstyleProperty `xml:"property,omitempty"`
	Modules    []CheckstyleModule   `xml:"module,omitempty"`
}

// CheckstyleProperty represents a property in Checkstyle XML.
type CheckstyleProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value    string   `xml:"value,attr"`
}

// CheckstyleConfig represents the root Checkstyle configuration.
type CheckstyleConfig struct {
	XMLName xml.Name           `xml:"module"`
	Name    string             `xml:"name,attr"`
	Modules []CheckstyleModule `xml:"module"`
}

// generateConfig generates Checkstyle XML configuration from a rule.
// The rule parameter should be a *core.Rule containing check configuration.
func generateConfig(ruleInterface interface{}) ([]byte, error) {
	// Type assert to *core.Rule
	rule, ok := ruleInterface.(*core.Rule)
	if !ok {
		return nil, fmt.Errorf("expected *core.Rule, got %T", ruleInterface)
	}

	// Get engine type to determine how to generate config
	engine := rule.GetString("engine")

	// Build check modules based on engine type
	var checkModules []CheckstyleModule

	switch engine {
	case "pattern":
		// For pattern engine rules, create a module based on target
		target := rule.GetString("target")
		pattern := rule.GetString("pattern")
		if target != "" && pattern != "" {
			module := CheckstyleModule{
				Name: target, // e.g., "TypeName", "MethodName", "MemberName"
				Properties: []CheckstyleProperty{
					{
						Name:  "format",
						Value: pattern,
					},
				},
			}
			checkModules = append(checkModules, module)
		}

	case "style":
		// For style engine rules, map style properties to Checkstyle modules
		checkModules = generateStyleModules(rule)

	case "length":
		// For length engine rules, create length check modules
		checkModules = generateLengthModules(rule)
	}

	// Separate TreeWalker modules from Checker-level modules
	var treeWalkerModules []CheckstyleModule
	var checkerModules []CheckstyleModule

	for _, module := range checkModules {
		// LineLength must be a direct child of Checker, not TreeWalker
		if module.Name == "LineLength" {
			checkerModules = append(checkerModules, module)
		} else {
			treeWalkerModules = append(treeWalkerModules, module)
		}
	}

	// Build the root configuration
	modules := []CheckstyleModule{}

	// Add TreeWalker with its children if any
	if len(treeWalkerModules) > 0 {
		modules = append(modules, CheckstyleModule{
			Name:    "TreeWalker",
			Modules: treeWalkerModules,
		})
	}

	// Add Checker-level modules
	modules = append(modules, checkerModules...)

	rootModule := CheckstyleConfig{
		Name:    "Checker",
		Modules: modules,
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(rootModule, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal checkstyle config: %w", err)
	}

	// Add XML header and DOCTYPE
	xmlHeader := `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
    "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
    "https://checkstyle.org/dtds/configuration_1_3.dtd">
`

	return []byte(xmlHeader + string(output)), nil
}

// generateStyleModules creates Checkstyle modules for style rules.
func generateStyleModules(rule *core.Rule) []CheckstyleModule {
	var modules []CheckstyleModule

	// Indentation
	if indent := rule.GetInt("indent"); indent > 0 {
		modules = append(modules, CheckstyleModule{
			Name: "Indentation",
			Properties: []CheckstyleProperty{
				{Name: "basicOffset", Value: fmt.Sprintf("%d", indent)},
				{Name: "braceAdjustment", Value: "0"},
				{Name: "caseIndent", Value: fmt.Sprintf("%d", indent)},
			},
		})
	}

	// Brace style
	if braceStyle := rule.GetString("braceStyle"); braceStyle == "same-line" {
		modules = append(modules, CheckstyleModule{
			Name: "LeftCurly",
			Properties: []CheckstyleProperty{
				{Name: "option", Value: "eol"},
			},
		})
	}

	// Space after keyword
	if rule.GetBool("spaceAfterKeyword") {
		modules = append(modules, CheckstyleModule{
			Name: "WhitespaceAfter",
			Properties: []CheckstyleProperty{
				{Name: "tokens", Value: "COMMA, SEMI, LITERAL_IF, LITERAL_ELSE, LITERAL_WHILE, LITERAL_DO, LITERAL_FOR"},
			},
		})
	}

	// Space around operators
	if rule.GetBool("spaceAroundOperators") {
		modules = append(modules, CheckstyleModule{
			Name: "WhitespaceAround",
			Properties: []CheckstyleProperty{
				{Name: "allowEmptyConstructors", Value: "true"},
				{Name: "allowEmptyMethods", Value: "true"},
			},
		})
	}

	// Line length
	if printWidth := rule.GetInt("printWidth"); printWidth > 0 {
		modules = append(modules, CheckstyleModule{
			Name: "LineLength",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", printWidth)},
			},
		})
	}

	// Blank lines between methods
	if rule.GetBool("blankLinesBetweenMethods") {
		modules = append(modules, CheckstyleModule{
			Name: "EmptyLineSeparator",
			Properties: []CheckstyleProperty{
				{Name: "allowNoEmptyLineBetweenFields", Value: "true"},
				{Name: "tokens", Value: "METHOD_DEF"},
			},
		})
	}

	// One statement per line
	if rule.GetBool("oneStatementPerLine") {
		modules = append(modules, CheckstyleModule{
			Name: "OneStatementPerLine",
		})
	}

	return modules
}

// generateLengthModules creates Checkstyle modules for length rules.
func generateLengthModules(rule *core.Rule) []CheckstyleModule {
	var modules []CheckstyleModule

	scope := rule.GetString("scope")
	max := rule.GetInt("max")

	if max == 0 {
		return modules
	}

	switch scope {
	case "line":
		modules = append(modules, CheckstyleModule{
			Name: "LineLength",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", max)},
			},
		})
	case "method":
		modules = append(modules, CheckstyleModule{
			Name: "MethodLength",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", max)},
			},
		})
	case "params", "parameters":
		modules = append(modules, CheckstyleModule{
			Name: "ParameterNumber",
			Properties: []CheckstyleProperty{
				{Name: "max", Value: fmt.Sprintf("%d", max)},
			},
		})
	}

	return modules
}
