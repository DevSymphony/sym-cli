# Test Data Directory

This directory contains test data files for integration testing of the validation engines.

## Directory Structure

```
testdata/
├── javascript/
│   ├── pattern/      # Pattern matching and naming convention tests
│   ├── length/       # Line, file, and function length tests
│   ├── style/        # Code formatting and style tests
│   └── ast/          # AST-based structural tests
├── typescript/
│   └── typechecker/  # Type checking tests
└── java/
    ├── pattern/      # Pattern matching and naming convention tests
    ├── length/       # Line, method, and parameter length tests
    ├── style/        # Code formatting and style tests
    └── ast/          # AST-based structural tests
```

## Engine Types

### Pattern Engine
Tests regex-based pattern matching and naming conventions.

**JavaScript Files:**
- `naming-violations.js` - Snake_case and incorrect naming patterns
- `security-violations.js` - Hardcoded secrets and security issues
- `valid.js` - Correct naming conventions

**Java Files:**
- `NamingViolations.java` - Invalid class, method, variable names
- `ValidNaming.java` - Correct PascalCase and camelCase usage

### Length Engine
Tests line length, file length, and parameter count limits.

**JavaScript Files:**
- `length-violations.js` - Long lines, long functions, too many parameters
- `valid.js` - Proper length constraints

**Java Files:**
- `LengthViolations.java` - Long lines, long methods, too many parameters
- `ValidLength.java` - Proper length constraints

### Style Engine
Tests code formatting and style conventions.

**JavaScript Files:**
- `style-violations.js` - Bad indentation, spacing, quotes
- `valid.js` - Proper formatting

**Java Files:**
- `StyleViolations.java` - Inconsistent indentation, brace placement, spacing
- `ValidStyle.java` - Standard Java formatting

### AST Engine
Tests structural patterns via Abstract Syntax Tree analysis.

**JavaScript Files:**
- `naming-violations.js` - AST-level naming issues
- `valid.js` - Valid AST structure

**Java Files:**
- `AstViolations.java` - Empty catch blocks, System.out usage, missing docs
- `ValidAst.java` - Proper exception handling and structure

### TypeChecker Engine
Tests type safety and TypeScript-specific checks.

**TypeScript Files:**
- `type-errors.ts` - Type mismatches and errors
- `strict-mode-errors.ts` - Strict mode violations
- `valid.ts` - Correct type usage

## File Naming Conventions

- **Violations**: Files containing rule violations are named `*-violations.*` or `*Violations.*`
- **Valid**: Files with compliant code are named `valid.*` or `Valid*.*`
- **Specific**: Files testing specific issues use descriptive names (e.g., `security-violations.js`)

## Adding New Test Data

When adding new test data:

1. Choose the appropriate engine directory (`pattern`, `length`, `style`, `ast`, `typechecker`)
2. Create both violation and valid files for comprehensive testing
3. Add clear comments explaining what each violation tests
4. Update integration tests to reference new files
5. Ensure files compile/parse correctly for their language

## Integration Test Usage

These files are referenced by integration tests in `tests/integration/*_integration_test.go`:

- `pattern_integration_test.go` - Uses pattern engine test data
- `length_integration_test.go` - Uses length engine test data
- `style_integration_test.go` - Uses style engine test data
- `ast_integration_test.go` - Uses ast engine test data
- `typechecker_integration_test.go` - Uses typechecker engine test data

## Validation Engines

Each engine uses specific adapters:

- **JavaScript/TypeScript**: ESLint, Prettier, TSC
- **Java**: Checkstyle, PMD

Test data files should reflect the validation capabilities of these underlying tools.
