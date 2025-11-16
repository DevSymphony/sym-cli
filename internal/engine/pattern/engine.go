package pattern

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/DevSymphony/sym-cli/internal/adapter"
	adapterRegistry "github.com/DevSymphony/sym-cli/internal/adapter/registry"
	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// Engine validates pattern rules (naming, forbidden patterns, imports).
//
// Supports multiple languages through adapter registry:
// - JavaScript/TypeScript: ESLint (id-match, no-restricted-syntax, no-restricted-imports)
// - Java: Checkstyle (naming conventions, forbidden imports)
type Engine struct {
	adapterRegistry *adapterRegistry.Registry
	config          core.EngineConfig
}

// NewEngine creates a new pattern engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Init initializes the engine with adapter registry.
func (e *Engine) Init(ctx context.Context, config core.EngineConfig) error {
	e.config = config

	// Use provided adapter registry or create default
	if config.AdapterRegistry != nil {
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

	// Check initialization
	if e.adapterRegistry == nil {
		return nil, fmt.Errorf("pattern engine not initialized")
	}

	// Detect language
	language := e.detectLanguage(rule, files)

	// Get appropriate adapter for language
	adp, err := e.adapterRegistry.GetAdapter(language, "pattern")
	if err != nil {
		return nil, fmt.Errorf("no adapter found for language %s: %w", language, err)
	}

	// Type assert to adapter.Adapter
	patternAdapter, ok := adp.(adapter.Adapter)
	if !ok {
		return nil, fmt.Errorf("invalid adapter type for language %s", language)
	}

	// Check if adapter is available, install if needed
	if err := patternAdapter.CheckAvailability(ctx); err != nil {
		if installErr := patternAdapter.Install(ctx, adapter.InstallConfig{}); installErr != nil {
			return nil, fmt.Errorf("adapter not available and installation failed: %w", installErr)
		}
	}

	// Generate config
	config, err := patternAdapter.GenerateConfig(&rule)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// Execute adapter
	output, err := patternAdapter.Execute(ctx, config, files)
	if err != nil && output == nil {
		return nil, fmt.Errorf("failed to execute adapter: %w", err)
	}

	// Parse output
	adapterViolations, err := patternAdapter.ParseOutput(output)
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
		Language:   language,
	}, nil
}

// GetCapabilities returns engine capabilities.
// Supported languages are determined dynamically based on registered adapters.
func (e *Engine) GetCapabilities() core.EngineCapabilities {
	caps := core.EngineCapabilities{
		Name:                "pattern",
		SupportedCategories: []string{"naming", "security", "custom"},
		SupportsAutofix:     false,
		RequiresCompilation: false,
	}

	// If registry is available, get languages dynamically
	if e.adapterRegistry != nil {
		caps.SupportedLanguages = e.adapterRegistry.GetSupportedLanguages("pattern")
	} else {
		// Fallback to default JS/TS
		caps.SupportedLanguages = []string{"javascript", "typescript", "jsx", "tsx"}
	}

	return caps
}

// Close cleans up resources.
func (e *Engine) Close() error {
	return nil
}

// filterFiles filters files based on selector using proper glob matching.
func (e *Engine) filterFiles(files []string, selector *core.Selector) []string {
	return core.FilterFiles(files, selector)
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
