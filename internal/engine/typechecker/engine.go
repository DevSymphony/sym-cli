package typechecker

import (
	"context"
	"fmt"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	"github.com/DevSymphony/sym-cli/internal/adapter/tsc"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates TypeScript/JavaScript type correctness using tsc.
//
// Strategy:
// - Uses TypeScript Compiler (tsc) for type checking
// - Supports strict mode and various compiler options
// - Works with TypeScript (.ts, .tsx) and optionally JavaScript (.js, .jsx)
type Engine struct {
	tsc     *tsc.Adapter
	config  core.EngineConfig
}

// NewEngine creates a new type checker engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Initialize tsc adapter
	e.tsc = tsc.NewAdapter(config.ToolsDir, config.WorkDir)

	// Check tsc availability
	if err := e.tsc.CheckAvailability(ctx); err != nil {
		if config.Debug {
			fmt.Printf("TSC not found, attempting install...\n")
		}

		installConfig := adapter.InstallConfig{
			ToolsDir: config.ToolsDir,
		}

		if err := e.tsc.Install(ctx, installConfig); err != nil {
			return fmt.Errorf("failed to install TypeScript: %w", err)
		}
	}

	return nil
}

// Validate validates files against type checking rules.
func (e *Engine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	start := time.Now()

	files = core.FilterFiles(files, rule.When)
	if len(files) == 0 {
		return &core.ValidationResult{
			RuleID:   rule.ID,
			Passed:   true,
			Engine:   "typechecker",
			Duration: time.Since(start),
		}, nil
	}

	// Generate tsc config
	tscConfig, err := e.tsc.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tsc config: %w", err)
	}

	// Execute tsc
	output, err := e.tsc.Execute(ctx, tscConfig, files)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tsc: %w", err)
	}

	// Parse violations
	adapterViolations, err := e.tsc.ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tsc output: %w", err)
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
			RuleID:   av.RuleID,
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
		Engine:     "typechecker",
		Language:   e.detectLanguage(files),
	}, nil
}

// GetCapabilities returns engine capabilities.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name:                "typechecker",
		SupportedLanguages:  []string{"typescript", "javascript", "tsx", "jsx"},
		SupportedCategories: []string{"type_safety", "correctness", "custom"},
		SupportsAutofix:     false,
		RequiresCompilation: false,
		ExternalTools: []core.ToolRequirement{
			{
				Name:           "typescript",
				Version:        "^5.0.0",
				Optional:       false,
				InstallCommand: "npm install -g typescript",
			},
		},
	}
}

// Close cleans up resources.
func (e *Engine) Close() error {
	return nil
}

// detectLanguage detects the language from file extensions.
func (e *Engine) detectLanguage(files []string) string {
	if len(files) == 0 {
		return "typescript"
	}

	// Check first file
	file := files[0]
	if len(file) > 3 {
		ext := file[len(file)-3:]
		switch ext {
		case ".ts":
			return "typescript"
		case ".js":
			return "javascript"
		case "jsx":
			return "jsx"
		case "tsx":
			return "tsx"
		}
	}

	return "typescript"
}
