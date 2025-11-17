# Setup Guide

This document describes the development environment setup and external tool requirements for the Symphony CLI project.

## Prerequisites

### Required Tools

#### 1. Go (Required)
- **Version**: Go 1.21 or higher
- **Purpose**: Build and run the CLI
- **Installation**: https://go.dev/doc/install

#### 2. Node.js & npm (Required for JavaScript/TypeScript validation)
- **Version**: Node.js 18+ with npm
- **Purpose**: Run ESLint, Prettier, TSC adapters
- **Installation**: https://nodejs.org/
- **Used by engines**: Pattern, Length, Style, AST, TypeChecker

#### 3. Java JDK (Required for Java validation)
- **Version**: Java 21 or higher (OpenJDK recommended)
- **Purpose**: Run Checkstyle and PMD adapters
- **Installation**:
  ```bash
  # Ubuntu/Debian
  sudo apt-get update
  sudo apt-get install -y default-jdk

  # macOS (Homebrew)
  brew install openjdk@21

  # Verify installation
  java -version
  ```
- **Used by engines**: Pattern (Java), Length (Java), Style (Java), AST (Java)
- **Note**: Java was installed on 2025-11-16 for integration testing

### Optional Tools

#### Git
- Required for running integration tests that involve git operations
- Most systems have git pre-installed

---

## External Tool Installation

The CLI automatically installs external validation tools when first needed:

### JavaScript/TypeScript Tools
- **ESLint**: Installed to `~/.symphony/tools/node_modules/eslint` via npm
- **Prettier**: Installed to `~/.symphony/tools/node_modules/prettier` via npm
- **TypeScript**: Installed to `~/.symphony/tools/node_modules/typescript` via npm

### Java Tools
- **Checkstyle**: Downloaded to `~/.symphony/tools/checkstyle-10.26.1.jar` from Maven Central
- **PMD**: Downloaded to `~/.symphony/tools/pmd-<version>/` from GitHub Releases

**Auto-installation happens when:**
1. A validation rule is executed for the first time
2. The engine checks if the adapter is available (`CheckAvailability()`)
3. If not found, the engine calls `Install()` to download and set up the tool

---

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Integration Tests
```bash
# Run all integration tests
go test ./tests/integration/...

# Run specific test suite
go test ./tests/integration/validator_policy_test.go ./tests/integration/helper.go -v

# Skip integration tests in short mode
go test -short ./...
```

**Note**: Integration tests require:
- Node.js and npm (for JavaScript/TypeScript tests)
- Java JDK (for Java tests)

---

## Troubleshooting

### Java Tests Fail with "java not found"
```bash
# Install Java JDK
sudo apt-get install -y default-jdk

# Verify installation
java -version  # Should show: openjdk version "21.0.8"
```

### ESLint/Prettier Installation Fails
```bash
# Ensure npm is available
npm --version

# Manually install tools
cd ~/.symphony/tools
npm install eslint@^8.0.0 prettier@latest
```

### Checkstyle Download Fails (HTTP 404)
- The Checkstyle version was updated to 10.26.1 on 2025-11-16
- If you see 404 errors, check `/workspace/internal/adapter/checkstyle/adapter.go` for the correct version
- Maven Central URL: https://repo1.maven.org/maven2/com/puppycrawl/tools/checkstyle/

---

## Development Environment

### Recommended VS Code Extensions
- Go (golang.go)
- YAML (redhat.vscode-yaml)
- JSON (vscode.json-language-features)

### Directory Structure
```
~/.symphony/tools/          # Auto-installed external tools
  ├── node_modules/         # JavaScript/TypeScript tools
  │   ├── eslint/
  │   ├── prettier/
  │   └── typescript/
  ├── checkstyle-10.26.1.jar  # Java style checker
  └── pmd-<version>/          # Java static analyzer
```

---

## Installation History

### 2025-11-16: Java JDK Installation
- **Installed**: OpenJDK 21.0.8
- **Reason**: Required for Java validation integration tests
- **Method**: `sudo apt-get install -y default-jdk`
- **Verification**: `java -version` shows OpenJDK 21.0.8+9
- **Impact**: Enables Checkstyle and PMD adapters for Java code validation
