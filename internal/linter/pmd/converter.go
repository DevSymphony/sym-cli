package pmd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

// Converter converts rules to PMD XML configuration using LLM
type Converter struct{}

// NewConverter creates a new PMD converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "pmd"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"java"}
}

// GetLLMDescription returns a description of PMD's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `Java code quality analysis (unused code, empty blocks, complexity, design issues)
  - CAN: Unused private methods, empty catch blocks, too many methods, hardcoded crypto keys, cyclomatic complexity, error handling patterns
  - CANNOT: Code formatting, whitespace, naming conventions (use Checkstyle instead), complex business logic validation`
}

// GetRoutingHints returns routing rules for LLM to decide when to use PMD
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For Java code quality (unused code, complexity, empty catch blocks) → use pmd",
		"For Java error handling patterns (empty catch, exception handling) → use pmd",
		"NEVER use pmd for naming conventions - use checkstyle instead",
	}
}

// pmdRuleset represents PMD ruleset
type pmdRuleset struct {
	XMLName     xml.Name  `xml:"ruleset"`
	Name        string    `xml:"name,attr"`
	XMLNS       string    `xml:"xmlns,attr"`
	XMLNSXSI    string    `xml:"xmlns:xsi,attr"`
	XSISchema   string    `xml:"xsi:schemaLocation,attr"`
	Description string    `xml:"description"`
	Rules       []pmdRule `xml:"rule"`
}

// pmdRule represents a PMD rule
type pmdRule struct {
	XMLName  xml.Name `xml:"rule"`
	Ref      string   `xml:"ref,attr"`
	Priority int      `xml:"priority,omitempty"`
}

// ConvertSingleRule converts ONE user rule to PMD rule.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by PMD (skip),
//	(nil, error) on actual conversion error.
//
// Note: Concurrency is handled by the main converter.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	pmdRule, err := c.convertToPMDRule(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	if pmdRule == nil {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data:   pmdRule,
	}, nil
}

// BuildConfig assembles PMD XML configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var pmdRules []pmdRule
	for _, r := range results {
		rule, ok := r.Data.(*pmdRule)
		if !ok {
			continue
		}
		pmdRules = append(pmdRules, *rule)
	}

	if len(pmdRules) == 0 {
		return nil, nil
	}

	// Build PMD ruleset
	ruleset := pmdRuleset{
		Name:        "Symphony Rules",
		XMLNS:       "http://pmd.sourceforge.net/ruleset/2.0.0",
		XMLNSXSI:    "http://www.w3.org/2001/XMLSchema-instance",
		XSISchema:   "http://pmd.sourceforge.net/ruleset/2.0.0 https://pmd.sourceforge.io/ruleset_2_0_0.xsd",
		Description: "Generated from Symphony user policy",
		Rules:       pmdRules,
	}

	// Marshal to XML
	content, err := xml.MarshalIndent(ruleset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	xmlHeader := `<?xml version="1.0"?>` + "\n"
	fullContent := []byte(xmlHeader + string(content))

	return &linter.LinterConfig{
		Filename: "pmd.xml",
		Content:  fullContent,
		Format:   "xml",
	}, nil
}

// convertToPMDRule converts a single rule using LLM
func (c *Converter) convertToPMDRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*pmdRule, error) {
	systemPrompt := `You are a PMD 7.x configuration expert. Convert natural language Java coding rules to PMD rule references.

Return ONLY a JSON object with exactly these two fields (no other fields):
{
  "rule_ref": "category/java/category.xml/RuleName",
  "priority": 1
}

Valid PMD 7.x categories and rules:
- category/java/bestpractices.xml/UnusedPrivateMethod
- category/java/bestpractices.xml/UnusedLocalVariable
- category/java/bestpractices.xml/UnusedFormalParameter
- category/java/bestpractices.xml/AvoidReassigningParameters
- category/java/codestyle.xml/ShortVariable
- category/java/codestyle.xml/LongVariable
- category/java/codestyle.xml/ShortMethodName
- category/java/codestyle.xml/ClassNamingConventions
- category/java/codestyle.xml/MethodNamingConventions
- category/java/codestyle.xml/FieldNamingConventions
- category/java/codestyle.xml/UnnecessaryImport
- category/java/design.xml/TooManyMethods
- category/java/design.xml/ExcessiveMethodLength
- category/java/design.xml/ExcessiveParameterList
- category/java/design.xml/CyclomaticComplexity
- category/java/design.xml/NPathComplexity
- category/java/design.xml/GodClass
- category/java/errorprone.xml/EmptyCatchBlock
- category/java/errorprone.xml/AvoidCatchingNPE
- category/java/errorprone.xml/EmptyIfStmt
- category/java/security.xml/HardCodedCryptoKey

Priority: 1=High, 2=Medium-High, 3=Medium, 4=Low, 5=Info

If the rule cannot be mapped to a valid PMD rule, return:
{
  "rule_ref": "",
  "priority": 3
}

IMPORTANT: Return ONLY the JSON object. Do NOT include description, message, or any other fields.`

	userPrompt := fmt.Sprintf("Convert this Java rule to PMD rule reference:\n\n%s", rule.Say)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response - extract JSON object
	response = linter.CleanJSONResponse(response)

	// Find JSON object boundaries to handle extra text
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no valid JSON object found in response")
	}
	response = response[startIdx : endIdx+1]

	var result struct {
		RuleRef  string `json:"rule_ref"`
		Priority int    `json:"priority"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	if result.RuleRef == "" {
		return nil, nil
	}

	// Validate rule_ref format: must start with "category/java/"
	if !strings.HasPrefix(result.RuleRef, "category/java/") {
		// Try to fix old format (rulesets/java/...) to new format (category/java/...)
		if strings.HasPrefix(result.RuleRef, "rulesets/java/") {
			result.RuleRef = strings.Replace(result.RuleRef, "rulesets/java/", "category/java/", 1)
		} else {
			return nil, nil // Invalid format, skip this rule
		}
	}

	// Validate priority range
	if result.Priority < 1 {
		result.Priority = 3
	}
	if result.Priority > 5 {
		result.Priority = 5
	}

	return &pmdRule{
		Ref:      result.RuleRef,
		Priority: result.Priority,
	}, nil
}
