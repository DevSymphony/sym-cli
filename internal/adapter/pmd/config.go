package pmd

import (
	"encoding/xml"
	"fmt"
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
func generateConfig(rule interface{}) ([]byte, error) {
	// For now, generate a minimal valid ruleset
	// In production, this should parse the rule and generate appropriate rules

	ruleset := PMDRuleset{
		Name:        "Symphony Convention Rules",
		XMLNS:       "http://pmd.sourceforge.net/ruleset/2.0.0",
		XMLNSXSI:    "http://www.w3.org/2001/XMLSchema-instance",
		XSISchema:   "http://pmd.sourceforge.net/ruleset/2.0.0 https://pmd.sourceforge.io/ruleset_2_0_0.xsd",
		Description: "Generated PMD ruleset from Symphony policy",
		Rules:       []PMDRule{
			// Example rules - will be expanded based on rule parameter
		},
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
