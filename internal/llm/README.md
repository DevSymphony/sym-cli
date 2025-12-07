# LLM Package

Unified interface for LLM providers.

## File Structure

```
internal/llm/
├── llm.go           # Provider interface, Config, ResponseFormat
├── registry.go      # RegisterProvider, New, ListProviders
├── config.go        # LoadConfig, Parse
├── parser.go        # Response parsing
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

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LLM_PROVIDER` | Provider name (claudecode, geminicli, openaiapi) | Yes |
| `LLM_MODEL` | Model name | No (uses default) |
| `OPENAI_API_KEY` | OpenAI API key | openaiapi only |

Config file: `.sym/.env`

```bash
LLM_PROVIDER=claudecode
LLM_MODEL=claude-sonnet-4-20250514
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
| `claudecode` | CLI | claude-haiku-4-5-20251001 | `npm i -g @anthropic-ai/claude-cli` |
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

### Step 2: Implement Provider Interface

```go
package myprovider

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/llm"
)

type Provider struct {
    model string
}

func (p *Provider) Name() string {
    return "myprovider"
}

func (p *Provider) Execute(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
    // Execute prompt
    response := callLLM(prompt)

    // Parse response (required)
    return llm.Parse(response, format)
}
```

### Step 3: Register in init()

```go
func init() {
    llm.RegisterProvider("myprovider", newProvider, llm.ProviderInfo{
        Name:         "myprovider",
        DisplayName:  "My Provider",
        DefaultModel: "default-model-v1",
        Available:    checkAvailability(), // Check CLI exists or API key set
        Path:         cliPath,             // CLI path (empty for API)
    })
}

func newProvider(cfg llm.Config) (llm.Provider, error) {
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

- Use `llm.Parse(response, format)` for response parsing
- Return clear error message if CLI not installed or API key missing
- Check availability in init() to set ProviderInfo.Available
- Refer to existing providers (claudecode, openaiapi) for patterns
