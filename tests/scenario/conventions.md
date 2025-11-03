# Test Scenario: Natural Language Coding Conventions

This document contains natural language coding conventions to test the LLM validator.

## Security Conventions

### 1. No Hardcoded Secrets
**Convention**: Never hardcode API keys, passwords, or secrets in the source code. Use environment variables or secure vaults instead.

**Examples of Violations**:
- `apiKey := "sk-1234567890abcdef"`
- `password := "mySecretPassword123"`
- `token := "ghp_xxxxxxxxxxxx"`

**Correct Approach**:
- `apiKey := os.Getenv("OPENAI_API_KEY")`
- `password := os.Getenv("DB_PASSWORD")`

### 2. No SQL Injection Vulnerabilities
**Convention**: Never concatenate user input directly into SQL queries. Always use parameterized queries or prepared statements.

**Examples of Violations**:
- `query := "SELECT * FROM users WHERE id = " + userId`
- `db.Exec("DELETE FROM posts WHERE id = " + postId)`

**Correct Approach**:
- `db.Query("SELECT * FROM users WHERE id = ?", userId)`
- `db.Exec("DELETE FROM posts WHERE id = ?", postId)`

## Architecture Conventions

### 3. Repository Pattern for Database Access
**Convention**: All database operations must go through repository interfaces. Direct database calls from handlers or controllers are not allowed.

**Examples of Violations**:
- Calling `db.Query()` directly in HTTP handlers
- Using `db.Exec()` in business logic functions

**Correct Approach**:
- Define repository interfaces in domain layer
- Implement repositories in infrastructure layer
- Inject repositories into handlers/services

### 4. Dependency Injection Required
**Convention**: Do not instantiate dependencies inside functions. All dependencies must be injected through constructors or function parameters.

**Examples of Violations**:
```go
func ProcessOrder(orderId int) {
    db := sql.Open("postgres", "...") // Bad: creating dependency inside
    // ...
}
```

**Correct Approach**:
```go
func ProcessOrder(db *sql.DB, orderId int) {
    // Good: dependency injected
}
```

## Error Handling Conventions

### 5. No Panic in Production Code
**Convention**: Never use `panic()` in production code except for truly unrecoverable situations during initialization. Always return errors instead.

**Examples of Violations**:
- `panic("failed to connect to database")`
- `panic(err)`

**Correct Approach**:
- `return fmt.Errorf("failed to connect: %w", err)`
- `return err`

### 6. Always Check Error Returns
**Convention**: Every function that returns an error must have its error checked immediately. No ignored errors allowed.

**Examples of Violations**:
```go
file.Write(data) // Not checking error
json.Marshal(obj) // Not checking error
```

**Correct Approach**:
```go
if _, err := file.Write(data); err != nil {
    return err
}
```

## Code Quality Conventions

### 7. No Magic Numbers
**Convention**: Numeric literals (except 0, 1) must be defined as named constants with clear meaning.

**Examples of Violations**:
- `time.Sleep(300 * time.Second)`
- `if score > 85 { ... }`

**Correct Approach**:
```go
const (
    DefaultTimeout = 300 * time.Second
    PassingScore = 85
)
```

### 8. Function Complexity Limit
**Convention**: Functions should not have more than 3 levels of nested control structures (if/for/switch). Extract complex logic into helper functions.

**Examples of Violations**:
```go
func ProcessData(data []Item) {
    for _, item := range data {
        if item.Valid {
            for _, child := range item.Children {
                if child.Active {
                    for _, tag := range child.Tags {
                        // Too deep!
                    }
                }
            }
        }
    }
}
```

## Testing Conventions

### 9. Test Coverage for Public APIs
**Convention**: Every exported function must have at least one test case. Test file must be in the same package with `_test.go` suffix.

**Examples of Violations**:
- Exported function without any tests
- Test file in different package

### 10. Table-Driven Tests Required
**Convention**: When testing multiple scenarios, use table-driven test pattern instead of writing separate test functions.

**Examples of Violations**:
```go
func TestAddPositive(t *testing.T) { ... }
func TestAddNegative(t *testing.T) { ... }
func TestAddZero(t *testing.T) { ... }
```

**Correct Approach**:
```go
func TestAdd(t *testing.T) {
    tests := []struct{
        name string
        a, b int
        want int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## Commit Message Conventions

### 11. Conventional Commit Format
**Convention**: Commit messages must follow the format: `<type>(<scope>): <subject>`

Types: feat, fix, docs, style, refactor, test, chore

**Examples of Violations**:
- "updated some files"
- "fix bug"
- "WIP"

**Correct Approach**:
- "feat(auth): add OAuth2 login support"
- "fix(api): resolve race condition in user cache"
- "docs(readme): update installation instructions"

## Documentation Conventions

### 12. Public API Documentation Required
**Convention**: All exported types, functions, and methods must have godoc comments explaining their purpose, parameters, and return values.

**Examples of Violations**:
```go
func ProcessPayment(amount float64, currency string) error {
    // Missing documentation
}
```

**Correct Approach**:
```go
// ProcessPayment processes a payment transaction for the given amount and currency.
// It returns an error if the payment fails or if the currency is not supported.
func ProcessPayment(amount float64, currency string) error {
    // implementation
}
```
