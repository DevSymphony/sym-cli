# Linter Package

Unified interface for static linting tools.

## File Structure

```
internal/linter/
├── linter.go        # Linter interface + Capabilities, InstallConfig, ToolOutput, Violation
├── converter.go     # Converter interface + LinterConfig, ConversionResult
├── registry.go      # Registry, Global(), RegisterTool(), GetLinter()
├── helpers.go       # DefaultToolsDir, WriteTempConfig, MapSeverity, FindTool
├── executor.go      # SubprocessExecutor
├── eslint/          # JavaScript/TypeScript
├── prettier/        # Code formatting
├── pylint/          # Python
├── tsc/             # TypeScript type checking
├── checkstyle/      # Java style
└── pmd/             # Java static analysis
```

## Usage

```go
import (
    "github.com/DevSymphony/sym-cli/internal/linter"
)

// Get linter by name
l, err := linter.Global().GetLinter("eslint")
if err != nil {
    return err
}

// Check availability and install if needed
if err := l.CheckAvailability(ctx); err != nil {
    if err := l.Install(ctx, linter.InstallConfig{}); err != nil {
        return err
    }
}

// Execute linter
output, err := l.Execute(ctx, config, files)
if err != nil {
    return err
}

// Parse output
violations, err := l.ParseOutput(output)
```

## Linter List

| Name | Languages | Categories | Config File |
|------|-----------|------------|-------------|
| `eslint` | JavaScript, TypeScript, JSX, TSX | pattern, length, style, ast | `.eslintrc.json` |
| `prettier` | JS, TS, JSON, CSS, HTML, Markdown | style | `.prettierrc.json` |
| `pylint` | Python | naming, style, docs, error_handling | `pylintrc` |
| `tsc` | TypeScript | typechecker | `tsconfig.json` |
| `checkstyle` | Java | naming, pattern, length, style | `checkstyle.xml` |
| `pmd` | Java | pattern, complexity, security, performance | `pmd-ruleset.xml` |

## Adding New Linter

### Step 1: Create Directory

```
internal/linter/<name>/
├── linter.go      # Main struct + interface implementation
├── register.go    # init() registration
├── converter.go   # LLM rule conversion
├── executor.go    # Tool execution
└── parser.go      # Output parsing
```

### Step 2: Implement Linter Interface

```go
package mylinter

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time interface check
var _ linter.Linter = (*Linter)(nil)

type Linter struct {
    ToolsDir string
    executor *linter.SubprocessExecutor
}

func New(toolsDir string) *Linter {
    if toolsDir == "" {
        toolsDir = linter.DefaultToolsDir()
    }
    return &Linter{
        ToolsDir: toolsDir,
        executor: linter.NewSubprocessExecutor(),
    }
}

func (l *Linter) Name() string {
    return "mylinter"
}

func (l *Linter) GetCapabilities() linter.Capabilities {
    return linter.Capabilities{
        Name:                "mylinter",
        SupportedLanguages:  []string{"ruby"},
        SupportedCategories: []string{"pattern", "style"},
        Version:             "^1.0.0",
    }
}

func (l *Linter) CheckAvailability(ctx context.Context) error {
    if path := linter.FindTool(l.localPath(), "mylinter"); path != "" {
        return nil
    }
    return fmt.Errorf("mylinter not found")
}

func (l *Linter) Install(ctx context.Context, cfg linter.InstallConfig) error {
    // Install logic (gem install, pip install, etc.)
}

func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
    // Execution logic
}

func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
    // Parse tool-specific output to violations
}
```

### Step 3: Implement Converter Interface

```go
// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

type Converter struct{}

func NewConverter() *Converter { return &Converter{} }

func (c *Converter) Name() string { return "mylinter" }

func (c *Converter) SupportedLanguages() []string {
    return []string{"ruby"}
}

func (c *Converter) GetLLMDescription() string {
    return "Ruby linter for style and quality"
}

func (c *Converter) GetRoutingHints() []string {
    return []string{"For Ruby code style → use mylinter"}
}

func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, provider llm.Provider) (*linter.ConversionResult, error) {
    // Use helper for parallel conversion
    results, successIDs, failedIDs := linter.ConvertRulesParallel(ctx, rules, c.convertSingle)

    // Build config from results
    config := buildConfig(results)

    return &linter.ConversionResult{
        Config:       config,
        SuccessRules: successIDs,
        FailedRules:  failedIDs,
    }, nil
}
```

### Step 4: Register in init()

```go
package mylinter

import (
    "github.com/DevSymphony/sym-cli/internal/linter"
)

func init() {
    _ = linter.Global().RegisterTool(
        New(linter.DefaultToolsDir()),
        NewConverter(),
        ".mylinter.yml",
    )
}
```

### Step 5: Add Import to Bootstrap

```go
// internal/bootstrap/linters.go
import (
    _ "github.com/DevSymphony/sym-cli/internal/linter/mylinter"
)
```

## Key Patterns

### Compile-Time Interface Checks

Every linter should include:
```go
var _ linter.Linter = (*Linter)(nil)
var _ linter.Converter = (*Converter)(nil)
```

### Helper Functions

Use the provided helpers from `base.go`:

```go
// Default tools directory (~/.sym/tools)
toolsDir := linter.DefaultToolsDir()

// Write temp config file
configPath, err := linter.WriteTempConfig(toolsDir, "mylinter", configBytes)

// Normalize severity
severity := linter.MapSeverity("warn")  // returns "warning"

// Find tool binary (local first, then PATH)
path := linter.FindTool(localPath, "mylinter")
```

### Parallel Rule Conversion

Use `ConvertRulesParallel` for efficient parallel processing:

```go
func (c *Converter) ConvertRules(ctx context.Context, rules []schema.UserRule, provider llm.Provider) (*linter.ConversionResult, error) {
    results, successIDs, failedIDs := linter.ConvertRulesParallel(ctx, rules,
        func(ctx context.Context, rule schema.UserRule) (*MyConfig, error) {
            return c.convertSingleRule(ctx, rule, provider)
        },
    )

    // Build final config from successful conversions
    // ...
}
```

## Error Handling

- Return clear error messages if tool not installed
- Track per-rule success/failure in ConversionResult
- Failed rules automatically fall back to llm-validator
