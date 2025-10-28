package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// Rule represents a validation rule from the policy.
// Maps to PolicyRule in pkg/schema/types.go.
type Rule struct {
	ID       string                 `json:"id"`
	Enabled  bool                   `json:"enabled"`
	Category string                 `json:"category"`
	Severity string                 `json:"severity"` // "error", "warning", "info"
	Desc     string                 `json:"desc,omitempty"`
	When     *Selector              `json:"when,omitempty"`
	Check    map[string]interface{} `json:"check"` // Engine-specific config
	Remedy   *Remedy                `json:"remedy,omitempty"`
	Message  string                 `json:"message,omitempty"`
}

// Selector defines when a rule applies.
type Selector struct {
	Languages []string `json:"languages,omitempty"` // ["javascript", "typescript"]
	Include   []string `json:"include,omitempty"`   // ["src/**/*.js"]
	Exclude   []string `json:"exclude,omitempty"`   // ["**/*.test.js"]
	Branches  []string `json:"branches,omitempty"`  // ["main", "develop"]
	Roles     []string `json:"roles,omitempty"`     // ["dev", "reviewer"]
	Tags      []string `json:"tags,omitempty"`      // ["critical", "style"]
}

// Remedy contains auto-fix configuration.
type Remedy struct {
	Autofix bool                   `json:"autofix"`
	Tool    string                 `json:"tool,omitempty"`   // "prettier", "eslint"
	Config  map[string]interface{} `json:"config,omitempty"` // Tool-specific config
}

// ValidationResult is the outcome of validating files against a rule.
type ValidationResult struct {
	RuleID     string        `json:"ruleId"`
	Passed     bool          `json:"passed"`
	Violations []Violation   `json:"violations,omitempty"`
	Metrics    *Metrics      `json:"metrics,omitempty"`
	Duration   time.Duration `json:"-"` // Serialized separately
	Engine     string        `json:"engine"`
	Language   string        `json:"language,omitempty"`
}

// Violation represents a single rule violation.
type Violation struct {
	File       string                 `json:"file"`
	Line       int                    `json:"line"`              // 1-indexed, 0 if N/A
	Column     int                    `json:"column"`            // 1-indexed, 0 if N/A
	EndLine    int                    `json:"endLine,omitempty"` // For multi-line
	EndColumn  int                    `json:"endColumn,omitempty"`
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"` // "error", "warning", "info"
	RuleID     string                 `json:"ruleId"`
	Category   string                 `json:"category,omitempty"`
	Suggestion *Suggestion            `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"` // Extra info
}

// Suggestion represents an auto-fix suggestion.
type Suggestion struct {
	Desc        string `json:"desc"`                  // "Change to single quotes"
	Replacement string `json:"replacement,omitempty"` // Fixed text
	Diff        string `json:"diff,omitempty"`        // Unified diff
}

// Metrics contains validation metrics.
type Metrics struct {
	FilesProcessed int                    `json:"filesProcessed"`
	LinesProcessed int                    `json:"linesProcessed"`
	Custom         map[string]interface{} `json:"custom,omitempty"` // Engine-specific
}

// String returns a human-readable violation description.
// Format: "path/to/file.js:10:5: message [RULE-ID]"
func (v *Violation) String() string {
	loc := v.File
	if v.Line > 0 {
		loc = fmt.Sprintf("%s:%d", loc, v.Line)
		if v.Column > 0 {
			loc = fmt.Sprintf("%s:%d", loc, v.Column)
		}
	}
	return fmt.Sprintf("%s: %s [%s]", loc, v.Message, v.RuleID)
}

// MarshalJSON customizes JSON serialization for ValidationResult.
// Converts Duration to string (e.g., "1.5s").
func (r *ValidationResult) MarshalJSON() ([]byte, error) {
	type Alias ValidationResult
	return json.Marshal(&struct {
		Duration string `json:"duration"`
		*Alias
	}{
		Duration: r.Duration.String(),
		Alias:    (*Alias)(r),
	})
}

// GetString safely extracts a string value from Check config.
// Returns empty string if key doesn't exist or type mismatch.
func (r *Rule) GetString(key string) string {
	if v, ok := r.Check[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt safely extracts an int value from Check config.
// Returns 0 if key doesn't exist or type mismatch.
func (r *Rule) GetInt(key string) int {
	if v, ok := r.Check[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64: // JSON numbers are float64
			return int(val)
		}
	}
	return 0
}

// GetBool safely extracts a bool value from Check config.
// Returns false if key doesn't exist or type mismatch.
func (r *Rule) GetBool(key string) bool {
	if v, ok := r.Check[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetStringSlice safely extracts a []string from Check config.
// Returns nil if key doesn't exist or type mismatch.
func (r *Rule) GetStringSlice(key string) []string {
	if v, ok := r.Check[key]; ok {
		// Handle []interface{} from JSON unmarshaling
		if arr, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		// Handle native []string
		if arr, ok := v.([]string); ok {
			return arr
		}
	}
	return nil
}
