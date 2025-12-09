# validator

코드 검증 오케스트레이터입니다.

정책에 정의된 규칙을 바탕으로 린터와 LLM 엔진을 조율하고, 검증 결과를 수집하여 위반 사항을 보고합니다.

## 패키지 구조

```
validator/
├── validator.go          # Validator 구조체, 4단계 파이프라인
├── execution_unit.go     # executionUnit 인터페이스 및 구현체
├── llm_validator.go      # LLM 기반 검증 로직
├── llm_validator_test.go
└── README.md
```

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `internal/cmd/validate.go` | sym validate CLI 명령어 |
| `internal/mcp/server.go` | MCP validate_code 도구 |

### 패키지 의존성

```
                  ┌─────────────┐
                  │  validator  │
                  └──────┬──────┘
        ┌────────────┬───┴───┬────────────┐
        ▼            ▼       ▼            ▼
  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
  │  linter  │ │   llm    │ │  roles   │ │   git    │
  └──────────┘ └──────────┘ └──────────┘ └──────────┘
        │
        ▼
  ┌────────────┐
  │ pkg/schema │
  └────────────┘
```

## Public / Private API

### Public API

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `Validator` | validator.go:37 | 검증 오케스트레이터 |
| `Violation` | validator.go:21 | 위반 사항 |
| `ValidationResult` | llm_validator.go:23 | 검증 결과 |
| `ValidationError` | llm_validator.go:16 | 엔진 실행 오류 |

#### Constructors

| 함수 | 파일 | 설명 |
|------|------|------|
| `NewValidator(policy, verbose)` | validator.go:49 | Validator 생성 |
| `NewValidatorWithWorkDir(policy, verbose, workDir)` | validator.go:71 | 커스텀 workDir로 생성 |

#### Methods

| 메서드 | 파일 | 설명 |
|--------|------|------|
| `(*Validator) SetLLMProvider(provider)` | validator.go:89 | LLM Provider 설정 |
| `(*Validator) ValidateChanges(ctx, changes)` | validator.go:280 | 검증 실행 |
| `(*Validator) Close()` | validator.go:413 | 리소스 정리 |

### Private API

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `llmValidator` | llm_validator.go:35 | LLM 전용 검증기 |
| `validationResponse` | llm_validator.go:241 | LLM 응답 파싱 결과 |
| `executionUnit` | execution_unit.go:21 | 실행 단위 인터페이스 |
| `linterExecutionUnit` | execution_unit.go:34 | 린터 실행 단위 |
| `llmExecutionUnit` | execution_unit.go:216 | LLM 실행 단위 |
| `ruleGroup` | validator.go:101 | 규칙 그룹화 |

#### Functions

| 함수 | 파일 | 설명 |
|------|------|------|
| `newLLMValidator(provider, policy)` | llm_validator.go:41 | llmValidator 생성 |
| `getEngineName(rule)` | validator.go:93 | 규칙에서 엔진명 추출 |
| `getDefaultConcurrency()` | validator.go:109 | 기본 동시성 레벨 |
| `getLanguageFromFile(filePath)` | validator.go:420 | 파일 확장자로 언어 판별 |
| `parseValidationResponse(response)` | llm_validator.go:256 | LLM 응답 파싱 |
| `extractJSONField(response, field)` | llm_validator.go:353 | JSON 필드 추출 |

#### Methods

| 메서드 | 파일 | 설명 |
|--------|------|------|
| `(*Validator) groupRulesByEngine(...)` | validator.go:121 | 규칙 그룹화 |
| `(*Validator) createExecutionUnits(...)` | validator.go:179 | 실행 단위 생성 |
| `(*Validator) executeUnitsParallel(...)` | validator.go:221 | 병렬 실행 |
| `(*Validator) filterChangesForRule(...)` | validator.go:384 | 규칙별 변경 필터 |
| `(*llmValidator) Validate(...)` | llm_validator.go:54 | LLM 검증 실행 |
| `(*llmValidator) checkRule(...)` | llm_validator.go:153 | 단일 규칙 검증 |
