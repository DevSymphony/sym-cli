package ast

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	adapterRegistry "github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates code structure using AST queries.
type Engine struct {
	adapterRegistry *adapterRegistry.Registry
	toolsDir        string
	workDir         string
}

// NewEngine creates a new AST engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the AST engine with Adapter Registry.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.toolsDir = config.ToolsDir
	e.workDir = config.WorkDir

	// Use provided adapter registry or create default
	if config.AdapterRegistry != nil {
		// Type assert to concrete type
		if reg, ok := config.AdapterRegistry.(*adapterRegistry.Registry); ok {
			e.adapterRegistry = reg
		} else {
			return fmt.Errorf("invalid adapter registry type")
		}
	} else {
		e.adapterRegistry = adapterRegistry.DefaultRegistry()
	}

	return nil
}

// Validate checks files against AST structure rules.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	// Filter files first - empty file list is valid without initialization
	files = e.filterFiles(files, rule.When)
	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:     rule.ID,
			Passed:     true,
			Violations: []core.Violation{},
		}, nil
	}

	// Check initialization
	if e.adapterRegistry == nil {
		return nil, fmt.Errorf("AST engine not initialized")
	}

	// Detect language from files and rule
	language := e.detectLanguage(rule, files)

	// Get appropriate adapter for language
	adp, err := e.adapterRegistry.GetAdapter(language, "ast")
	if err != nil {
		return nil, fmt.Errorf("no adapter found for language %s: %w", language, err)
	}

	// Type assert to adapter.Adapter
	astAdapter, ok := adp.(adapter.Adapter)
	if !ok {
		return nil, fmt.Errorf("invalid adapter type for language %s", language)
	}

	// Check if adapter is available, install if needed
	if err := astAdapter.CheckAvailability(ctx); err != nil {
		if installErr := astAdapter.Install(ctx, adapter.InstallConfig{}); installErr != nil {
			return nil, fmt.Errorf("adapter not available and installation failed: %w", installErr)
		}
	}

	// For ESLint adapter (JavaScript/TypeScript), use AST query
	if language == "javascript" || language == "typescript" || language == "jsx" || language == "tsx" {
		return e.validateWithESLint(ctx, rule, files, astAdapter)
	}

	// For other languages, generate config and execute
	config, err := astAdapter.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	output, err := astAdapter.Execute(ctx, config, files)
	if err != nil && output == nil {
		return nil, fmt.Errorf("adapter execution failed: %w", err)
	}

	adapterViolations, err := astAdapter.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// Convert adapter.Violation to core.Violation
	violations := make([]core.Violation, len(adapterViolations))
	for i, v := range adapterViolations {
		violations[i] = core.Violation{
			File:     v.File,
			Line:     v.Line,
			Column:   v.Column,
			Message:  v.Message,
			Severity: v.Severity,
			RuleID:   v.RuleID,
		}
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Engine:     "ast",
	}, nil
}

// validateWithESLint validates using ESLint AST queries.
func (e *Engine) validateWithESLint(ctx context.Context, rule core.Rule, files []string, adp adapter.Adapter) (*core.ValidationResult, error) {
	// Parse AST query
	query, err := eslint.ParseASTQuery(&rule)
	if err != nil {
		return nil, fmt.Errorf("invalid AST query: %w", err)
	}

	// Generate ESTree selector
	selector := eslint.GenerateESTreeSelector(query)

	// Generate ESLint config using no-restricted-syntax
	message := rule.Message
	if message == "" {
		message = fmt.Sprintf("AST rule %s violation", rule.ID)
	}

	config, err := e.generateESLintConfigWithSelector(rule, selector, message)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ESLint config: %w", err)
	}

	// Execute
	output, err := adp.Execute(ctx, config, files)
	if err != nil && output == nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Parse violations
	adapterViolations, err := adp.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// Convert violations
	violations := make([]core.Violation, len(adapterViolations))
	for i, v := range adapterViolations {
		violations[i] = core.Violation{
			File:     v.File,
			Line:     v.Line,
			Column:   v.Column,
			Message:  v.Message,
			Severity:  v.Severity,
			RuleID:   v.RuleID,
		}
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Engine:     "ast",
	}, nil
}

// GetCapabilities returns the engine's capabilities.
// Supported languages are determined dynamically based on registered adapters.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	caps := core.EngineCapabilities{
		Name:                "ast",
		SupportedCategories: []string{"error_handling", "custom"},
		SupportsAutofix:     false,
	}

	// If registry is available, get languages dynamically
	if e.adapterRegistry != nil {
		caps.SupportedLanguages = e.adapterRegistry.GetSupportedLanguages("ast")
	} else {
		// Fallback to default JS/TS
		caps.SupportedLanguages = []string{"javascript", "typescript", "jsx", "tsx"}
	}

	return caps
}

// Close cleans up the engine resources.
func (e *Engine) Close() error {
	return nil
}

// detectLanguage detects the primary language from files and rule configuration.
func (e *Engine) detectLanguage(rule core.Rule, files []string) string {
	// 1. Check rule.When.Languages if specified
	if rule.When != nil && len(rule.When.Languages) > 0 {
		return rule.When.Languages[0]
	}

	// 2. Detect from first file extension
	if len(files) > 0 {
		ext := strings.ToLower(filepath.Ext(files[0]))
		switch ext {
		case ".js":
			return "javascript"
		case ".ts":
			return "typescript"
		case ".jsx":
			return "jsx"
		case ".tsx":
			return "tsx"
		case ".java":
			return "java"
		case ".py":
			return "python"
		case ".go":
			return "go"
		case ".rs":
			return "rust"
		}
	}

	// 3. Default to JavaScript
	return "javascript"
}

// filterFiles filters files based on the when selector using proper glob matching.
func (e *Engine) filterFiles(files []string, when *core.Selector) []string {
	return core.FilterFiles(files, when)
}

// generateESLintConfigWithSelector generates ESLint config using no-restricted-syntax.
func (e *Engine) generateESLintConfigWithSelector(rule core.Rule, selector string, message string) ([]byte, error) {
	severity := eslint.MapSeverity(rule.Severity)

	config := map[string]interface{}{
		"env": map[string]bool{
			"es2021":  true,
			"node":    true,
			"browser": true,
		},
		"parserOptions": map[string]interface{}{
			"ecmaVersion": "latest",
			"sourceType":  "module",
		},
		"rules": map[string]interface{}{
			"no-restricted-syntax": []interface{}{
				severity,
				map[string]interface{}{
					"selector": selector,
					"message":  message,
				},
			},
		},
	}

	return eslint.MarshalConfig(config)
}
