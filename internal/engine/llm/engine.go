package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
	"github.com/DevSymphony/sym-cli/internal/envutil"
	"github.com/DevSymphony/sym-cli/internal/llm"
)

// Engine validates code using LLM-based analysis.
// Unlike other engines that use static analysis tools, this engine
// uses an LLM to understand and validate code against natural language rules.
type Engine struct {
	client *llm.Client
	config core.EngineConfig
}

// NewEngine creates a new LLM engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Initialize LLM client
	apiKey := envutil.GetAPIKey("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = envutil.GetAPIKey("OPENAI_API_KEY")
	}

	if apiKey == "" {
		return fmt.Errorf("LLM API key not found (ANTHROPIC_API_KEY or OPENAI_API_KEY in environment or .sym/.env)")
	}

	e.client = llm.NewClient(apiKey)
	return nil
}

// Validate validates files against an LLM-based rule.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	start := time.Now()

	// Filter files by selector
	files = core.FilterFiles(files, rule.When)

	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:   rule.ID,
			Passed:   true,
			Engine:   "llm-validator",
			Duration: time.Since(start),
		}, nil
	}

	violations := make([]core.Violation, 0)

	// Validate each file
	for _, file := range files {
		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			if e.config.Debug {
				fmt.Printf("⚠️  Failed to read file %s: %v\n", file, err)
			}
			continue
		}

		// Validate with LLM
		fileViolations, err := e.validateFile(ctx, rule, file, string(content))
		if err != nil {
			if e.config.Debug {
				fmt.Printf("⚠️  Failed to validate file %s: %v\n", file, err)
			}
			continue
		}

		violations = append(violations, fileViolations...)
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Duration:   time.Since(start),
		Engine:     "llm-validator",
		Metrics: &core.Metrics{
			FilesProcessed: len(files),
		},
	}, nil
}

// validateFile validates a single file using LLM
func (e *Engine) validateFile(ctx context.Context, rule core.Rule, filePath string, content string) ([]core.Violation, error) {
	// Build prompt for LLM
	systemPrompt := `You are a code reviewer. Check if the code violates the given coding convention.

Respond with JSON only:
{
  "violates": true/false,
  "description": "explanation of violation if any",
  "suggestion": "how to fix it if violated",
  "line": line_number_if_applicable (0 if not applicable)
}`

	userPrompt := fmt.Sprintf(`File: %s

Coding Convention:
%s

Code:
%s

Does this code violate the convention?`, filePath, rule.Desc, content)

	// Call LLM
	response, err := e.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	result := parseValidationResponse(response)
	if !result.Violates {
		return []core.Violation{}, nil
	}

	message := result.Description
	if result.Suggestion != "" {
		message += fmt.Sprintf(" | Suggestion: %s", result.Suggestion)
	}

	// Use custom message if provided in rule
	if rule.Message != "" {
		message = rule.Message + " | " + message
	}

	violation := core.Violation{
		RuleID:   rule.ID,
		Severity: rule.Severity,
		Message:  message,
		File:     filePath,
		Line:     result.Line,
		Category: rule.Category,
	}

	return []core.Violation{violation}, nil
}

// GetCapabilities returns engine capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name: "llm-validator",
		// LLM is language-agnostic - can understand any programming language
		SupportedLanguages: []string{
			"javascript", "typescript", "jsx", "tsx",
			"python", "go", "java", "rust", "c", "cpp",
			"ruby", "php", "swift", "kotlin", "scala",
		},
		SupportedCategories: []string{
			"convention", "style", "best-practice",
			"security", "performance", "maintainability",
		},
		SupportsAutofix:     false, // Future enhancement
		RequiresCompilation: false,
		ExternalTools:       []core.ToolRequirement{}, // No external tools needed
	}
}

// Close cleans up resources.
func (e *Engine) Close() error {
	return nil
}

// validationResponse represents the parsed LLM response
type validationResponse struct {
	Violates    bool
	Description string
	Suggestion  string
	Line        int
}

// parseValidationResponse parses the LLM response
func parseValidationResponse(response string) validationResponse {
	// Default to no violation
	result := validationResponse{
		Violates:    false,
		Description: "",
		Suggestion:  "",
		Line:        0,
	}

	lower := strings.ToLower(response)

	// Check if no violation
	if strings.Contains(lower, `"violates": false`) ||
		strings.Contains(lower, `"violates":false`) ||
		strings.Contains(lower, "does not violate") {
		return result
	}

	// Check if violates
	if strings.Contains(lower, `"violates": true`) ||
		strings.Contains(lower, `"violates":true`) {
		result.Violates = true

		// Extract description
		if desc := extractJSONField(response, "description"); desc != "" {
			result.Description = desc
		} else {
			result.Description = "Rule violation detected"
		}

		// Extract suggestion
		if sugg := extractJSONField(response, "suggestion"); sugg != "" {
			result.Suggestion = sugg
		}

		// Extract line number
		if lineStr := extractJSONField(response, "line"); lineStr != "" {
			// Parse line number
			var line int
			if _, err := fmt.Sscanf(lineStr, "%d", &line); err == nil {
				result.Line = line
			}
		}
	}

	return result
}

// extractJSONField extracts a field value from JSON response
func extractJSONField(response, field string) string {
	// Look for "field": "value"
	key := fmt.Sprintf(`"%s"`, field)
	idx := strings.Index(response, key)
	if idx == -1 {
		return ""
	}

	// Find : after field name
	colonIdx := strings.Index(response[idx:], ":") + idx
	if colonIdx <= idx {
		return ""
	}

	// Find opening quote or number
	start := colonIdx + 1
	for start < len(response) && (response[start] == ' ' || response[start] == '\t' || response[start] == '\n') {
		start++
	}

	if start >= len(response) {
		return ""
	}

	// Handle string value
	if response[start] == '"' {
		openIdx := start
		closeIdx := openIdx + 1
		for closeIdx < len(response) {
			if response[closeIdx] == '"' && (closeIdx == openIdx+1 || response[closeIdx-1] != '\\') {
				return response[openIdx+1 : closeIdx]
			}
			closeIdx++
		}
		return ""
	}

	// Handle numeric value
	end := start
	for end < len(response) && response[end] >= '0' && response[end] <= '9' {
		end++
	}
	if end > start {
		return response[start:end]
	}

	return ""
}
