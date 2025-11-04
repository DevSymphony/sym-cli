# Symphony MCP Server

AI coding tools용 컨벤션 린터 MCP 서버

[![npm version](https://img.shields.io/npm/v/@dev-symphony/sym.svg)](https://www.npmjs.com/package/@dev-symphony/sym)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 Quick Start (Claude Code)

```bash
claude mcp add symphony npx @dev-symphony/sym@latest mcp
```

That's it! 이제 Claude에게 "프로젝트 컨벤션이 뭐야?"라고 물어보세요.

## 📦 Direct Installation

```bash
npm install -g @dev-symphony/sym
```

## 🔧 Manual MCP Configuration

Claude Desktop / Cursor / Continue.dev 등에서 사용:

### Config File Locations

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

### Configuration

```json
{
  "mcpServers": {
    "symphony": {
      "command": "npx",
      "args": ["-y", "@dev-symphony/sym@latest", "mcp"],
      "env": {
        "SYM_POLICY_PATH": "${workspaceFolder}/.sym/user-policy.json"
      }
    }
  }
}
```

## 🎯 Available MCP Tools

### 1. `query_conventions`

프로젝트 컨벤션 조회

**Parameters**:
- `category` (optional): "naming", "formatting", "security", "error_handling", "testing", "documentation" 등
- `files` (optional): 파일 경로 배열
- `languages` (optional): 언어 필터 (예: ["go", "typescript"])

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "query_conventions",
  "params": {
    "category": "naming",
    "languages": ["go", "typescript"]
  },
  "id": 1
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "conventions": [
      {
        "id": "NAMING-CLASS-PASCAL",
        "category": "naming",
        "description": "Class names should use PascalCase",
        "message": "클래스명은 PascalCase여야 합니다",
        "severity": "error"
      }
    ],
    "total": 1
  },
  "id": 1
}
```

### 2. `validate_code`

코드 검증

**Parameters**:
- `files`: 검증할 파일 경로 배열
- `role` (optional): RBAC 역할

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "validate_code",
  "params": {
    "files": ["./src/main.go"]
  },
  "id": 1
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "valid": false,
    "violations": [
      {
        "rule_id": "FMT-LINE-100",
        "message": "Line exceeds 100 characters",
        "severity": "warning",
        "file": "./src/main.go",
        "line": 42,
        "column": 101
      }
    ],
    "total": 1
  },
  "id": 1
}
```

## 🧪 Test the MCP Server

### stdio Mode (Default)

```bash
# Start MCP server in stdio mode
npx @dev-symphony/sym@latest mcp

# Test with echo
echo '{"jsonrpc":"2.0","method":"query_conventions","params":{},"id":1}' | \
  npx @dev-symphony/sym@latest mcp
```

### HTTP Mode (For Testing)

```bash
# Start HTTP server on port 4000
npx @dev-symphony/sym@latest mcp --port 4000

# Health check
curl http://localhost:4000/health

# Test query_conventions
curl -X POST http://localhost:4000 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"query_conventions","params":{"category":"naming"},"id":1}'
```

## 📋 Requirements

- **Node.js**: >= 16.0.0
- **Policy File**: `.sym/user-policy.json` in your project root

## 🗂️ Policy File Example

Create `.sym/user-policy.json` in your project:

```json
{
  "version": "1.0.0",
  "defaults": {
    "languages": ["go", "typescript"],
    "severity": "warning",
    "autofix": true
  },
  "rules": [
    {
      "say": "Functions should be documented",
      "category": "documentation"
    },
    {
      "say": "Lines should be less than 100 characters",
      "category": "formatting",
      "params": { "max": 100 }
    },
    {
      "say": "No hardcoded secrets",
      "category": "security",
      "severity": "error"
    }
  ]
}
```

## 🔍 Supported Platforms

- ✅ macOS (Apple Silicon, Intel)
- ✅ Linux (x64, ARM64)
- ✅ Windows (x64)

## 📚 Documentation

- **Full Documentation**: [https://github.com/DevSymphony/sym-cli](https://github.com/DevSymphony/sym-cli)
- **Schema Guide**: [Policy Schema Documentation](https://github.com/DevSymphony/sym-cli/blob/main/.claude/schema.md)
- **Examples**: [https://github.com/DevSymphony/sym-cli/tree/main/examples](https://github.com/DevSymphony/sym-cli/tree/main/examples)

## 🐛 Troubleshooting

### MCP 서버가 시작되지 않음

```bash
# Clear npm cache and reinstall
npm cache clean --force
npm install -g @dev-symphony/sym

# Verify installation
sym --version
```

### 정책 파일을 찾을 수 없음

Create `.sym/user-policy.json` in your project root:

```json
{
  "version": "1.0.0",
  "rules": [
    { "say": "Functions should be documented" }
  ]
}
```

### Permission denied (Unix/Linux/macOS)

```bash
# Make binary executable
chmod +x $(which sym)

# Or reinstall with proper permissions
sudo npm install -g @dev-symphony/sym
```

### Binary download failed

The package automatically downloads platform-specific binaries from GitHub Releases. If download fails:

1. Check your internet connection
2. Verify the release exists: [https://github.com/DevSymphony/sym-cli/releases](https://github.com/DevSymphony/sym-cli/releases)
3. If behind a proxy, set `HTTPS_PROXY` environment variable

```bash
export HTTPS_PROXY=http://proxy.example.com:8080
npm install -g @dev-symphony/sym
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](https://github.com/DevSymphony/sym-cli/blob/main/CONTRIBUTING.md)

## 📄 License

MIT License - see [LICENSE](https://github.com/DevSymphony/sym-cli/blob/main/LICENSE) for details

## 🔗 Links

- **GitHub**: [https://github.com/DevSymphony/sym-cli](https://github.com/DevSymphony/sym-cli)
- **Issues**: [https://github.com/DevSymphony/sym-cli/issues](https://github.com/DevSymphony/sym-cli/issues)
- **npm**: [https://www.npmjs.com/package/@dev-symphony/sym](https://www.npmjs.com/package/@dev-symphony/sym)

---

**Note**: This package is part of the Symphony project, an LLM-friendly convention linter that helps AI coding tools maintain project standards and conventions.
