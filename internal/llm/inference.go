package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

const systemPrompt = `You are an expert code linting rule analyzer. Your task is to analyze natural language coding convention rules and extract structured information.

Given a natural language rule, extract:
1. **engine**: The type of validation engine needed (pattern/length/style/ast/custom)
   - "pattern": For naming conventions, forbidden patterns, import restrictions
   - "length": For size constraints (line length, file length, function length, parameter count)
   - "style": For code formatting (indentation, quotes, semicolons)
   - "ast": For structural analysis (cyclomatic complexity, nesting depth)
   - "custom": For rules that don't fit other categories

2. **category**: The rule category (naming/formatting/security/error_handling/testing/documentation/dependency/commit/performance/architecture/custom)

3. **target**: What the rule targets (identifier/content/import/class/method/function/variable/constant/file/line/etc.)

4. **scope**: The scope of validation (line/file/function/method/class/module/project)

5. **patterns**: Any regex patterns or keywords to match (e.g., for naming conventions)

6. **params**: Extracted parameters as JSON object
   - For length rules: {"max": number, "min": number}
   - For naming: {"case": "PascalCase|camelCase|snake_case|SCREAMING_SNAKE_CASE"}
   - For style: {"indent": number, "quote": "single|double", "semi": boolean}

7. **confidence**: Your confidence level (0.0-1.0) in this interpretation

8. **reasoning**: Brief explanation of why you chose this interpretation

Respond ONLY with valid JSON in this exact format:
{
  "engine": "pattern|length|style|ast|custom",
  "category": "naming|formatting|security|...",
  "target": "identifier|content|import|...",
  "scope": "line|file|function|...",
  "patterns": ["pattern1", "pattern2"],
  "params": {},
  "confidence": 0.95,
  "reasoning": "explanation here"
}`

// Inferencer handles rule inference using LLM
type Inferencer struct {
	client *Client
	cache  *inferenceCache
}

// inferenceCache caches inference results to minimize API calls
type inferenceCache struct {
	mu      sync.RWMutex
	entries map[string]*RuleIntent
}

// newInferenceCache creates a new cache
func newInferenceCache() *inferenceCache {
	return &inferenceCache{
		entries: make(map[string]*RuleIntent),
	}
}

// Get retrieves a cached result
func (c *inferenceCache) Get(key string) (*RuleIntent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	intent, ok := c.entries[key]
	return intent, ok
}

// Set stores a result in cache
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

// InferRuleIntent analyzes a natural language rule and extracts structured intent
func (i *Inferencer) InferRuleIntent(ctx context.Context, req InferenceRequest) (*InferenceResult, error) {
	// Check cache first
	cacheKey := i.makeCacheKey(req)
	if cached, ok := i.cache.Get(cacheKey); ok {
		return &InferenceResult{
			Intent:    cached,
			Success:   true,
			UsedCache: true,
		}, nil
	}

	// If no client available, use fallback immediately
	if i.client == nil {
		intent := i.fallbackInference(req)
		i.cache.Set(cacheKey, intent)
		return &InferenceResult{
			Intent:  intent,
			Success: true,
			Error:   fmt.Errorf("no LLM client, used fallback"),
		}, nil
	}

	// Build user prompt with hints
	userPrompt := i.buildUserPrompt(req)

	// Call OpenAI API
	response, err := i.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		// Fallback to pattern-based inference
		intent := i.fallbackInference(req)
		return &InferenceResult{
			Intent:  intent,
			Success: true,
			Error:   fmt.Errorf("LLM failed, used fallback: %w", err),
		}, nil
	}

	// Parse JSON response
	intent, err := i.parseIntentResponse(response)
	if err != nil {
		// Fallback to pattern-based inference
		fallbackIntent := i.fallbackInference(req)
		return &InferenceResult{
			Intent:  fallbackIntent,
			Success: true,
			Error:   fmt.Errorf("failed to parse LLM response, used fallback: %w", err),
		}, nil
	}

	// Cache the result
	i.cache.Set(cacheKey, intent)

	return &InferenceResult{
		Intent:    intent,
		Success:   true,
		UsedCache: false,
	}, nil
}

// makeCacheKey creates a cache key from the request
func (i *Inferencer) makeCacheKey(req InferenceRequest) string {
	// Simple key: normalized "say" text
	return strings.ToLower(strings.TrimSpace(req.Say))
}

// buildUserPrompt constructs the user prompt with hints
func (i *Inferencer) buildUserPrompt(req InferenceRequest) string {
	var sb strings.Builder

	sb.WriteString("Natural language rule:\n")
	sb.WriteString(req.Say)
	sb.WriteString("\n\n")

	if req.Category != "" {
		sb.WriteString(fmt.Sprintf("Hint - Category: %s\n", req.Category))
	}

	if len(req.Params) > 0 {
		paramsJSON, _ := json.Marshal(req.Params)
		sb.WriteString(fmt.Sprintf("Hint - Parameters: %s\n", string(paramsJSON)))
	}

	return sb.String()
}

// parseIntentResponse parses the JSON response from LLM
func (i *Inferencer) parseIntentResponse(response string) (*RuleIntent, error) {
	// Extract JSON from response (handle markdown code blocks)
	jsonStr := strings.TrimSpace(response)
	if strings.HasPrefix(jsonStr, "```json") {
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	} else if strings.HasPrefix(jsonStr, "```") {
		jsonStr = strings.TrimPrefix(jsonStr, "```")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	}

	var intent RuleIntent
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if intent.Engine == "" {
		return nil, fmt.Errorf("missing engine field")
	}

	if intent.Confidence == 0 {
		intent.Confidence = 0.5 // Default confidence if not provided
	}

	return &intent, nil
}

// fallbackInference provides pattern-based inference when LLM fails
func (i *Inferencer) fallbackInference(req InferenceRequest) *RuleIntent {
	say := strings.ToLower(req.Say)

	intent := &RuleIntent{
		Engine:     "custom",
		Category:   req.Category,
		Params:     make(map[string]any),
		Confidence: 0.4, // Low confidence for fallback
		Reasoning:  "Fallback pattern-based inference (LLM unavailable)",
	}

	// Copy hint params
	for k, v := range req.Params {
		intent.Params[k] = v
	}

	// Pattern-based detection
	if containsAny(say, []string{"name", "pascalcase", "camelcase", "snake_case", "kebab-case"}) {
		intent.Engine = "pattern"
		intent.Category = "naming"
		intent.Target = "identifier"
		intent.Confidence = 0.7
	} else if containsAny(say, []string{"line", "length", "characters", "max", "exceed", "complexity"}) {
		intent.Engine = "length"
		intent.Category = "formatting"
		intent.Scope = "line"

		// Detect complexity
		if containsAny(say, []string{"complexity", "cyclomatic"}) {
			intent.Category = "complexity"
			intent.Scope = "function"
		}

		// Try to extract number from params first
		if max, ok := req.Params["max"].(int); ok && max > 0 {
			intent.Params["max"] = max
			intent.Confidence = 0.75
		} else if max, ok := req.Params["complexity"].(int); ok && max > 0 {
			intent.Params["complexity"] = max
			intent.Confidence = 0.75
		} else if max := extractNumber(say); max > 0 {
			intent.Params["max"] = max
			intent.Confidence = 0.75
		}
	} else if containsAny(say, []string{"indent", "space", "tab", "quote", "semicolon"}) {
		intent.Engine = "style"
		intent.Category = "formatting"
		intent.Confidence = 0.7
	} else if containsAny(say, []string{"secret", "password", "token", "key", "credential", "hardcoded"}) {
		intent.Engine = "pattern"
		intent.Category = "security"
		intent.Target = "content"
		intent.Confidence = 0.8
	} else if containsAny(say, []string{"import", "require", "dependency", "layer"}) {
		intent.Engine = "pattern"
		intent.Category = "dependency"
		intent.Target = "import"
		intent.Confidence = 0.7
	}

	// Use category hint if provided and no better match
	if req.Category != "" && intent.Category == "" {
		intent.Category = req.Category
	}

	if intent.Category == "" {
		intent.Category = "custom"
	}

	return intent
}

// InferFromUserRule is a convenience method to infer from schema.UserRule
func (i *Inferencer) InferFromUserRule(ctx context.Context, userRule *schema.UserRule) (*InferenceResult, error) {
	req := InferenceRequest{
		Say:      userRule.Say,
		Category: userRule.Category,
		Params:   userRule.Params,
	}
	return i.InferRuleIntent(ctx, req)
}

// Helper functions

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func extractNumber(s string) int {
	// Simple number extraction (e.g., "100 characters" -> 100)
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}
