package adapter

import (
	"context"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Adapter wraps external tools (ESLint, Prettier, etc.) for use by engines.
//
// Design:
// - Adapters handle tool installation, config generation, execution
// - Engines delegate to adapters for language-specific validation
// - One adapter per tool (ESLintAdapter, PrettierAdapter, etc.)
type Adapter interface {
	// Name returns the adapter name (e.g., "eslint", "prettier").
	Name() string

	// GetCapabilities returns the adapter's capabilities.
	// This includes supported languages, categories, and version info.
	GetCapabilities() AdapterCapabilities

	// CheckAvailability checks if the tool is installed and usable.
	// Returns nil if available, error with details if not.
	CheckAvailability(ctx context.Context) error

	// Install installs the tool if not available.
	// Returns error if installation fails.
	Install(ctx context.Context, config InstallConfig) error

	// Execute runs the tool with the given config and files.
	// Config is read from .sym directory (e.g., .sym/.eslintrc.json).
	// Returns raw tool output.
	Execute(ctx context.Context, config []byte, files []string) (*ToolOutput, error)

	// ParseOutput converts tool output to standard violations.
	ParseOutput(output *ToolOutput) ([]Violation, error)
}

// AdapterCapabilities describes what an adapter can do.
type AdapterCapabilities struct {
	// Name is the adapter identifier (e.g., "eslint", "checkstyle").
	Name string

	// SupportedLanguages lists languages this adapter can validate.
	// Examples: ["javascript", "typescript", "java"]
	SupportedLanguages []string

	// SupportedCategories lists rule categories this adapter can handle.
	// Examples: ["pattern", "length", "style", "ast", "complexity"]
	SupportedCategories []string

	// Version is the tool version (e.g., "8.0.0", "10.12.0").
	Version string
}

// InstallConfig holds tool installation settings.
type InstallConfig struct {
	// ToolsDir is where to install the tool.
	// Default: ~/.sym/tools
	ToolsDir string

	// Version is the tool version to install.
	// Empty = latest
	Version string

	// Force reinstalls even if already installed.
	Force bool
}

// ToolOutput is the raw output from a tool execution.
type ToolOutput struct {
	// Stdout is the standard output.
	Stdout string

	// Stderr is the error output.
	Stderr string

	// ExitCode is the process exit code.
	ExitCode int

	// Duration is how long the tool took to run.
	Duration string
}

// Violation represents a single violation found by a tool.
// This is a simplified version that adapters return.
// Engines convert this to core.Violation.
type Violation struct {
	File     string
	Line     int
	Column   int
	Message  string
	Severity string // "error", "warning", "info"
	RuleID   string
}

// LinterConverter converts user rules to native linter configuration using LLM.
// This interface is implemented by each linter's converter (e.g., ESLintConverter).
type LinterConverter interface {
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

	// ConvertRules converts user rules to native linter configuration using LLM.
	// Returns ConversionResult with per-rule success/failure tracking for fallback support.
	ConvertRules(ctx context.Context, rules []schema.UserRule, provider llm.Provider) (*ConversionResult, error)
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
