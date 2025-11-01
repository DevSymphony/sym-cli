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

## Validation Scripts

### Validate ESLint Config
```bash
./scripts/validate-eslint.sh
```

### Validate Checkstyle Config
```bash
./scripts/validate-checkstyle.sh
```

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
3. **Validate** generated configs:
   ```bash
   ./scripts/validate-eslint.sh
   ./scripts/validate-checkstyle.sh
   ```
4. **Run linters** on git changes:
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
