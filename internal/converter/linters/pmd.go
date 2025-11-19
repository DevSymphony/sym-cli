package linters

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// PMDLinterConverter converts rules to PMD XML using LLM
type PMDLinterConverter struct{}

// NewPMDLinterConverter creates a new PMD converter
func NewPMDLinterConverter() *PMDLinterConverter {
	return &PMDLinterConverter{}
}

// Name returns the linter name
func (c *PMDLinterConverter) Name() string {
	return "pmd"
}

// SupportedLanguages returns supported languages
func (c *PMDLinterConverter) SupportedLanguages() []string {
	return []string{"java"}
}

// PMDRuleset represents PMD ruleset
type PMDRuleset struct {
	XMLName     xml.Name  `xml:"ruleset"`
	Name        string    `xml:"name,attr"`
	XMLNS       string    `xml:"xmlns,attr"`
	XMLNSXSI    string    `xml:"xmlns:xsi,attr"`
	XSISchema   string    `xml:"xsi:schemaLocation,attr"`
	Description string    `xml:"description"`
	Rules       []PMDRule `xml:"rule"`
}

// PMDRule represents a PMD rule
type PMDRule struct {
	XMLName  xml.Name `xml:"rule"`
	Ref      string   `xml:"ref,attr"`
	Priority int      `xml:"priority,omitempty"`
}

// ConvertRules converts user rules to PMD configuration using LLM
func (c *PMDLinterConverter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Convert rules in parallel
	type ruleResult struct {
		index int
		rule  *PMDRule
		err   error
	}

	results := make(chan ruleResult, len(rules))
	var wg sync.WaitGroup

	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, r schema.UserRule) {
			defer wg.Done()

			pmdRule, err := c.convertSingleRule(ctx, r, llmClient)
			results <- ruleResult{
				index: idx,
				rule:  pmdRule,
				err:   err,
			}
		}(i, rule)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect rules
	var pmdRules []PMDRule
	var errors []string

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("Rule %d: %v", result.index+1, result.err))
			continue
		}

		if result.rule != nil {
			pmdRules = append(pmdRules, *result.rule)
		}
	}

	if len(pmdRules) == 0 {
		return nil, fmt.Errorf("no rules converted: %v", errors)
	}

	// Build PMD ruleset
	ruleset := PMDRuleset{
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

	return &LinterConfig{
		Filename: "pmd.xml",
		Content:  fullContent,
		Format:   "xml",
	}, nil
}

// convertSingleRule converts a single rule using LLM
func (c *PMDLinterConverter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (*PMDRule, error) {
	systemPrompt := `You are a PMD configuration expert. Convert natural language Java coding rules to PMD rule references.

Return ONLY a JSON object (no markdown fences):
{
  "rule_ref": "category/java/ruleset.xml/RuleName",
  "priority": 1-5
}

Common PMD rules:
- Best Practices: rulesets/java/bestpractices.xml/UnusedPrivateMethod
- Code Style: rulesets/java/codestyle.xml/ShortVariable
- Design: rulesets/java/design.xml/TooManyMethods
- Error Handling: rulesets/java/errorprone.xml/EmptyCatchBlock
- Security: rulesets/java/security.xml/HardCodedCryptoKey

Priority levels: 1=High, 2=Medium-High, 3=Medium, 4=Low, 5=Info

If cannot convert, return:
{
  "rule_ref": "",
  "priority": 3
}

Example:

Input: "Avoid unused private methods"
Output:
{
  "rule_ref": "rulesets/java/bestpractices.xml/UnusedPrivateMethod",
  "priority": 2
}`

	userPrompt := fmt.Sprintf("Convert this Java rule to PMD rule reference:\n\n%s", rule.Say)

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
		RuleRef  string `json:"rule_ref"`
		Priority int    `json:"priority"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if result.RuleRef == "" {
		return nil, nil
	}

	return &PMDRule{
		Ref:      result.RuleRef,
		Priority: result.Priority,
	}, nil
}
