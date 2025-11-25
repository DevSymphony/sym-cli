package checkstyle

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts rules to Checkstyle XML configuration using LLM
type Converter struct{}

// NewConverter creates a new Checkstyle converter
func NewConverter() *Converter {
	return &Converter{}
}

// Name returns the linter name
func (c *Converter) Name() string {
	return "checkstyle"
}

// SupportedLanguages returns supported languages
func (c *Converter) SupportedLanguages() []string {
	return []string{"java"}
}

// GetLLMDescription returns a description of Checkstyle's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `Java style checks (naming, whitespace, imports, line length, complexity)
  - CAN: Class/method/variable naming, line/method length, indentation, import checks, cyclomatic complexity, JavaDoc
  - CANNOT: Runtime behavior, business logic, security vulnerabilities, advanced design patterns`
}

// checkstyleModule represents a Checkstyle module
type checkstyleModule struct {
	XMLName    xml.Name             `xml:"module"`
	Name       string               `xml:"name,attr"`
	Properties []checkstyleProperty `xml:"property,omitempty"`
	Modules    []checkstyleModule   `xml:"module,omitempty"`
}

// checkstyleProperty represents a property
type checkstyleProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

// checkstyleConfig represents root configuration
type checkstyleConfig struct {
	XMLName xml.Name           `xml:"module"`
	Name    string             `xml:"name,attr"`
	Modules []checkstyleModule `xml:"module"`
}

// ConvertRules converts user rules to Checkstyle configuration using LLM
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*adapter.LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Convert rules in parallel
	type moduleResult struct {
		index  int
		module *checkstyleModule
		err    error
	}

	results := make(chan moduleResult, len(rules))
	var wg sync.WaitGroup

	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, r schema.UserRule) {
			defer wg.Done()

			module, err := c.convertSingleRule(ctx, r, llmClient)
			results <- moduleResult{
				index:  idx,
				module: module,
				err:    err,
			}
		}(i, rule)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect modules
	var modules []checkstyleModule
	var errors []string

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("Rule %d: %v", result.index+1, result.err))
			continue
		}

		if result.module != nil {
			modules = append(modules, *result.module)
		}
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("no rules converted: %v", errors)
	}

	// Build Checkstyle configuration
	treeWalker := checkstyleModule{
		Name:    "TreeWalker",
		Modules: modules,
	}

	config := checkstyleConfig{
		Name:    "Checker",
		Modules: []checkstyleModule{treeWalker},
	}

	// Marshal to XML
	content, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add XML header
	xmlHeader := `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
    "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
    "https://checkstyle.org/dtds/configuration_1_3.dtd">
`
	fullContent := []byte(xmlHeader + string(content))

	return &adapter.LinterConfig{
		Filename: "checkstyle.xml",
		Content:  fullContent,
		Format:   "xml",
	}, nil
}

// convertSingleRule converts a single rule using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (*checkstyleModule, error) {
	systemPrompt := `You are a Checkstyle configuration expert. Convert natural language Java coding rules to Checkstyle modules.

Return ONLY a JSON object (no markdown fences):
{
  "module_name": "CheckstyleModuleName",
  "severity": "error|warning|info",
  "properties": {"key": "value", ...}
}

Common Checkstyle modules:
- Naming: TypeName, MethodName, ParameterName, LocalVariableName, ConstantName
- Length: LineLength, MethodLength, ParameterNumber, FileLength
- Style: Indentation, WhitespaceAround, NeedBraces, LeftCurly, RightCurly
- Imports: AvoidStarImport, IllegalImport, UnusedImports
- Complexity: CyclomaticComplexity, NPathComplexity
- JavaDoc: JavadocMethod, JavadocType, MissingJavadocMethod

If cannot convert, return:
{
  "module_name": "",
  "severity": "error",
  "properties": {}
}

Examples:

Input: "Methods must not exceed 50 lines"
Output:
{
  "module_name": "MethodLength",
  "severity": "error",
  "properties": {"max": "50"}
}

Input: "Use camelCase for local variables"
Output:
{
  "module_name": "LocalVariableName",
  "severity": "error",
  "properties": {"format": "^[a-z][a-zA-Z0-9]*$"}
}`

	userPrompt := fmt.Sprintf("Convert this Java rule to Checkstyle module:\n\n%s", rule.Say)

	response, err := llmClient.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result struct {
		ModuleName string            `json:"module_name"`
		Severity   string            `json:"severity"`
		Properties map[string]string `json:"properties"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if result.ModuleName == "" {
		return nil, nil
	}

	// Build module
	module := &checkstyleModule{
		Name:       result.ModuleName,
		Properties: []checkstyleProperty{},
	}

	// Add severity
	module.Properties = append(module.Properties, checkstyleProperty{
		Name:  "severity",
		Value: mapCheckstyleSeverity(result.Severity),
	})

	// Add other properties
	for key, value := range result.Properties {
		module.Properties = append(module.Properties, checkstyleProperty{
			Name:  key,
			Value: value,
		})
	}

	return module, nil
}

// mapCheckstyleSeverity maps severity to Checkstyle severity
func mapCheckstyleSeverity(severity string) string {
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
