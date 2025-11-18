package linters

import (
	"context"

	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// LinterConverter converts user rules to native linter configuration using LLM
type LinterConverter interface {
	// Name returns the linter name (e.g., "eslint", "checkstyle", "pmd")
	Name() string

	// SupportedLanguages returns the languages this linter supports
	SupportedLanguages() []string

	// ConvertRules converts user rules to native linter configuration using LLM
	// This is the main entry point for parallel conversion
	ConvertRules(ctx context.Context, rules []schema.UserRule, llmClient *llm.Client) (*LinterConfig, error)
}

// LinterConfig represents a generated configuration file
type LinterConfig struct {
	Filename string // e.g., ".eslintrc.json", "checkstyle.xml"
	Content  []byte // File content
	Format   string // "json", "xml", "yaml"
}
