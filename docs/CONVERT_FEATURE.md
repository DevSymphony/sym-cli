# Convert Feature - Multi-Target Linter Configuration Generator

## Overview

The enhanced `convert` command transforms natural language coding conventions into linter-specific configuration files using LLM-powered inference.

## Features

- **LLM-Powered Inference**: Uses OpenAI API to intelligently analyze natural language rules
- **Multi-Target Support**: Generates configurations for multiple linters simultaneously
  - **ESLint** (JavaScript/TypeScript)
  - **Checkstyle** (Java)
  - **PMD** (Java)
- **Fallback Mechanism**: Pattern-based inference when LLM is unavailable
- **Confidence Scoring**: Tracks inference confidence and warns on low-confidence conversions
- **1:N Rule Mapping**: One user rule can generate multiple linter rules
- **Caching**: Minimizes API calls by caching inference results

## Architecture

```
User Policy (user-policy.json)
       ↓
   Converter
       ↓
   LLM Inference (OpenAI API) ← Fallback (Pattern Matching)
       ↓
   Rule Intent Detection
       ↓
   Linter Converters
       ├── ESLint Converter → .eslintrc.json
       ├── Checkstyle Converter → checkstyle.xml
       └── PMD Converter → pmd-ruleset.xml
```

### Package Structure

```
internal/
├── llm/
│   ├── client.go          # OpenAI API client
│   ├── inference.go       # Rule inference engine
│   └── types.go           # Intent and result types
├── converter/
│   ├── converter.go       # Main conversion logic
│   └── linters/
│       ├── converter.go   # Linter converter interface
│       ├── registry.go    # Converter registry
│       ├── eslint.go      # ESLint converter
│       ├── checkstyle.go  # Checkstyle converter
│       └── pmd.go         # PMD converter
└── cmd/
    └── convert.go         # CLI command
```

## Usage

### Basic Usage

```bash
# Convert to all supported linters
sym convert -i user-policy.json --targets all --output-dir .linters

# Convert only for JavaScript/TypeScript
sym convert -i user-policy.json --targets eslint --output-dir .linters

# Convert for Java
sym convert -i user-policy.json --targets checkstyle,pmd --output-dir .linters
```

### Advanced Options

```bash
# Use specific OpenAI model
sym convert -i user-policy.json \
  --targets all \
  --output-dir .linters \
  --openai-model gpt-4o

# Adjust confidence threshold
sym convert -i user-policy.json \
  --targets eslint \
  --output-dir .linters \
  --confidence-threshold 0.8

# Enable verbose output
sym convert -i user-policy.json \
  --targets all \
  --output-dir .linters \
  --verbose
```

### Legacy Mode

```bash
# Generate only internal code-policy.json (no linter configs)
sym convert -i user-policy.json -o code-policy.json
```

## Configuration

### Environment Variables

- `OPENAI_API_KEY`: OpenAI API key (required for LLM inference)
  - If not set, fallback pattern-based inference is used

### Flags

- `--targets`: Target linters (comma-separated or "all")
- `--output-dir`: Output directory for generated files
- `--openai-model`: OpenAI model to use (default: gpt-4o)
- `--confidence-threshold`: Minimum confidence for inference (default: 0.7)
- `--timeout`: API call timeout in seconds (default: 30)
- `--verbose`: Enable verbose logging

## User Policy Schema

### Example

```json
{
  "version": "1.0.0",
  "defaults": {
    "severity": "error",
    "autofix": false
  },
  "rules": [
    {
      "id": "naming-class-pascalcase",
      "say": "Class names must be PascalCase",
      "category": "naming",
      "languages": ["javascript", "typescript", "java"],
      "params": {
        "case": "PascalCase"
      }
    },
    {
      "id": "length-max-line",
      "say": "Maximum line length is 100 characters",
      "category": "length",
      "params": {
        "max": 100
      }
    }
  ]
}
```

### Supported Categories

- `naming`: Identifier naming conventions
- `length`: Size constraints (line/file/function length)
- `style`: Code formatting (indentation, quotes, semicolons)
- `complexity`: Cyclomatic/cognitive complexity
- `security`: Security-related rules
- `error_handling`: Exception handling patterns
- `dependency`: Import/dependency restrictions

### Supported Engine Types

- `pattern`: Naming conventions, forbidden patterns, import restrictions
- `length`: Line/file/function length, parameter count
- `style`: Indentation, quotes, semicolons, whitespace
- `ast`: Cyclomatic complexity, nesting depth
- `custom`: Rules that don't fit other categories

## Output Files

### Generated Files

1. **`.eslintrc.json`**: ESLint configuration for JavaScript/TypeScript
2. **`checkstyle.xml`**: Checkstyle configuration for Java
3. **`pmd-ruleset.xml`**: PMD ruleset for Java
4. **`code-policy.json`**: Internal validation policy
5. **`conversion-report.json`**: Detailed conversion report

### Conversion Report Format

```json
{
  "timestamp": "2025-10-30T19:52:22+09:00",
  "input_file": "user-policy.json",
  "total_rules": 5,
  "targets": ["eslint", "checkstyle", "pmd"],
  "openai_model": "gpt-4o",
  "confidence_threshold": 0.7,
  "linters": {
    "eslint": {
      "rules_generated": 5,
      "warnings": 2,
      "errors": 0
    }
  },
  "warnings": [
    "eslint: Rule 2: low confidence (0.40 < 0.70): Maximum line length is 100 characters"
  ]
}
```

## LLM Inference

### How It Works

1. **Cache Check**: First checks if the rule has been inferred before
2. **LLM Analysis**: Sends rule to OpenAI API with structured prompt
3. **Intent Extraction**: Parses JSON response to extract:
   - Engine type (pattern/length/style/ast)
   - Category (naming/security/etc.)
   - Target (identifier/content/import)
   - Scope (line/file/function)
   - Parameters (max, min, case, etc.)
   - Confidence score (0.0-1.0)
4. **Fallback**: If LLM fails, uses pattern matching
5. **Conversion**: Maps intent to linter-specific rules

### System Prompt

The LLM is instructed to:
- Analyze natural language coding rules
- Extract structured intent with high precision
- Provide confidence scores for interpretations
- Return results in strict JSON format

### Fallback Inference

When LLM is unavailable, pattern-based rules detect:
- **Naming rules**: Keywords like "PascalCase", "camelCase", "name"
- **Length rules**: Keywords like "line", "length", "max", "characters"
- **Style rules**: Keywords like "indent", "spaces", "tabs", "quote"
- **Security rules**: Keywords like "secret", "password", "hardcoded"
- **Import rules**: Keywords like "import", "dependency", "layer"

## Example: Rule Conversion Flow

### Input Rule
```json
{
  "say": "Class names must be PascalCase",
  "category": "naming",
  "languages": ["javascript", "java"]
}
```

### LLM Inference Result
```json
{
  "engine": "pattern",
  "category": "naming",
  "target": "identifier",
  "params": {"case": "PascalCase"},
  "confidence": 0.95
}
```

### ESLint Output
```json
{
  "rules": {
    "id-match": ["error", "^[A-Z][a-zA-Z0-9]*$", {
      "properties": false,
      "classFields": false,
      "onlyDeclarations": true
    }]
  }
}
```

### Checkstyle Output
```xml
<module name="TypeName">
  <property name="format" value="^[A-Z][a-zA-Z0-9]*$"/>
  <property name="severity" value="error"/>
</module>
```

### PMD Output
```xml
<rule ref="category/java/codestyle.xml/ClassNamingConventions">
  <priority>1</priority>
</rule>
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run LLM inference tests
go test ./internal/llm/...

# Run linter converter tests
go test ./internal/converter/linters/...
```

### Integration Test

```bash
# Test with example policy
./bin/sym convert \
  -i tests/testdata/user-policy-example.json \
  --targets all \
  --output-dir /tmp/test-output \
  --verbose
```

## Limitations

### ESLint
- Limited support for complex AST patterns
- Some rules require custom ESLint plugins
- Style rules may conflict with Prettier

### Checkstyle
- Module configuration can be complex
- Some rules require additional checks
- Limited support for custom patterns

### PMD
- Rule references must match PMD versions
- Property configuration varies by rule
- Some categories have limited coverage

### LLM Inference
- Requires OpenAI API key (costs apply)
- May produce incorrect interpretations for complex rules
- Confidence scores are estimates
- Network dependency and latency

## Future Enhancements

- [ ] Support for additional linters (Pylint, RuboCop, etc.)
- [ ] Custom rule templates
- [ ] Rule conflict detection
- [ ] Interactive mode for ambiguous rules
- [ ] Cost estimation for LLM API calls
- [ ] Local LLM support (Ollama, etc.)
- [ ] Rule similarity clustering
- [ ] Automatic rule categorization
- [ ] Multi-language rule mapping optimization

## Troubleshooting

### "OPENAI_API_KEY not set"
- Set environment variable: `export OPENAI_API_KEY=sk-...`
- Or use fallback mode (lower accuracy)

### "low confidence" warnings
- Increase `--confidence-threshold` to reduce warnings
- Provide more specific `category` and `params` in rules
- Use LLM instead of fallback for better accuracy

### Generated rules don't work
- Check linter version compatibility
- Verify rule syntax in linter documentation
- Adjust rule parameters manually if needed
- Report issue with conversion-report.json

### Slow conversion
- Reduce number of rules
- Use caching (re-run with same rules)
- Increase `--timeout` for large rule sets
- Use faster OpenAI model (gpt-4o)

## Performance

### Benchmarks (5 rules, no cache)

- **With LLM (gpt-4o)**: ~5-10 seconds
- **Fallback only**: <1 second
- **With cache**: <100ms

### Cost Estimation

- **gpt-4o**: ~$0.001 per rule
- **gpt-4o**: ~$0.01 per rule
- **Caching**: Reduces cost by ~90% for repeated rules

## Contributing

When adding new linter support:

1. Implement `LinterConverter` interface in `internal/converter/linters/`
2. Register converter in `init()` function
3. Add tests in `*_test.go`
4. Update this documentation
5. Add example output to `tests/testdata/`

## License

Same as sym-cli project license.
