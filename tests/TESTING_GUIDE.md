# Symphony CLI Testing Guide

## 전체 워크플로우

Symphony CLI는 다음과 같은 워크플로우로 동작합니다:

```
┌─────────────┐
│   1. User   │  자연어 컨벤션을 JSON으로 작성
│   Policy    │  (user-policy.json)
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ 2. Convert  │  `sym convert` 명령어로 변환
│   Command   │  자연어 → Structured Policy
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ 3. LLM Tool │  MCP를 통해 컨벤션 조회
│  (via MCP)  │  - get_conventions_by_category()
│             │  - get_all_conventions()
└──────┬──────┘
       │
       ↓
┌─────────────┐
│  4. Code    │  LLM이 컨벤션 기반 코드 생성
│ Generation  │
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ 5. Validate │  `sym validate` 명령어로 검증
│   Command   │  생성된 코드가 컨벤션 준수하는지 확인
└─────────────┘
```

## 테스트 구조

### 1. Unit Tests (API 키 불필요)

**위치**: `./internal/validator/*_test.go`

```bash
# Git diff 추출 테스트
go test ./internal/validator/... -v -run TestExtract

# LLM 응답 파싱 테스트
go test ./internal/validator/... -v -run TestParse
```

**테스트 내용**:
- Git diff에서 추가된 라인 추출
- JSON 필드 추출
- LLM 응답 파싱 (위반/비위반 판별)

### 2. Integration Tests (API 키 필요)

**위치**: `./tests/e2e/full_workflow_test.go`

```bash
# 전체 워크플로우 테스트
export OPENAI_API_KEY="sk-..."
go test ./tests/e2e/... -v -run TestE2E_FullWorkflow

# MCP 통합 테스트
go test ./tests/e2e/... -v -run TestE2E_MCPToolIntegration

# 피드백 루프 테스트
go test ./tests/e2e/... -v -run TestE2E_CodeGenerationFeedbackLoop
```

**테스트 시나리오**:

#### Scenario 1: Full Workflow
1. 자연어 정책 생성
2. LLM으로 변환
3. MCP로 조회
4. 위반 코드 검증 → 위반 검출
5. 정상 코드 검증 → 통과

#### Scenario 2: MCP Tool Integration
- `get_conventions_by_category("security")` 테스트
- `get_conventions_by_category("architecture")` 테스트
- 심각도 필터링 테스트

#### Scenario 3: Feedback Loop
- 위반 코드 생성 → 검증 → 위반 검출
- LLM이 수정 → 재검증 → 통과

### 3. MCP Integration Tests (in E2E)

**위치**: `./tests/e2e/`

**MCP 관련 파일들**:
- `.sym/js-code-policy.json`: JavaScript 컨벤션 정책 (10개 규칙)
- `examples/bad-example.js`: 10가지 위반사항 (JavaScript)
- `examples/good-example.js`: 모든 규칙 준수 (JavaScript)
- `mcp_integration_test.go`: MCP 통합 테스트
- `MCP_INTEGRATION.md`: MCP 테스트 상세 가이드

**실행 방법**:
```bash
# API 키 설정
export OPENAI_API_KEY="sk-..."

# 1. 컨벤션 조회 테스트 (API 키 불필요)
go test -v ./tests/e2e/... -run TestMCP_GetConventionsByCategory

# 2. AI 코드 검증 테스트 (API 키 필요)
go test -v ./tests/e2e/... -run TestMCP_ValidateAIGeneratedCode -timeout 3m

# 3. 전체 End-to-End 워크플로우 테스트
go test -v ./tests/e2e/... -run TestMCP_EndToEndWorkflow -timeout 3m

# 4. 전체 E2E 테스트 실행 (Go + JavaScript)
go test -v ./tests/e2e/... -timeout 5m
```

## 주요 검증 항목

### Convert 단계
- [ ] 자연어 규칙이 structured policy로 변환됨
- [ ] 카테고리가 올바르게 매핑됨
- [ ] 심각도가 유지됨
- [ ] 적용 대상 언어가 설정됨

### MCP 단계
- [ ] 카테고리별 규칙 조회 가능
- [ ] 심각도별 필터링 가능
- [ ] 전체 규칙 조회 가능
- [ ] 언어별 필터링 가능

### Validate 단계
- [ ] Git diff에서 변경사항 추출
- [ ] LLM에게 규칙과 코드 전달
- [ ] LLM 응답 파싱 (위반/비위반)
- [ ] 위반사항 보고서 생성
- [ ] 종료 코드 설정 (위반 시 1)

## 통합 테스트 데이터 구조

### testdata 디렉토리

검증 엔진의 정확성을 보장하기 위한 테스트 데이터는 `tests/testdata/` 디렉토리에 엔진별, 언어별로 구조화되어 있습니다:

```
tests/testdata/
├── javascript/
│   ├── pattern/      # 패턴 매칭 및 네이밍 컨벤션 테스트
│   │   ├── naming-violations.js
│   │   ├── security-violations.js
│   │   └── valid.js
│   ├── length/       # 라인/함수 길이 제한 테스트
│   │   ├── length-violations.js
│   │   └── valid.js
│   ├── style/        # 코드 스타일 및 포맷팅 테스트
│   │   ├── style-violations.js
│   │   └── valid.js
│   └── ast/          # AST 구조 검증 테스트
│       ├── naming-violations.js
│       └── valid.js
├── typescript/
│   └── typechecker/  # 타입 체킹 테스트
│       ├── type-errors.ts
│       ├── strict-mode-errors.ts
│       └── valid.ts
└── java/
    ├── pattern/      # Checkstyle 패턴 테스트
    │   ├── NamingViolations.java
    │   └── ValidNaming.java
    ├── length/       # Checkstyle 길이 제한 테스트
    │   ├── LengthViolations.java
    │   └── ValidLength.java
    ├── style/        # Checkstyle 스타일 테스트
    │   ├── StyleViolations.java
    │   └── ValidStyle.java
    └── ast/          # PMD AST 검증 테스트
        ├── AstViolations.java
        └── ValidAst.java
```

**파일 네이밍 컨벤션**:
- `*-violations.*` / `*Violations.*`: 규칙 위반 케이스
- `valid.*` / `Valid*.*`: 규칙 준수 케이스

각 엔진은 해당 언어의 testdata를 사용하여 통합 테스트를 실행합니다:
- Pattern Engine: 정규식 패턴 검증 (ESLint/Checkstyle)
- Length Engine: 길이 제한 검증 (ESLint/Checkstyle)
- Style Engine: 포맷팅 검증 (Prettier/Checkstyle)
- AST Engine: 구조 검증 (ESLint/PMD)
- TypeChecker Engine: 타입 검증 (TSC)

자세한 내용은 [testdata/README.md](testdata/README.md)를 참고하세요.

## E2E 테스트 데이터

### 자연어 컨벤션 예시

```json
{
  "rules": [
    {
      "say": "API 키나 비밀번호를 코드에 하드코딩하면 안됩니다. 환경변수를 사용하세요",
      "category": "security",
      "severity": "error"
    },
    {
      "say": "SQL 쿼리에 사용자 입력을 직접 연결하면 안됩니다. prepared statement를 사용하세요",
      "category": "security",
      "severity": "error"
    },
    {
      "say": "데이터베이스 접근은 반드시 repository 패턴을 통해서만 해야 합니다",
      "category": "architecture",
      "severity": "error"
    }
  ]
}
```

### 위반 코드 예시

```go
// VIOLATION: Hardcoded API key
const APIKey = "sk-1234567890abcdef"

// VIOLATION: SQL injection
query := "SELECT * FROM users WHERE id = " + userId

// VIOLATION: Direct DB access in handler
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    db.Query("SELECT * FROM users")  // Should use repository
}
```

### 정상 코드 예시

```go
// GOOD: Using environment variable
var APIKey = os.Getenv("API_KEY")

// GOOD: Parameterized query
db.Query("SELECT * FROM users WHERE id = ?", userId)

// GOOD: Using repository pattern
func HandleRequest(repo UserRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        repo.FindAll()
    }
}
```

## MCP Tool API

### get_conventions_by_category

**입력**:
```json
{
  "category": "security"
}
```

**출력**:
```json
{
  "conventions": [
    {
      "id": "SEC-001",
      "message": "No hardcoded secrets",
      "severity": "error"
    },
    {
      "id": "SEC-002",
      "message": "Use parameterized queries",
      "severity": "error"
    }
  ]
}
```

### validate_code

**입력**:
```json
{
  "code": "const APIKey = \"sk-test\"",
  "conventions": ["No hardcoded secrets"]
}
```

**출력**:
```json
{
  "violations": [
    {
      "rule_id": "SEC-001",
      "severity": "error",
      "message": "Hardcoded API key detected",
      "suggestion": "Use os.Getenv(\"API_KEY\") instead"
    }
  ]
}
```

## CI/CD 통합

### GitHub Actions 예시

```yaml
name: Validate Code Conventions

on:
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install Symphony CLI
        run: go install github.com/DevSymphony/sym-cli/cmd/sym@latest

      - name: Validate Changes
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          sym validate --staged
```

## 트러블슈팅

### API 키 오류
```bash
Error: OPENAI_API_KEY environment variable not set

# 해결:
export OPENAI_API_KEY="sk-..."
```

### 변경사항 없음
```bash
No changes to validate

# 해결:
git add <file>  # 파일을 staging
# 또는
sym validate  # unstaged 변경사항 검증 (--staged 제거)
```

### 테스트 실패
```bash
# 로그 확인
go test ./tests/e2e/... -v

# 특정 테스트만 실행
go test ./tests/e2e/... -v -run TestE2E_FullWorkflow

# 타임아웃 늘리기
go test ./tests/e2e/... -v -timeout 5m
```

## 성능 고려사항

- **convert**: 규칙 당 1-3초 (LLM 호출)
- **validate**: 규칙 당 2-5초 (LLM 호출)
- **추천 모델**:
  - 개발: `gpt-4o` (빠르고 저렴)
  - 프로덕션: `gpt-4o` (더 정확)

## 다음 단계

1. [ ] MCP 서버 구현 완료
2. [ ] Claude Code 통합 테스트
3. [ ] 실제 프로젝트에 적용
4. [ ] 성능 최적화 (캐싱, 배치 처리)
5. [ ] 추가 언어 지원 (Python, TypeScript)
