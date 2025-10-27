package ast

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates code structure using AST queries.
type Engine struct {
	eslint   *eslint.Adapter
	toolsDir string
	workDir  string
}

// NewEngine creates a new AST engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the AST engine with ESLint adapter.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.toolsDir = config.ToolsDir
	e.workDir = config.WorkDir

	e.eslint = eslint.NewAdapter(e.toolsDir, e.workDir)

	// Check ESLint availability
	if err := e.eslint.CheckAvailability(ctx); err != nil {
		// Try to install
		if installErr := e.eslint.Install(ctx, adapter.InstallConfig{}); installErr != nil {
			return fmt.Errorf("eslint not available and installation failed: %w", installErr)
		}
	}

	return nil
}

// Validate checks files against AST structure rules.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	if e.eslint == nil {
		return nil, fmt.Errorf("AST engine not initialized")
	}

	files = e.filterFiles(files, rule.When)
	if len(files) == 0 {
		return &core.ValidationResult{Violations: []core.Violation{}}, nil
	}

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

	// Execute ESLint
	output, err := e.eslint.Execute(ctx, config, files)
	if err != nil && output == nil {
		return nil, fmt.Errorf("eslint execution failed: %w", err)
	}

	// Parse violations
	adapterViolations, err := e.eslint.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ESLint output: %w", err)
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

// GetCapabilities returns the engine's capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name:                "ast",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"error_handling", "custom"},
		SupportsAutofix:     false,
	}
}

// Close cleans up the engine resources.
func (e *Engine) Close() error {
	return nil
}

// filterFiles filters files based on the when selector.
func (e *Engine) filterFiles(files []string, when *core.Selector) []string {
	if when == nil {
		return files
	}

	var filtered []string
	for _, file := range files {
		if e.matchesSelector(file, when) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// matchesSelector checks if a file matches the selector criteria.
func (e *Engine) matchesSelector(file string, when *core.Selector) bool {
	// Language filter
	if len(when.Languages) > 0 {
		ext := filepath.Ext(file)
		matched := false
		for _, lang := range when.Languages {
			if matchesLanguage(ext, lang) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Include/Exclude filters
	if len(when.Include) > 0 {
		matched := false
		for _, pattern := range when.Include {
			if matchGlob(file, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(when.Exclude) > 0 {
		for _, pattern := range when.Exclude {
			if matchGlob(file, pattern) {
				return false
			}
		}
	}

	return true
}

// matchesLanguage checks if file extension matches language.
func matchesLanguage(ext, lang string) bool {
	langExtMap := map[string][]string{
		"javascript": {".js", ".mjs", ".cjs"},
		"typescript": {".ts", ".mts", ".cts"},
		"jsx":        {".jsx"},
		"tsx":        {".tsx"},
	}

	exts, ok := langExtMap[lang]
	if !ok {
		return false
	}

	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

// matchGlob is a simple glob matcher (simplified version).
func matchGlob(path, pattern string) bool {
	matched, _ := filepath.Match(pattern, path)
	return matched
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
