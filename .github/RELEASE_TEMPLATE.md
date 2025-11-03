# 🎵 Symphony CLI v{VERSION}

> LLM-friendly convention linter for AI coding assistants

## 🚀 Quick Start

### For Claude Code Users

**One-line installation:**
```bash
claude mcp add symphony npx @devsymphony/sym@latest mcp
```

Restart Claude Desktop and ask: "What are the project conventions?"

### For Other Tools (Cursor, Continue.dev, etc.)

**Manual MCP Configuration:**

Add to your config file:
```json
{
  "mcpServers": {
    "symphony": {
      "command": "npx",
      "args": ["-y", "@devsymphony/sym@latest", "mcp"]
    }
  }
}
```

**Config locations:**
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

### Direct Installation

**npm:**
```bash
npm install -g @devsymphony/sym
```

**Binary:** Download from assets below

---

## ✨ What's New

<!-- List new features, improvements, and changes here -->

### New Features
-

### Improvements
-

### Bug Fixes
-

### Documentation
-

---

## 📦 Installation Options

### 1. MCP Server (Recommended for AI Tools)

```bash
# Claude Code
claude mcp add symphony npx @devsymphony/sym@latest mcp

# Manual configuration
# See "Quick Start" section above
```

### 2. npm Global Install

```bash
npm install -g @devsymphony/sym

# Verify installation
sym --version
```

### 3. Binary Download

Download platform-specific binaries from the assets below:

- **macOS Apple Silicon**: `sym-darwin-arm64.tar.gz`
- **macOS Intel**: `sym-darwin-amd64.tar.gz`
- **Linux x64**: `sym-linux-amd64.tar.gz`
- **Linux ARM64**: `sym-linux-arm64.tar.gz`
- **Windows x64**: `sym-windows-amd64.exe.tar.gz`

**GPG Signature Verification:**
```bash
# Download binary and signature
wget https://github.com/DevSymphony/sym-cli/releases/download/v{VERSION}/sym-linux-amd64
wget https://github.com/DevSymphony/sym-cli/releases/download/v{VERSION}/sym-linux-amd64.asc

# Verify signature (if GPG signing is enabled)
gpg --verify sym-linux-amd64.asc sym-linux-amd64
```

---

## 🎯 Usage Examples

### MCP Server Mode

```bash
# stdio mode (for AI tools)
sym mcp

# HTTP mode (for testing)
sym mcp --port 4000

# Custom policy file
sym mcp --config .sym/custom-policy.json
```

### CLI Mode

```bash
# Initialize project
sym init

# Validate code
sym validate ./src

# Query conventions
sym export --context "authentication code"

# Web dashboard
sym dashboard
```

---

## 🔧 MCP Tools Available

Once configured, AI tools can use these MCP tools:

### 1. `query_conventions`
Query project conventions by category, language, or files.

**Example:**
```json
{
  "method": "query_conventions",
  "params": {
    "category": "naming",
    "languages": ["go", "typescript"]
  }
}
```

### 2. `validate_code`
Validate code against project conventions.

**Example:**
```json
{
  "method": "validate_code",
  "params": {
    "files": ["./src/main.go"]
  }
}
```

---

## 🔍 Supported Platforms

- ✅ macOS (Apple Silicon M1/M2/M3, Intel)
- ✅ Linux (x64, ARM64)
- ✅ Windows (x64)
- ✅ Node.js >= 16.0.0 (for npm installation)

---

## 📚 Documentation

- **Full Documentation**: [https://github.com/DevSymphony/sym-cli](https://github.com/DevSymphony/sym-cli)
- **MCP Setup Guide**: [npm/README.md](https://github.com/DevSymphony/sym-cli/blob/main/npm/README.md)
- **Policy Schema**: [.claude/schema.md](https://github.com/DevSymphony/sym-cli/blob/main/.claude/schema.md)
- **Examples**: [examples/](https://github.com/DevSymphony/sym-cli/tree/main/examples)

---

## 🐛 Known Issues

<!-- List any known issues or limitations -->

None reported for this release.

---

## 🔄 Upgrade Guide

### From v0.1.x to v{VERSION}

```bash
# npm users
npm update -g @devsymphony/sym

# Binary users
# Download and replace binary from assets below

# MCP users
# No action needed - npx automatically uses latest version
```

---

## 📊 Checksums

<!-- Auto-generated during release -->

```
SHA256 checksums will be listed here
```

---

## 🙏 Acknowledgments

Thanks to all contributors who made this release possible!

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/DevSymphony/sym-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/DevSymphony/sym-cli/discussions)
- **Email**: support@devsymphony.com

---

**Full Changelog**: https://github.com/DevSymphony/sym-cli/compare/v{PREVIOUS_VERSION}...v{VERSION}
