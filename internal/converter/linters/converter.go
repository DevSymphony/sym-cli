package linters

import (
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// LinterConverter converts user rules to linter-specific configurations
type LinterConverter interface {
	// Name returns the linter name
	Name() string

	// SupportedLanguages returns the list of supported programming languages
	SupportedLanguages() []string

	// SupportedCategories returns the list of supported rule categories
	SupportedCategories() []string

	// Convert converts a user rule with inferred intent to linter configuration
	Convert(userRule *schema.UserRule, intent *llm.RuleIntent) (*LinterRule, error)

	// GenerateConfig generates the final linter configuration file from rules
	GenerateConfig(rules []*LinterRule) (*LinterConfig, error)
}

// LinterRule represents a single rule in linter-specific format
type LinterRule struct {
	ID       string         // Rule identifier
	Severity string         // error/warning/info
	Config   map[string]any // Linter-specific configuration
	Comment  string         // Optional comment (original "say")
}

// LinterConfig represents a linter configuration file
type LinterConfig struct {
	Format   string // "json", "xml", "yaml", "ini", "properties"
	Filename string // ".eslintrc.json", "checkstyle.xml", etc.
	Content  []byte // File content
}

// ConversionResult represents the result of converting rules for a linter
type ConversionResult struct {
	LinterName    string
	Config        *LinterConfig
	Rules         []*LinterRule
	Warnings      []string          // Conversion warnings
	Errors        []error           // Non-fatal errors
	RuleEngineMap map[string]string // Maps rule ID to engine name (eslint/checkstyle/pmd/llm-validator)
}
