# Testdata Directory

This directory contains test files for integration and unit tests.

## Structure

```
testdata/
├── javascript/     # JavaScript test files
├── typescript/     # TypeScript test files
└── mixed/          # JSX/TSX test files
```

## JavaScript Test Files

### Naming Violations
- `naming-violations.js` - Contains snake_case, lowercase class names
- `bad-naming.js` - Additional naming convention violations

### Style Violations
- `style-violations.js` - Indentation, quote, semicolon issues
- `bad-style.js` - Additional style violations

### Length Violations
- `length-violations.js` - General length violations
- `long-lines.js` - Lines exceeding max length
- `long-function.js` - Functions exceeding max lines
- `long-file.js` - Files exceeding max lines
- `many-params.js` - Functions with too many parameters

### Security Violations
- `security-violations.js` - Hardcoded credentials
- `hardcoded-secrets.js` - API keys, passwords

### AST/Error Handling
- `async-with-try.js` - Async code with proper try/catch
- `async-without-try.js` - Async code without error handling

### Import Violations
- `bad-imports.js` - Restricted import patterns

### Valid Code
- `valid.js` - Well-formatted, compliant code
- `good-code.js` - Additional valid examples
- `good-style.js` - Properly styled code

## TypeScript Test Files

- `type-errors.ts` - Type assignment errors
- `strict-mode-errors.ts` - Strict mode violations
- `valid.ts` - Valid TypeScript code

## Mixed Files

- `component.jsx` - React JSX component
- `component.tsx` - React TypeScript component

## Usage

Integration tests reference these files via:
```go
filepath.Join(workDir, "testdata/javascript/naming-violations.js")
```

Where `workDir` is the project root directory.
