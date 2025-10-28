package style

import (
	"context"
	"fmt"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter/eslint"
	"github.com/DevSymphony/sym-cli/internal/adapter/prettier"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates code style rules (indent, quotes, semicolons, etc.).
//
// Strategy:
// - Validation: Use ESLint (indent, quotes, semi rules)
// - Autofix: Use Prettier (--write mode)
// - Prettier is opinionated, so we validate with ESLint to respect user config
type Engine struct {
	eslint   *eslint.Adapter
	prettier *prettier.Adapter
	config   core.EngineConfig
}

// NewEngine creates a new style engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Initialize adapters
	e.eslint = eslint.NewAdapter(config.ToolsDir, config.WorkDir)
	e.prettier = prettier.NewAdapter(config.ToolsDir, config.WorkDir)

	// Check ESLint availability
	if err := e.eslint.CheckAvailability(ctx); err != nil {
		if config.Debug {
			fmt.Printf("ESLint not found, attempting install...\n")
		}

		installConfig := struct {
			ToolsDir string
			Version  string
			Force    bool
		}{ToolsDir: config.ToolsDir}

		if err := e.eslint.Install(ctx, installConfig); err != nil {
			return fmt.Errorf("failed to install ESLint: %w", err)
		}
	}

	// Prettier is optional (can validate with ESLint only)
	if err := e.prettier.CheckAvailability(ctx); err != nil {
		if config.Debug {
			fmt.Printf("Prettier not found, autofix will be limited\n")
		}
	}

	return nil
}

// Validate validates files against a style rule.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	start := time.Now()

	files = e.filterFiles(files, rule.When)
	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:   rule.ID,
			Passed:   true,
			Engine:   "style",
			Duration: time.Since(start),
		}, nil
	}

	// Generate ESLint config for validation
	eslintConfig, err := e.eslint.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ESLint config: %w", err)
	}

	// Execute ESLint
	output, err := e.eslint.Execute(ctx, eslintConfig, files)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ESLint: %w", err)
	}

	// Parse violations
	adapterViolations, err := e.eslint.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ESLint output: %w", err)
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

		// Add autofix suggestion if Prettier available and remedy enabled
		if rule.Remedy != nil && rule.Remedy.Autofix {
			violations[i].Suggestion = &core.Suggestion{
				Desc: "Run prettier --write to auto-fix style issues",
			}
		}
	}

	return &core.ValidationResult{
		RuleID:     rule.ID,
		Passed:     len(violations) == 0,
		Violations: violations,
		Duration:   time.Since(start),
		Engine:     "style",
		Language:   "javascript",
	}, nil
}

// GetCapabilities returns engine capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name:                "style",
		SupportedLanguages:  []string{"javascript", "typescript", "jsx", "tsx"},
		SupportedCategories: []string{"style", "formatting"},
		SupportsAutofix:     true,
		RequiresCompilation: false,
		ExternalTools: []core.ToolRequirement{
			{
				Name:           "eslint",
				Version:        "^8.0.0",
				Optional:       false,
				InstallCommand: "npm install -g eslint",
			},
			{
				Name:           "prettier",
				Version:        "^3.0.0",
				Optional:       true,
				InstallCommand: "npm install -g prettier",
			},
		},
	}
}

// Close cleans up resources.
func (e *Engine) Close() error {
	return nil
}

func (e *Engine) filterFiles(files []string, selector *core.Selector) []string {
	if selector == nil {
		return files
	}

	var filtered []string
	for _, file := range files {
		if len(file) > 3 {
			ext := file[len(file)-3:]
			if ext == ".js" || ext == ".ts" || ext == "jsx" || ext == "tsx" {
				filtered = append(filtered, file)
			}
		}
	}

	return filtered
}
