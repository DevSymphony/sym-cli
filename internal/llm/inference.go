package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

const systemPrompt = `You are a code linting rule analyzer. Extract structured information from natural language coding rules.

Extract:
1. **engine**: pattern|length|style|ast|llm-validator
   - Use "style" for code formatting rules (semicolons, quotes, indentation, spacing)
   - Use "pattern" for naming conventions or content matching
   - Use "length" for size/length constraints
   - Use "ast" for structural complexity rules
   - Use "llm-validator" for complex semantic rules that cannot be expressed with simple patterns

2. **category**: naming|formatting|security|error_handling|testing|documentation|dependency|commit|performance|architecture|custom

3. **target**: identifier|content|import|class|method|function|variable|file|line

4. **scope**: line|file|function|method|class|module|project

5. **patterns**: Array of regex patterns or keywords

6. **params**: JSON object with rule parameters. Examples:
   - For semicolons: {"semi": true} or {"semi": false}
   - For quotes: {"quote": "single"} or {"quote": "double"}
   - For indentation: {"indent": 2} or {"indent": 4}
   - For trailing commas: {"trailingComma": "always"} or {"trailingComma": "never"}
   - For case styles: {"case": "PascalCase"} or {"case": "camelCase"} or {"case": "snake_case"}
   - For length limits: {"max": 80}, {"min": 10}

7. **confidence**: 0.0-1.0

Examples:

Input: "All statements should end with semicolons"
Output:
{
  "engine": "style",
  "category": "formatting",
  "target": "content",
  "scope": "line",
  "patterns": [],
  "params": {"semi": true},
  "confidence": 0.95
}

Input: "Use single quotes for strings"
Output:
{
  "engine": "style",
  "category": "formatting",
  "target": "content",
  "scope": "file",
  "patterns": [],
  "params": {"quote": "single"},
  "confidence": 0.95
}

Input: "Class names must be PascalCase"
Output:
{
  "engine": "pattern",
  "category": "naming",
  "target": "class",
  "scope": "file",
  "patterns": ["^[A-Z][a-zA-Z0-9]*$"],
  "params": {"case": "PascalCase"},
  "confidence": 0.95
}

Input: "Lines should not exceed 80 characters"
Output:
{
  "engine": "length",
  "category": "formatting",
  "target": "line",
  "scope": "line",
  "patterns": [],
  "params": {"max": 80},
  "confidence": 0.95
}

Respond with valid JSON only.`

// Inferencer handles rule inference using LLM
type Inferencer struct {
	client *Client
	cache  *inferenceCache
}

// inferenceCache caches inference results
type inferenceCache struct {
	mu      sync.RWMutex
	entries map[string]*RuleIntent
}

func newInferenceCache() *inferenceCache {
	return &inferenceCache{
		entries: make(map[string]*RuleIntent),
	}
}

func (c *inferenceCache) Get(key string) (*RuleIntent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	intent, ok := c.entries[key]
	return intent, ok
}

func (c *inferenceCache) Set(key string, intent *RuleIntent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = intent
}

// NewInferencer creates a new rule inferencer
func NewInferencer(client *Client) *Inferencer {
	return &Inferencer{
		client: client,
		cache:  newInferenceCache(),
	}
}

// InferRuleIntent analyzes a rule and extracts structured intent
func (i *Inferencer) InferRuleIntent(ctx context.Context, req InferenceRequest) (*InferenceResult, error) {
	if i.client == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// Check cache
	cacheKey := strings.ToLower(strings.TrimSpace(req.Say))
	if cached, ok := i.cache.Get(cacheKey); ok {
		return &InferenceResult{
			Intent:    cached,
			Success:   true,
			UsedCache: true,
		}, nil
	}

	// Build prompt
	userPrompt := fmt.Sprintf("Rule: %s", req.Say)
	if req.Category != "" {
		userPrompt += fmt.Sprintf("\nCategory: %s", req.Category)
	}

	// Call LLM
	response, err := i.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM inference failed: %w", err)
	}

	// Parse response
	intent, err := parseIntent(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Merge hint params
	if intent.Params == nil {
		intent.Params = make(map[string]any)
	}
	for k, v := range req.Params {
		if _, exists := intent.Params[k]; !exists {
			intent.Params[k] = v
		}
	}

	// Cache result
	i.cache.Set(cacheKey, intent)

	return &InferenceResult{
		Intent:    intent,
		Success:   true,
		UsedCache: false,
	}, nil
}

// InferFromUserRule convenience method
func (i *Inferencer) InferFromUserRule(ctx context.Context, userRule *schema.UserRule) (*InferenceResult, error) {
	req := InferenceRequest{
		Say:      userRule.Say,
		Category: userRule.Category,
		Params:   userRule.Params,
	}
	return i.InferRuleIntent(ctx, req)
}

func parseIntent(response string) (*RuleIntent, error) {
	// Extract JSON from markdown code blocks
	jsonStr := strings.TrimSpace(response)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	var intent RuleIntent
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if intent.Engine == "" {
		return nil, fmt.Errorf("missing engine field")
	}
	if intent.Confidence == 0 {
		intent.Confidence = 0.5
	}

	return &intent, nil
}
