package llm

// RuleIntent represents the structured interpretation of a natural language rule
type RuleIntent struct {
	Engine     string         // "pattern", "length", "style", "ast", "custom"
	Category   string         // "naming", "formatting", "security", "error_handling", etc.
	Target     string         // "identifier", "content", "import", "class", "method", etc.
	Scope      string         // "line", "file", "function", "method", "class", etc.
	Patterns   []string       // Extracted regex patterns or keywords
	Params     map[string]any // Extracted parameters (e.g., max, min, indent, quote)
	Confidence float64        // 0.0-1.0 confidence score from LLM
	Reasoning  string         // Explanation of why this intent was inferred
}

// InferenceResult represents the result of rule inference
type InferenceResult struct {
	Intent    *RuleIntent
	Success   bool
	Error     error
	UsedCache bool // Whether result came from cache
}

// InferenceRequest represents a request to infer rule intent
type InferenceRequest struct {
	Say      string         // Natural language rule
	Category string         // Optional category hint
	Params   map[string]any // Optional parameter hints
}
