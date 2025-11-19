# Convert Command Usage Guide

## Quick Start

Convert natural language rules to linter configurations:

```bash
# Convert to all supported linters (outputs to <git-root>/.sym)
sym convert -i user-policy.json --targets all

# Convert only for JavaScript/TypeScript
sym convert -i user-policy.json --targets eslint

# Convert for Java
sym convert -i user-policy.json --targets checkstyle,pmd
```

## Default Output Directory

**Important**: The convert command automatically creates a `.sym` directory at your git repository root and saves all generated files there.

### Directory Structure

```
your-project/
├── .git/
├── .sym/                      # Auto-generated
│   ├── .eslintrc.json        # ESLint config
│   ├── checkstyle.xml        # Checkstyle config
│   ├── pmd-ruleset.xml       # PMD config
│   ├── code-policy.json      # Internal policy
│   └── conversion-report.json # Conversion report
├── src/
└── user-policy.json          # Your input file
```

### Why .sym?

- **Consistent location**: Always at git root, easy to find
- **Version control**: Add to `.gitignore` to keep generated files out of git
- **CI/CD friendly**: Scripts can always find configs at `<git-root>/.sym`

### Custom Output Directory

If you need a different location:

```bash
sym convert -i user-policy.json --targets all --output-dir ./custom-dir
```

## Prerequisites

1. **Git repository**: Run the command from within a git repository
2. **OpenAI API key** (optional): Set `OPENAI_API_KEY` for better inference
   ```bash
   export OPENAI_API_KEY=sk-...
   ```

Without API key, fallback pattern matching is used (lower accuracy).

## User Policy File

Create a `user-policy.json` with natural language rules:

```json
{
  "version": "1.0.0",
  "defaults": {
    "severity": "error",
    "autofix": false
  },
  "rules": [
    {
      "say": "Class names must be PascalCase",
      "category": "naming",
      "languages": ["javascript", "typescript", "java"]
    },
    {
      "say": "Maximum line length is 100 characters",
      "category": "length"
    },
    {
      "say": "Use 4 spaces for indentation",
      "category": "style"
    }
  ]
}
```

## Command Options

### Basic Options

- `-i, --input`: Input user policy file (default: `user-policy.json`)
- `--targets`: Target linters (comma-separated or `all`)
  - `eslint` - JavaScript/TypeScript
  - `checkstyle` - Java
  - `pmd` - Java
  - `all` - All supported linters

### Advanced Options

- `--output-dir`: Custom output directory (default: `<git-root>/.sym`)
- `--openai-model`: OpenAI model (default: `gpt-4o`)
  - `gpt-4o` - Fast, cheap, good quality
  - `gpt-4o` - Slower, more expensive, best quality
- `--confidence-threshold`: Minimum confidence (default: `0.7`)
  - Range: 0.0 to 1.0
  - Lower values = more rules converted, more warnings
- `--timeout`: API timeout in seconds (default: `30`)
- `-v, --verbose`: Enable detailed logging

### Legacy Mode

Generate only internal `code-policy.json`:

```bash
sym convert -i user-policy.json -o code-policy.json
```

## Example Workflows

### JavaScript/TypeScript Project

```bash
# 1. Create user-policy.json
cat > user-policy.json <<EOF
{
  "rules": [
    {
      "say": "Class names must be PascalCase",
      "category": "naming"
    },
    {
      "say": "Use single quotes for strings",
      "category": "style"
    }
  ]
}
EOF

# 2. Convert to ESLint config
sym convert -i user-policy.json --targets eslint

# 3. Use generated config
npx eslint --config .sym/.eslintrc.json src/
```

### Java Project

```bash
# 1. Create user-policy.json with Java rules
sym convert -i user-policy.json --targets checkstyle,pmd

# 2. Use Checkstyle
java -jar checkstyle.jar -c .sym/checkstyle.xml src/

# 3. Use PMD
pmd check -R .sym/pmd-ruleset.xml -d src/
```

### Multi-Language Project

```bash
# Convert for all languages at once
sym convert -i user-policy.json --targets all

# Files generated:
# - .sym/.eslintrc.json (JS/TS)
# - .sym/checkstyle.xml (Java)
# - .sym/pmd-ruleset.xml (Java)
```

## Output Files

### Generated Files

1. **`.eslintrc.json`**: ESLint configuration
   - Rules: naming, length, style, complexity
   - Comments with original "say" text

2. **`checkstyle.xml`**: Checkstyle configuration
   - Modules: TypeName, LineLength, Indentation, etc.
   - Properties configured from params

3. **`pmd-ruleset.xml`**: PMD ruleset
   - Rule references to PMD categories
   - Priority mapped from severity

4. **`code-policy.json`**: Internal validation policy
   - Used by `sym validate` command
   - Structured rule definitions

5. **`conversion-report.json`**: Conversion report
   - Statistics per linter
   - Warnings and errors
   - Confidence scores

### Conversion Report

The report helps you understand what was converted:

```json
{
  "timestamp": "2025-10-30T21:13:00+09:00",
  "total_rules": 5,
  "targets": ["eslint", "checkstyle", "pmd"],
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

## Confidence and Warnings

### Confidence Scores

Rules are assigned confidence scores (0.0-1.0):
- **High (0.8-1.0)**: Strong match, rule will work well
- **Medium (0.6-0.8)**: Good match, may need tweaking
- **Low (0.4-0.6)**: Uncertain, check generated rule
- **Very Low (0.0-0.4)**: Pattern matching fallback only

### Low Confidence Warnings

When a rule has low confidence:
```
⚠ eslint: Rule 2: low confidence (0.40 < 0.70): Maximum line length is 100 characters
```

**What to do:**
1. Review the generated rule in the config file
2. Adjust manually if needed
3. Provide more specific `category` and `params` in user rule
4. Use OpenAI API instead of fallback for better accuracy

### Adjusting Threshold

```bash
# Stricter (fewer warnings, may miss rules)
sym convert -i user-policy.json --targets all --confidence-threshold 0.8

# More lenient (more warnings, more rules)
sym convert -i user-policy.json --targets all --confidence-threshold 0.5
```

## Troubleshooting

### "not in a git repository"

**Problem**: Command fails with git repository error

**Solution**:
```bash
# Either initialize git
git init

# Or use custom output directory
sym convert --targets all --output-dir ./linter-configs
```

### "OPENAI_API_KEY not set"

**Problem**: Warning about missing API key

**Solution**:
```bash
# Set the API key (recommended)
export OPENAI_API_KEY=sk-your-key-here

# Or accept fallback mode (lower accuracy)
# Conversion will still work, just with pattern matching
```

### Generated rules don't work

**Problem**: Linter rejects generated config

**Solution**:
1. Check linter version compatibility
2. Review conversion-report.json for errors
3. Manually adjust the problematic rule
4. Report issue with your user-policy.json

### Slow conversion

**Problem**: Takes too long to convert

**Solution**:
```bash
# Increase timeout
sym convert --targets all --timeout 60

# Use faster model (slightly less accurate)
sym convert --targets all --openai-model gpt-4o

# Or split rules into smaller batches
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Validate Code Conventions

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Convert policies
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          ./sym convert -i user-policy.json --targets all

      - name: Run ESLint
        run: npx eslint --config .sym/.eslintrc.json src/

      - name: Run Checkstyle
        run: java -jar checkstyle.jar -c .sym/checkstyle.xml src/
```

### GitLab CI

```yaml
validate:
  script:
    - ./sym convert -i user-policy.json --targets all
    - npx eslint --config .sym/.eslintrc.json src/
  only:
    - merge_requests
```

## Tips and Best Practices

### Writing Better Rules

1. **Be specific**: "Class names must be PascalCase" vs "use good names"
2. **Include params**: Provide numeric values in `params` field
3. **Set category**: Helps inference choose correct engine type
4. **Specify languages**: Target specific languages when needed

### Managing Generated Files

```bash
# Add to .gitignore (already done by default)
echo ".sym/" >> .gitignore

# But commit user-policy.json
git add user-policy.json
git commit -m "Add coding conventions policy"
```

### Sharing Configs

```bash
# Share with team
git add .sym/*.{json,xml}
git commit -m "Add generated linter configs"

# Or regenerate on each machine
# (Each developer runs: sym convert -i user-policy.json --targets all)
```

### Updating Rules

```bash
# 1. Edit user-policy.json
# 2. Regenerate configs
sym convert -i user-policy.json --targets all

# 3. Review changes
git diff .sym/

# 4. Apply to project
npx eslint --config .sym/.eslintrc.json src/
```

## Next Steps

- [Full Feature Documentation](CONVERT_FEATURE.md)
- [User Policy Schema Reference](../tests/testdata/user-policy-example.json)
- [Contributing Guide](../AGENTS.md)
