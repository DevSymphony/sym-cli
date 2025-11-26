package pmd

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

// ConvertRules converts user rules to PMD configuration using LLM
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*adapter.LinterConfig, error) {
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	// Convert rules in parallel
	type ruleResult struct {
		index int
		rule  *pmdRule
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
	var pmdRules []pmdRule
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

	return &adapter.LinterConfig{
		Filename: "pmd.xml",
		Content:  fullContent,
		Format:   "xml",
	}, nil
}

// convertSingleRule converts a single rule using LLM
func (c *Converter) convertSingleRule(ctx context.Context, rule schema.UserRule, llmClient *llm.Client) (*pmdRule, error) {
	systemPrompt := `You are a PMD 7.x configuration expert. Convert natural language Java coding rules to PMD rule references.

Return ONLY a JSON object (no markdown fences):
{
  "rule_ref": "category/java/CATEGORY.xml/RuleName",
  "priority": 1-5
}

IMPORTANT: PMD 7.x uses "category/java/" prefix, NOT "rulesets/java/".

Common PMD 7.x rules:
- Best Practices: category/java/bestpractices.xml/UnusedPrivateMethod
- Code Style: category/java/codestyle.xml/ShortVariable
- Design: category/java/design.xml/TooManyMethods, category/java/design.xml/CyclomaticComplexity
- Error Prone: category/java/errorprone.xml/EmptyCatchBlock
- Security: category/java/security.xml/HardCodedCryptoKey

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
  "rule_ref": "category/java/bestpractices.xml/UnusedPrivateMethod",
  "priority": 2
}

Input: "No empty catch blocks"
Output:
{
  "rule_ref": "category/java/errorprone.xml/EmptyCatchBlock",
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

	if response == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}

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

	return &pmdRule{
		Ref:      result.RuleRef,
		Priority: result.Priority,
	}, nil
}
