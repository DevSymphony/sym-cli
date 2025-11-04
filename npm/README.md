# Symphony MCP Server

LLM-friendly convention linter for AI coding tools.

## Installation

### One-line MCP Setup

```bash
claude mcp add symphony npx @dev-symphony/sym@latest mcp
```

### Direct Installation

```bash
npm install -g @dev-symphony/sym
```

## Usage

### MCP Configuration

Add to your MCP config file:

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

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
