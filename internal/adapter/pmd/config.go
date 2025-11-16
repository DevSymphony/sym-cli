package pmd

import (
	"encoding/xml"
	"fmt"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// PMDRuleset represents the root PMD ruleset.
type PMDRuleset struct {
	XMLName     xml.Name  `xml:"ruleset"`
	Name        string    `xml:"name,attr"`
	XMLNS       string    `xml:"xmlns,attr"`
	XMLNSXSI    string    `xml:"xmlns:xsi,attr"`
	XSISchema   string    `xml:"xsi:schemaLocation,attr"`
	Description string    `xml:"description"`
	Rules       []PMDRule `xml:"rule"`
}

// PMDRule represents a single PMD rule reference.
type PMDRule struct {
	XMLName    xml.Name      `xml:"rule"`
	Ref        string        `xml:"ref,attr,omitempty"`
	Name       string        `xml:"name,attr,omitempty"`
	Message    string        `xml:"message,attr,omitempty"`
	Priority   int           `xml:"priority,omitempty"`
	Properties []PMDProperty `xml:"properties>property,omitempty"`
}

// PMDProperty represents a property in PMD rule.
type PMDProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr,omitempty"`
}

// generateConfig generates PMD ruleset XML configuration from a rule.
func generateConfig(ruleInterface interface{}) ([]byte, error) {
	// Type assert to *core.Rule
	rule, ok := ruleInterface.(*core.Rule)
	if !ok {
		return nil, fmt.Errorf("expected *core.Rule, got %T", ruleInterface)
	}

	// Generate PMD rules based on AST node type
	pmdRules := generatePMDRules(rule)

	ruleset := PMDRuleset{
		Name:        "Symphony Convention Rules",
		XMLNS:       "http://pmd.sourceforge.net/ruleset/2.0.0",
		XMLNSXSI:    "http://www.w3.org/2001/XMLSchema-instance",
		XSISchema:   "http://pmd.sourceforge.net/ruleset/2.0.0 https://pmd.sourceforge.io/ruleset_2_0_0.xsd",
		Description: "Generated PMD ruleset from Symphony policy",
		Rules:       pmdRules,
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(ruleset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PMD ruleset: %w", err)
	}

	// Add XML header
	xmlHeader := `<?xml version="1.0"?>
`

	return []byte(xmlHeader + string(output)), nil
}

// generatePMDRules maps AST rules to PMD built-in rules.
func generatePMDRules(rule *core.Rule) []PMDRule {
	var rules []PMDRule

	node := rule.GetString("node")

	// Map common AST patterns to PMD rules
	switch node {
	case "MethodCallExpr":
		// Check if it's System.out usage
		if where, ok := rule.Check["where"].(map[string]interface{}); ok {
			if scope, ok := where["scope"].(string); ok && scope == "System.out" {
				rules = append(rules, PMDRule{
					Ref:      "category/java/bestpractices.xml/SystemPrintln",
					Priority: 3,
				})
			}
		}

	case "CatchClause":
		// Check if it's empty catch or generic exception
		if where, ok := rule.Check["where"].(map[string]interface{}); ok {
			// Empty catch block
			if size, ok := where["body.statements.size"].(float64); ok && size == 0 {
				rules = append(rules, PMDRule{
					Ref:      "category/java/errorprone.xml/EmptyCatchBlock",
					Priority: 3,
				})
			}
			// Generic Exception catch
			if paramType, ok := where["parameter.type.name"].(string); ok && paramType == "Exception" {
				rules = append(rules, PMDRule{
					Ref:      "category/java/design.xml/AvoidCatchingGenericException",
					Priority: 3,
				})
			}
		}

	case "MethodDeclaration":
		// Check for missing Javadoc
		if where, ok := rule.Check["where"].(map[string]interface{}); ok {
			if isPublic, ok := where["isPublic"].(bool); ok && isPublic {
				if hasJavadoc, ok := where["hasJavadoc"].(bool); ok && !hasJavadoc {
					rules = append(rules, PMDRule{
						Ref:      "category/java/documentation.xml/CommentRequired",
						Priority: 3,
						Properties: []PMDProperty{
							{Name: "methodWithOverrideCommentRequirement", Value: "Ignored"},
							{Name: "accessorCommentRequirement", Value: "Ignored"},
							{Name: "classCommentRequirement", Value: "Ignored"},
							{Name: "fieldCommentRequirement", Value: "Ignored"},
							{Name: "publicMethodCommentRequirement", Value: "Required"},
							{Name: "protectedMethodCommentRequirement", Value: "Ignored"},
							{Name: "enumCommentRequirement", Value: "Ignored"},
							{Name: "violationSuppressRegex", Value: ".*main\\(.*"},
						},
					})
				}
			}
		}
	}

	return rules
}
