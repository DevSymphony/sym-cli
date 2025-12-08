package linter

import (
	"context"
)

// Linter wraps external linting tools (ESLint, Prettier, etc.) for use by engines.
//
// Design:
// - Linters handle tool installation, config generation, execution
// - Engines delegate to linters for language-specific validation
// - One linter per tool (ESLint, Prettier, etc.)
type Linter interface {
	// Name returns the linter name (e.g., "eslint", "prettier").
	Name() string

	// GetCapabilities returns the linter's capabilities.
	// This includes supported languages, categories, and version info.
	GetCapabilities() Capabilities

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

// Capabilities describes what a linter can do.
type Capabilities struct {
	// Name is the linter identifier (e.g., "eslint", "checkstyle").
	Name string

	// SupportedLanguages lists languages this linter can validate.
	// Examples: ["javascript", "typescript", "java"]
	SupportedLanguages []string

	// SupportedCategories lists rule categories this linter can handle.
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
// This is a simplified version that linters return.
// Engines convert this to core.Violation.
type Violation struct {
	File     string
	Line     int
	Column   int
	Message  string
	Severity string // "error", "warning", "info"
	RuleID   string
}
