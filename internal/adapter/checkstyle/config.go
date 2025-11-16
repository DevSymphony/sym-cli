package checkstyle

import (
	"encoding/xml"
	"fmt"
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
// The rule parameter should be a map or struct containing check configuration.
func generateConfig(rule interface{}) ([]byte, error) {
	// For now, generate a minimal valid config
	// In production, this should parse the rule and generate appropriate modules

	rootModule := CheckstyleConfig{
		Name: "Checker",
		Modules: []CheckstyleModule{
			{
				Name: "TreeWalker",
				Modules: []CheckstyleModule{
					// Example: Add modules based on rule
					// This will be expanded when integrated with converter
				},
			},
		},
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
