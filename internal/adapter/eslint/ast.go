package eslint

import (
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// ASTQuery represents a parsed AST query from a rule.
type ASTQuery struct {
	Node     string                 `json:"node"`
	Where    map[string]interface{} `json:"where,omitempty"`
	Has      []string               `json:"has,omitempty"`
	NotHas   []string               `json:"notHas,omitempty"`
	Language string                 `json:"language,omitempty"`
}

// ParseASTQuery extracts AST query from a rule's check field.
func ParseASTQuery(rule *core.Rule) (*ASTQuery, error) {
	node, ok := rule.Check["node"].(string)
	if !ok || node == "" {
		return nil, fmt.Errorf("AST rule requires 'node' field")
	}

	query := &ASTQuery{
		Node: node,
	}

	if where, ok := rule.Check["where"].(map[string]interface{}); ok {
		query.Where = where
	}

	if has, ok := rule.Check["has"].([]interface{}); ok {
		query.Has = interfaceSliceToStringSlice(has)
	}

	if notHas, ok := rule.Check["notHas"].([]interface{}); ok {
		query.NotHas = interfaceSliceToStringSlice(notHas)
	}

	if lang, ok := rule.Check["language"].(string); ok {
		query.Language = lang
	}

	return query, nil
}

// GenerateESTreeSelector generates ESLint AST selector from AST query.
// Uses ESLint's no-restricted-syntax with ESTree selectors.
func GenerateESTreeSelector(query *ASTQuery) string {
	var parts []string

	// Start with node type
	parts = append(parts, query.Node)

	// Add where conditions as attribute selectors
	if len(query.Where) > 0 {
		for key, value := range query.Where {
			selector := generateAttributeSelector(key, value)
			if selector != "" {
				parts = append(parts, selector)
			}
		}
	}

	// Combine into single selector
	selector := strings.Join(parts, "")

	// For "has" queries, use descendant combinator
	if len(query.Has) > 0 {
		// ESLint selector: "FunctionDeclaration:not(:has(TryStatement))"
		for _, nodeType := range query.Has {
			selector = fmt.Sprintf("%s:not(:has(%s))", selector, nodeType)
		}
	}

	// For "notHas" queries, check presence
	if len(query.NotHas) > 0 {
		for _, nodeType := range query.NotHas {
			selector = fmt.Sprintf("%s:has(%s)", selector, nodeType)
		}
	}

	return selector
}

// generateAttributeSelector creates an attribute selector for ESTree.
func generateAttributeSelector(key string, value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return fmt.Sprintf("[%s=true]", key)
		}
		return fmt.Sprintf("[%s=false]", key)
	case string:
		return fmt.Sprintf("[%s=\"%s\"]", key, v)
	case float64, int:
		return fmt.Sprintf("[%s=%v]", key, v)
	case map[string]interface{}:
		// Handle operators
		if eq, ok := v["eq"]; ok {
			return generateAttributeSelector(key, eq)
		}
		// Other operators not supported in ESTree selectors
	}
	return ""
}

// interfaceSliceToStringSlice converts []interface{} to []string.
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
