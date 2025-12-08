package linter

import (
	"context"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Converter converts user rules to native linter configuration using LLM.
// This interface is implemented by each linter's converter (e.g., ESLintConverter).
//
// The main converter (internal/converter) handles all concurrency control.
// Individual linter converters only implement single-rule conversion logic.
type Converter interface {
	// Name returns the linter name (e.g., "eslint", "checkstyle", "pmd")
	Name() string

	// SupportedLanguages returns the languages this linter supports
	SupportedLanguages() []string

	// GetLLMDescription returns a description of the linter's capabilities for LLM routing.
	// This is used in the LLM prompt to help route rules to appropriate linters.
	GetLLMDescription() string

	// GetRoutingHints returns routing rules for LLM to decide when to use this linter.
	// Each hint is a rule like "For Java naming rules → ALWAYS use checkstyle".
	// These hints are collected and included in the LLM prompt for rule routing.
	GetRoutingHints() []string

	// ConvertSingleRule converts ONE user rule to linter-specific data.
	// Returns (result, nil) on success,
	//         (nil, nil) if rule cannot be converted by this linter (skip),
	//         (nil, error) on actual conversion error.
	// Note: Concurrency is handled by the main converter, not here.
	ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*SingleRuleResult, error)

	// BuildConfig assembles final linter config from successful conversions.
	// Called by main converter after collecting all successful SingleRuleResults.
	BuildConfig(results []*SingleRuleResult) (*LinterConfig, error)
}

// SingleRuleResult represents the conversion result for a single rule.
// The Data field contains linter-specific data that BuildConfig understands.
type SingleRuleResult struct {
	RuleID string      // Original user rule ID
	Data   interface{} // Linter-specific data (e.g., ESLint rule config, Checkstyle module)
}

// LinterConfig represents a generated configuration file.
type LinterConfig struct {
	Filename string // e.g., ".eslintrc.json", "checkstyle.xml"
	Content  []byte // File content
	Format   string // "json", "xml", "yaml"
}

// ConversionResult contains the conversion output with per-rule tracking.
// This allows the main converter to know which rules succeeded vs failed,
// enabling fallback to llm-validator for failed rules.
type ConversionResult struct {
	Config       *LinterConfig // Generated config file (may be nil if all rules failed)
	SuccessRules []string      // Rule IDs that converted successfully
	FailedRules  []string      // Rule IDs that couldn't be converted (fallback to llm-validator)
}
