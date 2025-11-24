# Linter Configuration Validation

## Purpose

The convert feature generates linter-specific configurations from natural language coding conventions. These configurations are used to validate code changes tracked by git.

## Supported Linters

### JavaScript/TypeScript
- **ESLint**: Validates JS/TS code style, patterns, and best practices
- Output: `.sym/.eslintrc.json`

### Java
- **Checkstyle**: Validates Java code formatting and style
- Output: `.sym/checkstyle.xml`
- **PMD**: Validates Java code quality and detects code smells
- Output: `.sym/pmd-ruleset.xml`

### Future Support
- **SonarQube**: Multi-language static analysis
- **LLM Validator**: Custom rules that cannot be expressed in traditional linters

## Engine Assignment

Each rule in `code-policy.json` has an `engine` field that specifies which tool validates it:

- `eslint`: Rule converted to ESLint configuration
- `checkstyle`: Rule converted to Checkstyle module
- `pmd`: Rule converted to PMD ruleset
- `sonarqube`: Future support
- `llm-validator`: Complex rules requiring LLM analysis

## Example Workflow

1. **Define conventions** in `user-policy.json`
2. **Convert** to linter configs:
   ```bash
   sym convert -i user-policy.json --targets eslint,checkstyle,pmd
   ```
3. **Run linters** on git changes:
   ```bash
   # JavaScript/TypeScript
   eslint --config .sym/.eslintrc.json src/**/*.{js,ts}

   # Java
   checkstyle -c .sym/checkstyle.xml src/**/*.java
   pmd check -R .sym/pmd-ruleset.xml -d src/
   ```

## Code Policy Schema

Generated `code-policy.json` contains:
```json
{
  "version": "1.0.0",
  "rules": [
    {
      "id": "naming-class-pascalcase",
      "engine": "eslint",
      "check": {...}
    },
    {
      "id": "security-no-secrets",
      "engine": "llm-validator",
      "check": {...}
    }
  ]
}
```

Rules with `engine: "llm-validator"` cannot be checked by traditional linters and require custom LLM-based validation.

## Testing

### Integration Test Data

Validation engines are tested using structured test data in `tests/testdata/`:

```
tests/testdata/
├── javascript/      # ESLint-based validation tests
│   ├── pattern/     # Naming conventions, regex patterns
│   ├── length/      # Line/function length limits
│   ├── style/       # Code formatting
│   └── ast/         # AST structure validation
├── typescript/      # TSC-based validation tests
│   └── typechecker/ # Type checking tests
└── java/            # Checkstyle/PMD-based validation tests
    ├── pattern/     # Naming conventions (PascalCase, camelCase)
    ├── length/      # Line/method/parameter length limits
    ├── style/       # Java formatting conventions
    └── ast/         # Code structure (exception handling, etc.)
```

Each directory contains:
- **Violation files**: Code that violates conventions (e.g., `NamingViolations.java`)
- **Valid files**: Code that complies with conventions (e.g., `ValidNaming.java`)

### Running Integration Tests

```bash
# All integration tests
go test ./tests/integration/... -v

# Specific engine tests
go test ./tests/integration/... -v -run TestPatternEngine
go test ./tests/integration/... -v -run TestLengthEngine
go test ./tests/integration/... -v -run TestStyleEngine
go test ./tests/integration/... -v -run TestASTEngine
go test ./tests/integration/... -v -run TestTypeChecker
```

For detailed test data structure, see [tests/testdata/README.md](../tests/testdata/README.md).
