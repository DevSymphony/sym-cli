# LLM Package

Unified interface for LLM providers.

## File Structure

```
internal/llm/
├── llm.go           # Provider, RawProvider interface, Config, ResponseFormat
├── registry.go      # RegisterProvider, New, ListProviders
├── wrapper.go       # parsedProvider (auto parsing wrapper)
├── config.go        # LoadConfig
├── parser.go        # Response parsing (private)
├── DESIGN.md        # Architecture documentation
├── claudecode/      # Claude Code CLI provider
├── geminicli/       # Gemini CLI provider
└── openaiapi/       # OpenAI API provider
```

## Usage

```go
import "github.com/DevSymphony/sym-cli/internal/llm"

// 1. Load config
cfg := llm.LoadConfig()

// 2. Create provider
provider, err := llm.New(cfg)
if err != nil {
    return err // CLI not installed or API key missing
}

// 3. Execute prompt
response, err := provider.Execute(ctx, prompt, llm.JSON)
```

### Configuration

Config file: `.sym/config.json`

```json
{
  "llm": {
    "provider": "claudecode",
    "model": "sonnet"
  }
}
```

For OpenAI API, also add API key to `.sym/.env`:

```bash
OPENAI_API_KEY=sk-...
```

### Response Format

| Format | Description |
|--------|-------------|
| `llm.Text` | Return raw text |
| `llm.JSON` | Extract JSON from LLM response |
| `llm.XML` | Extract XML from LLM response |

`llm.JSON` and `llm.XML` automatically extract structured data when LLM returns JSON/XML with preamble text.

## Provider List

| Name | Type | Default Model | Installation |
|------|------|---------------|--------------|
| `claudecode` | CLI | sonnet | `npm i -g @anthropic-ai/claude-cli` |
| `geminicli` | CLI | gemini-2.5-flash | `npm i -g @google/gemini-cli` |
| `openaiapi` | API | gpt-4o-mini | Set `OPENAI_API_KEY` env var |

### Check Provider Status

```go
providers := llm.ListProviders()
for _, p := range providers {
    fmt.Printf("%s: available=%v\n", p.Name, p.Available)
}
```

## Adding New Provider

### Step 1: Create Directory

```
internal/llm/<provider>/
└── provider.go
```

### Step 2: Implement RawProvider Interface

```go
package myprovider

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/llm"
)

type Provider struct {
    model string
}

// Compile-time check: Provider must implement RawProvider interface
var _ llm.RawProvider = (*Provider)(nil)

func (p *Provider) Name() string {
    return "myprovider"
}

func (p *Provider) ExecuteRaw(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
    // Execute prompt and return raw response
    response := callLLM(prompt)
    return response, nil  // No manual parsing needed
}

func (p *Provider) Close() error {
    return nil
}
```

> **Note**: The `var _ llm.RawProvider = (*Provider)(nil)` line is a compile-time check that ensures `Provider` implements the `RawProvider` interface. If any method is missing or has the wrong signature, the code will fail to compile.

### Step 3: Register in init()

```go
func init() {
    llm.RegisterProvider("myprovider", newProvider, llm.ProviderInfo{
        Name:         "myprovider",
        DisplayName:  "My Provider",
        DefaultModel: "default-model-v1",
        Available:    checkAvailability(), // Check CLI exists or API key set
        Path:         cliPath,             // CLI path (empty for API)
        Models: []llm.ModelInfo{
            {ID: "model-v1", DisplayName: "Model V1", Description: "Standard model", Recommended: true},
            {ID: "model-v2", DisplayName: "Model V2", Description: "Advanced model", Recommended: false},
        },
        APIKey: llm.APIKeyConfig{
            Required:   true,                    // Set false for CLI-based providers
            EnvVarName: "MY_PROVIDER_API_KEY",   // Environment variable name
            Prefix:     "mp-",                   // Expected prefix for validation (optional)
        },
    })
}

func newProvider(cfg llm.Config) (llm.RawProvider, error) {
    // Return error if CLI not installed or API key missing
    if !isAvailable() {
        return nil, fmt.Errorf("provider not available")
    }

    model := cfg.Model
    if model == "" {
        model = "default-model-v1"
    }

    return &Provider{model: model}, nil
}
```

### Step 4: Add Import to Bootstrap

```go
// internal/bootstrap/providers.go
import (
    _ "github.com/DevSymphony/sym-cli/internal/llm/myprovider"
)
```

### Key Rules

- Add compile-time check: `var _ llm.RawProvider = (*Provider)(nil)`
- Implement `RawProvider.ExecuteRaw()` - parsing is handled automatically by wrapper
- Return clear error message if CLI not installed or API key missing
- Check availability in init() to set ProviderInfo.Available
- Define `Models` with at least one model marked as `Recommended: true`
- Set `APIKey.Required: false` for CLI-based providers (they handle auth internally)
- Refer to existing providers (claudecode, openaiapi) for patterns
