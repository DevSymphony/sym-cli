package pattern

import (
	"context"
	"fmt"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates pattern rules (naming, forbidden patterns, imports).
//
// For JavaScript/TypeScript:
// - Uses ESLint id-match for identifier patterns
// - Uses ESLint no-restricted-syntax for content patterns
// - Uses ESLint no-restricted-imports for import patterns
type Engine struct {
	eslint *eslint.Adapter
	config core.EngineConfig
}

// NewEngine creates a new pattern engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Initialize ESLint adapter
	e.eslint = eslint.NewAdapter(config.ToolsDir, config.WorkDir)

	// Check if ESLint is available
	if err := e.eslint.CheckAvailability(ctx); err != nil {
		// Try to install
		if config.Debug {
			fmt.Printf("ESLint not found, attempting install...\n")
		}

		installConfig := struct {
			ToolsDir string
			Version  string
			Force    bool
		}{
			ToolsDir: config.ToolsDir,
			Version:  "",
			Force:    false,
		}

		// Convert to adapter.InstallConfig
		// (Note: we'd need to import adapter package, but for now let's inline)
		if err := e.eslint.Install(ctx, installConfig); err != nil {
			return fmt.Errorf("failed to install ESLint: %w", err)
		}
	}

	return nil
}

// Validate validates files against a pattern rule.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	start := time.Now()

	// Filter files by selector
	files = e.filterFiles(files, rule.When)

	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:   rule.ID,
			Passed:   true,
			Engine:   "pattern",
			Duration: time.Since(start),
		}, nil
	}

	// Generate ESLint config
	config, err := e.eslint.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// Execute ESLint
	output, err := e.eslint.Execute(ctx, config, files)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ESLint: %w", err)
	}

	// Parse output
	adapterViolations, err := e.eslint.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// Convert to core violations
	violations := make([]core.Violation, len(adapterViolations))
	for i, av := range adapterViolations {
		violations[i] = core.Violation{
			File:     av.File,
			Line:     av.Line,
			Column:   av.Column,
			Message:  av.Message,
			Severity: av.Severity,
			RuleID:   rule.ID,
			Category: rule.Category,
		}

		// Use custom message if provided
		if rule.Message != "" {
			violations[i].Message = rule.Message
		}
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Duration:   time.Since(start),
		Engine:     "pattern",
		Language:   e.detectLanguage(files),
	}, nil
}

// GetCapabilities returns engine capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name:                "pattern",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"naming", "security", "custom"},
		SupportsAutofix:     false,
		RequiresCompilation: false,
		ExternalTools: []core.ToolRequirement{
			{
				Name:           "eslint",
				Version:        "^8.0.0",
				Optional:       false,
				InstallCommand: "npm install -g eslint",
			},
		},
	}
}

// Close cleans up resources.
func (e *Engine) Close() error {
	return nil
}

// filterFiles filters files based on selector.
func (e *Engine) filterFiles(files []string, selector *core.Selector) []string {
	if selector == nil {
		return files
	}

	// Simple filter by extension
	// TODO: Implement proper glob matching
	var filtered []string
	for _, file := range files {
		if e.matchesLanguage(file, selector.Languages) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// matchesLanguage checks if file matches language selector.
func (e *Engine) matchesLanguage(file string, languages []string) bool {
	if len(languages) == 0 {
		return true // No filter
	}

	// Check by extension
	for _, lang := range languages {
		switch lang {
		case "javascript", "js":
			if len(file) > 3 && file[len(file)-3:] == ".js" {
				return true
			}
		case "typescript", "ts":
			if len(file) > 3 && file[len(file)-3:] == ".ts" {
				return true
			}
		case "jsx":
			if len(file) > 4 && file[len(file)-4:] == ".jsx" {
				return true
			}
		case "tsx":
			if len(file) > 4 && file[len(file)-4:] == ".tsx" {
				return true
			}
		}
	}

	return false
}

// detectLanguage detects the language from file extensions.
func (e *Engine) detectLanguage(files []string) string {
	if len(files) == 0 {
		return "javascript"
	}

	// Check first file
	file := files[0]
	if len(file) > 3 {
		ext := file[len(file)-3:]
		switch ext {
		case ".ts":
			return "typescript"
		case "jsx":
			return "jsx"
		case "tsx":
			return "tsx"
		}
	}

	return "javascript"
}
