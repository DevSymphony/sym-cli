# Linter Package

Unified interface for static linting tools.

## File Structure

```
internal/linter/
├── linter.go        # Linter interface (execution)
├── converter.go     # Converter interface (rule conversion)
├── registry.go      # Global registry, RegisterTool(), GetLinter()
├── helpers.go       # CleanJSONResponse, DefaultToolsDir, WriteTempConfig
├── subprocess.go    # SubprocessExecutor
├── eslint/          # JavaScript/TypeScript
├── prettier/        # Code formatting
├── pylint/          # Python
├── tsc/             # TypeScript type checking
├── checkstyle/      # Java style
└── pmd/             # Java static analysis
```

## Usage

```go
import "github.com/DevSymphony/sym-cli/internal/linter"

// 1. Get linter by name
l, err := linter.Global().GetLinter("eslint")
if err != nil {
    return err
}

// 2. Check availability and install if needed
if err := l.CheckAvailability(ctx); err != nil {
    if err := l.Install(ctx, linter.InstallConfig{}); err != nil {
        return err
    }
}

// 3. Execute linter
output, err := l.Execute(ctx, config, files)
if err != nil {
    return err
}

// 4. Parse output to violations
violations, err := l.ParseOutput(output)
```

### Getting Converter

```go
converter, ok := linter.Global().GetConverter("eslint")
if !ok {
    return fmt.Errorf("converter not found")
}

// Convert single rule (called by main converter in parallel)
result, err := converter.ConvertSingleRule(ctx, rule, llmProvider)

// Build config from results
config, err := converter.BuildConfig(results)
```

## Linter List

| Name | Languages | Config File |
|------|-----------|-------------|
| `eslint` | JavaScript, TypeScript, JSX, TSX | `.eslintrc.json` |
| `prettier` | JS, TS, JSON, CSS, HTML, Markdown | `.prettierrc` |
| `pylint` | Python | `.pylintrc` |
| `tsc` | TypeScript | `tsconfig.json` |
| `checkstyle` | Java | `checkstyle.xml` |
| `pmd` | Java | `pmd.xml` |

## Adding New Linter

### Step 1: Create Directory

```
internal/linter/<name>/
├── linter.go      # Linter implementation
├── converter.go   # Converter implementation
├── executor.go    # Tool execution logic
├── parser.go      # Output parsing
└── register.go    # init() registration
```

### Step 2: Implement Linter Interface

```go
package mylinter

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/linter"
)

// Compile-time check
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

func (l *Linter) Name() string { return "mylinter" }

func (l *Linter) GetCapabilities() linter.Capabilities {
    return linter.Capabilities{
        Name:                "mylinter",
        SupportedLanguages:  []string{"ruby"},
        SupportedCategories: []string{"pattern", "style"},
        Version:             "1.0.0",
    }
}

func (l *Linter) CheckAvailability(ctx context.Context) error {
    // Check if tool binary exists
}

func (l *Linter) Install(ctx context.Context, cfg linter.InstallConfig) error {
    // Install tool (gem install, pip install, npm install, etc.)
}

func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
    // Run tool and return output
}

func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
    // Parse tool-specific output to violations
}
```

### Step 3: Implement Converter Interface

```go
package mylinter

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/linter"
    "github.com/DevSymphony/sym-cli/internal/llm"
    "github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time check
var _ linter.Converter = (*Converter)(nil)

type Converter struct{}

func NewConverter() *Converter { return &Converter{} }

func (c *Converter) Name() string { return "mylinter" }

func (c *Converter) SupportedLanguages() []string {
    return []string{"ruby"}
}

func (c *Converter) GetLLMDescription() string {
    return `Ruby code quality (style, naming, complexity)
  - CAN: Naming conventions, line length, cyclomatic complexity
  - CANNOT: Business logic, runtime behavior`
}

func (c *Converter) GetRoutingHints() []string {
    return []string{
        "For Ruby code style → use mylinter",
        "For Ruby naming conventions → use mylinter",
    }
}

// ConvertSingleRule converts ONE user rule to linter-specific data.
// Concurrency is handled by main converter, not here.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
    // Call LLM to convert rule
    config, err := c.callLLM(ctx, rule, provider)
    if err != nil {
        return nil, err
    }

    // Return nil, nil if rule cannot be enforced by this linter
    if config == nil {
        return nil, nil
    }

    return &linter.SingleRuleResult{
        RuleID: rule.ID,
        Data:   config,  // Linter-specific data
    }, nil
}

// BuildConfig assembles final config from all successful conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
    if len(results) == 0 {
        return nil, nil
    }

    // Build config from results
    config := buildMyLinterConfig(results)

    content, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return nil, err
    }

    return &linter.LinterConfig{
        Filename: ".mylinter.yml",
        Content:  content,
        Format:   "yaml",
    }, nil
}
```

### Step 4: Register in init()

```go
package mylinter

import "github.com/DevSymphony/sym-cli/internal/linter"

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
// cmd/sym/bootstrap.go
import (
    _ "github.com/DevSymphony/sym-cli/internal/linter/mylinter"
)
```

## Key Rules

- Add compile-time checks for both interfaces:
  ```go
  var _ linter.Linter = (*Linter)(nil)
  var _ linter.Converter = (*Converter)(nil)
  ```
- `ConvertSingleRule()` handles ONE rule only - concurrency is managed by main converter
- Return `(nil, nil)` from `ConvertSingleRule()` if rule cannot be enforced (falls back to llm-validator)
- Use `linter.CleanJSONResponse()` to strip markdown fences from LLM responses
- Return clear error messages if tool not installed
- Refer to existing linters (eslint, pylint) for patterns
