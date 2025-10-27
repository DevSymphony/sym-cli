package length

import (
	"context"
	"fmt"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates length constraint rules (line, file, function, params).
//
// For JavaScript/TypeScript:
// - Uses ESLint max-len for line length
// - Uses ESLint max-lines for file length
// - Uses ESLint max-lines-per-function for function length
// - Uses ESLint max-params for parameter count
type Engine struct {
	eslint *eslint.Adapter
	config core.EngineConfig
}

// NewEngine creates a new length engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Initialize ESLint adapter
	e.eslint = eslint.NewAdapter(config.ToolsDir, config.WorkDir)

	// Check availability (same as pattern engine)
	if err := e.eslint.CheckAvailability(ctx); err != nil {
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

		if err := e.eslint.Install(ctx, installConfig); err != nil {
			return fmt.Errorf("failed to install ESLint: %w", err)
		}
	}

	return nil
}

// Validate validates files against a length rule.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	start := time.Now()

	// Filter files
	files = e.filterFiles(files, rule.When)

	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:   rule.ID,
			Passed:   true,
			Engine:   "length",
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

		if rule.Message != "" {
			violations[i].Message = rule.Message
		}
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Duration:   time.Since(start),
		Engine:     "length",
		Language:   "javascript",
	}, nil
}

// GetCapabilities returns engine capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name:                "length",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"formatting", "style"},
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

	// Simple extension-based filter
	var filtered []string
	for _, file := range files {
		// Accept .js, .ts, .jsx, .tsx
		if len(file) > 3 {
			ext := file[len(file)-3:]
			if ext == ".js" || ext == ".ts" || ext == "jsx" || ext == "tsx" {
				filtered = append(filtered, file)
			}
		}
	}

	return filtered
}
