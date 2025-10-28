package core

import (
	"context"
	"time"
)

// Engine is the interface that all validation engines must implement.
// Engines validate code against specific rule types (pattern, length, style, etc.).
//
// Design Philosophy:
// - Engines are language-agnostic at the interface level
// - Language-specific implementations are provided via LanguageProvider
// - External tools (ESLint, Prettier, etc.) are wrapped by adapters
type Engine interface {
	// Init initializes the engine with configuration.
	// Called once before validation begins.
	Init(ctx context.Context, config EngineConfig) error

	// Validate validates files against a rule.
	// Returns violations found, or nil if validation passed.
	//
	// The engine should:
	// 1. Parse rule.Check configuration
	// 2. Delegate to appropriate adapter (e.g., ESLint for JavaScript)
	// 3. Collect and return violations
	Validate(ctx context.Context, rule Rule, files []string) (*ValidationResult, error)

	// GetCapabilities returns engine capabilities (languages, features, etc.).
	GetCapabilities() EngineCapabilities

	// Close cleans up resources (close connections, temp files, etc.).
	Close() error
}

// EngineConfig holds engine initialization settings.
type EngineConfig struct {
	// WorkDir is the working directory (usually project root).
	WorkDir string

	// ToolsDir is where external tools are installed.
	// Default: ~/.symphony/tools
	ToolsDir string

	// CacheDir is for caching validation results.
	// Default: ~/.symphony/cache
	CacheDir string

	// Timeout is the max time for a single validation.
	// Default: 2 minutes
	Timeout time.Duration

	// Parallelism is max concurrent validations.
	// 0 = runtime.NumCPU()
	Parallelism int

	// Debug enables verbose logging.
	Debug bool

	// Extra holds engine-specific config.
	Extra map[string]interface{}
}

// EngineCapabilities describes what an engine supports.
type EngineCapabilities struct {
	// Name is the engine identifier (e.g., "pattern", "style").
	Name string

	// SupportedLanguages lists supported languages.
	// Empty = language-agnostic (e.g., commit engine).
	// Example: ["javascript", "typescript", "jsx", "tsx"]
	SupportedLanguages []string

	// SupportedCategories lists rule categories this engine handles.
	// Example: ["naming", "security"] for pattern engine.
	SupportedCategories []string

	// SupportsAutofix indicates if engine can auto-fix violations.
	SupportsAutofix bool

	// RequiresCompilation indicates if compiled artifacts needed.
	// Example: ArchUnit requires .class files (Java).
	RequiresCompilation bool

	// ExternalTools lists required external tools.
	// Example: ["eslint@^8.0.0", "prettier@^3.0.0"]
	ExternalTools []ToolRequirement
}

// ToolRequirement specifies an external tool dependency.
type ToolRequirement struct {
	// Name is the tool name (e.g., "eslint").
	Name string

	// Version is the required version (e.g., "^8.0.0").
	// Empty = any version.
	Version string

	// Optional indicates if tool is optional.
	// If true, engine falls back to internal implementation.
	Optional bool

	// InstallCommand is the command to install the tool.
	// Example: "npm install -g eslint@^8.0.0"
	InstallCommand string
}
