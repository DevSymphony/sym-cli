package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Extractor uses LLM to extract conventions from document content
type Extractor struct {
	provider llm.Provider
	verbose  bool
}

// NewExtractor creates a new Extractor instance
func NewExtractor(provider llm.Provider, verbose bool) *Extractor {
	return &Extractor{
		provider: provider,
		verbose:  verbose,
	}
}

// Extract analyzes document content and extracts coding conventions
func (e *Extractor) Extract(ctx context.Context, doc *DocumentContent) (*ExtractedConventions, error) {
	// Build prompt
	prompt := e.buildExtractionPrompt(doc.Content, filepath.Base(doc.Path))

	// Call LLM
	response, err := e.provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return nil, fmt.Errorf("LLM execution failed: %w", err)
	}

	// Parse response
	conventions, err := e.parseExtractionResponse(response, doc.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return conventions, nil
}

// buildExtractionPrompt builds the LLM prompt for convention extraction
func (e *Extractor) buildExtractionPrompt(content string, filename string) string {
	// Truncate content if too long
	maxContentLen := 40000
	if len(content) > maxContentLen {
		content = content[:maxContentLen] + "\n\n... (content truncated)"
	}

	return fmt.Sprintf(`You are a coding standards expert. Analyze the following document and extract coding conventions/rules from it.

SOURCE DOCUMENT: %s

DOCUMENT CONTENT:
---
%s
---

TASK: Extract all coding conventions, rules, and guidelines from this document.

OUTPUT FORMAT: Return ONLY valid JSON (no markdown fencing, no preamble text):
{
  "categories": [
    {"name": "category_name", "description": "1-2 sentence description of the category"}
  ],
  "rules": [
    {
      "id": "CATEGORY-001",
      "say": "Natural language description of what the rule enforces",
      "category": "category_name",
      "languages": ["javascript", "typescript"],
      "severity": "error",
      "message": "Short message shown when rule is violated",
      "example": "Optional example of correct/incorrect code"
    }
  ]
}

RULES FOR EXTRACTION:
1. Category names MUST be lowercase with underscores (e.g., "error_handling", "code_style")
2. Use standard categories when applicable: security, style, documentation, error_handling, architecture, performance, testing, naming, formatting
3. Rule IDs MUST be unique and follow pattern: UPPERCASE_CATEGORY-NNN (e.g., SEC-001, STYLE-001, DOC-001)
4. The "say" field MUST be a clear, actionable statement (e.g., "Use async/await instead of Promise callbacks")
5. Languages should be lowercase (e.g., "javascript", "python", "go", "java")
6. Severity MUST be one of: "error", "warning", "info"
7. If the document doesn't contain coding conventions, return: {"categories": [], "rules": []}
8. Extract ONLY coding conventions, not general documentation or explanations
9. Each rule should be specific and enforceable

EXAMPLES OF GOOD EXTRACTIONS:
- "All functions must have JSDoc comments" -> {"id": "DOC-001", "say": "All functions must have JSDoc comments", "category": "documentation", "severity": "warning"}
- "No console.log in production code" -> {"id": "STYLE-001", "say": "Remove all console.log statements from production code", "category": "style", "severity": "error"}
- "Use parameterized queries" -> {"id": "SEC-001", "say": "Use parameterized queries for all database operations to prevent SQL injection", "category": "security", "severity": "error"}
- "Function names must use camelCase" -> {"id": "NAMING-001", "say": "Function names must use camelCase convention", "category": "naming", "severity": "warning"}`, filename, content)
}

// parseExtractionResponse parses the LLM JSON response into conventions
func (e *Extractor) parseExtractionResponse(response string, source string) (*ExtractedConventions, error) {
	// Clean response - remove potential markdown fencing
	response = cleanJSONResponse(response)

	// Parse JSON
	var llmResponse LLMExtractionResponse
	if err := json.Unmarshal([]byte(response), &llmResponse); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w (response: %s)", err, truncateString(response, 200))
	}

	// Convert to schema types
	conventions := &ExtractedConventions{
		Source:     source,
		Categories: make([]schema.CategoryDef, 0, len(llmResponse.Categories)),
		Rules:      make([]schema.UserRule, 0, len(llmResponse.Rules)),
	}

	// Convert categories
	for _, cat := range llmResponse.Categories {
		if cat.Name == "" {
			continue
		}
		conventions.Categories = append(conventions.Categories, schema.CategoryDef{
			Name:        normalizeCategory(cat.Name),
			Description: cat.Description,
		})
	}

	// Convert rules
	for _, rule := range llmResponse.Rules {
		if rule.ID == "" || rule.Say == "" {
			continue
		}

		userRule := schema.UserRule{
			ID:        rule.ID,
			Say:       rule.Say,
			Category:  normalizeCategory(rule.Category),
			Languages: rule.Languages,
			Severity:  normalizeSeverity(rule.Severity),
			Message:   rule.Message,
			Example:   rule.Example,
		}
		conventions.Rules = append(conventions.Rules, userRule)
	}

	return conventions, nil
}

// cleanJSONResponse removes markdown fencing and extra whitespace from JSON response
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code fencing
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	response = strings.TrimSuffix(response, "```")

	return strings.TrimSpace(response)
}

// normalizeCategory normalizes category name to lowercase with underscores
func normalizeCategory(category string) string {
	if category == "" {
		return "general"
	}
	// Convert to lowercase and replace spaces/hyphens with underscores
	category = strings.ToLower(category)
	category = strings.ReplaceAll(category, " ", "_")
	category = strings.ReplaceAll(category, "-", "_")
	return category
}

// normalizeSeverity normalizes severity to valid values
func normalizeSeverity(severity string) string {
	severity = strings.ToLower(strings.TrimSpace(severity))
	switch severity {
	case "error", "err":
		return "error"
	case "warning", "warn":
		return "warning"
	case "info", "information":
		return "info"
	default:
		return "warning" // Default to warning
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
