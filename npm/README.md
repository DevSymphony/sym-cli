# Symphony MCP Server

LLM-friendly convention linter for AI coding tools.

## Quick Start

```bash
# 1. Install
npm install -g @dev-symphony/sym

# 2. Initialize (GitHub OAuth login + MCP auto-setup)
sym login
sym init
```

> **Note**: `OPENAI_API_KEY` environment variable is required for LLM-based convention conversion.

## MCP Configuration

MCP is auto-configured during `sym init`.

For manual setup:

```json
{
  "mcpServers": {
    "symphony": {
      "command": "npx",
      "args": ["-y", "@dev-symphony/sym@latest", "mcp"]
    }
  }
}
```

### Available Tools

**query_conventions**
- Query project conventions by category, files, or languages
- All parameters are optional

**validate_code**
- Validate code against defined conventions
- Parameters: files (required)

## Policy File

Create `.sym/user-policy.json` in your project root:

```json
{
  "version": "1.0.0",
  "rules": [
    {
      "say": "Functions should be documented",
      "category": "documentation"
    },
    {
      "say": "Lines should be less than 100 characters",
      "category": "formatting",
      "params": { "max": 100 }
    }
  ]
}
```

## Requirements

- Node.js >= 16.0.0
- Policy file: `.sym/user-policy.json`

## Supported Platforms

- macOS (Intel, Apple Silicon)
- Linux (x64, ARM64)
- Windows (x64)

## License

MIT
