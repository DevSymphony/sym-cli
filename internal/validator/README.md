# validator

Code validation orchestrator that validates code changes against policy using linters and LLM.

Implements a 4-phase pipeline:
1. RBAC permission checking
2. Rule grouping by engine
3. Execution unit creation
4. Parallel execution with concurrency control

## Package Structure

```
validator/
├── validator.go          # Main orchestrator, 4-phase validation pipeline
├── validator_test.go     # Unit tests for validator
├── execution_unit.go     # Execution unit interface and implementations
├── llm_validator.go      # LLM-based validation logic
├── llm_validator_test.go # Unit tests for LLM validator
└── README.md
```

## Dependencies

### Package Users

| Location | Purpose |
|----------|---------|
| `internal/cmd/validate.go` | CLI `sym validate` command |
| `internal/mcp/server.go` | MCP `validate_code` tool |

### Package Dependencies

| Package | Purpose |
|---------|---------|
| `internal/linter` | Linter registry and execution |
| `internal/llm` | LLM provider interface |
| `internal/roles` | RBAC permission validation |
| `internal/util/git` | Git change types and diff utilities |
| `pkg/schema` | Policy and rule definitions |

```
                  ┌─────────────┐
                  │  validator  │
                  └──────┬──────┘
        ┌────────────┬───┴───┬────────────┬────────────┐
        ▼            ▼       ▼            ▼            ▼
  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐
  │  linter  │ │   llm    │ │  roles   │ │ util/git │ │ pkg/schema │
  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────────┘
```

## Public/Private API

### Public API

#### Types

| Type | File | Description |
|------|------|-------------|
| `Validator` | validator.go | Main validation orchestrator |
| `Violation` | validator.go | Represents a policy violation |
| `ValidationResult` | llm_validator.go | Aggregated validation results |
| `ValidationError` | llm_validator.go | Engine execution error |

#### Constructors

| Function | Description |
|----------|-------------|
| `NewValidator(policy, verbose) *Validator` | Creates validator with current working directory |
| `NewValidatorWithWorkDir(policy, verbose, workDir) *Validator` | Creates validator with custom working directory |

#### Methods

| Method | Description |
|--------|-------------|
| `(*Validator) SetLLMProvider(provider)` | Sets LLM provider for llm-validator rules |
| `(*Validator) ValidateChanges(ctx, changes) (*ValidationResult, error)` | Runs 4-phase validation pipeline |
| `(*Validator) Close() error` | Releases resources |

### Private API

#### Interfaces

| Interface | File | Description |
|-----------|------|-------------|
| `executionUnit` | execution_unit.go | Polymorphic execution contract |

#### Types

| Type | File | Description |
|------|------|-------------|
| `linterExecutionUnit` | execution_unit.go | Batches rules for single linter execution |
| `llmExecutionUnit` | execution_unit.go | Single (file, rule) pair for LLM validation |
| `llmValidator` | llm_validator.go | LLM-specific validation logic |
| `ruleGroup` | validator.go | Groups rules by engine for batching |
| `validationResponse` | llm_validator.go | Parsed LLM response structure |
| `jsonValidationResponse` | llm_validator.go | JSON deserialization target |

#### Functions

| Function | File | Description |
|----------|------|-------------|
| `getEngineName(rule)` | validator.go | Extracts engine name from rule |
| `getDefaultConcurrency()` | validator.go | Returns CPU/2 bounded to [1,8] |
| `getLanguageFromFile(filePath)` | validator.go | Maps file extension to language |
| `newLLMValidator(provider, policy)` | llm_validator.go | Creates LLM validator instance |
| `parseValidationResponse(response)` | llm_validator.go | Parses LLM JSON response |
| `parseValidationResponseFallback(response)` | llm_validator.go | Fallback string-based parsing |
| `parseJSON(jsonStr, target)` | llm_validator.go | JSON unmarshaling wrapper |
| `extractJSONField(response, field)` | llm_validator.go | Manual JSON field extraction |
