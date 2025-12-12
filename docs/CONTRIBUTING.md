# Symphony CLI 기여 가이드

Symphony CLI에 기여해 주셔서 감사합니다. 이 가이드는 시작하는 데 도움이 될 것입니다.

## 목차

- [개발 환경 설정](#개발-환경-설정)
- [프로젝트 구조](#프로젝트-구조)
- [패키지 README.md 가이드](#패키지-readmemd-가이드)
- [커밋 컨벤션](#커밋-컨벤션)
- [Pull Request 가이드](#pull-request-가이드)
- [확장 가이드](#확장-가이드)
- [테스트 가이드](#테스트-가이드)
- [참고 문서](#참고-문서)

---

## 개발 환경 설정

이 섹션에서는 Symphony CLI에 기여하기 위한 개발 환경 설정 방법을 설명합니다.

### 요구 사항

Symphony CLI를 빌드하고 실행하려면 다음 도구가 필요합니다:

| 도구 | 버전 | 설명 |
|------|------|------|
| Go | 1.25.1 | 프로그래밍 언어 |
| Git | 최신 | 버전 관리 |

### DevContainer (권장)

일관되고 미리 구성된 개발 환경을 위해 VS Code DevContainer 사용을 권장합니다. DevContainer에는 모든 필수 종속성과 도구가 포함되어 있습니다.

**사용 방법:**

1. [Docker](https://www.docker.com/)와 [VS Code](https://code.visualstudio.com/) 설치
2. VS Code에서 [Dev Containers 확장](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) 설치
3. VS Code에서 프로젝트 폴더 열기
4. `Ctrl+Shift+P` (macOS에서는 `Cmd+Shift+P`)를 누르고 **"Dev Containers: Reopen in Container"** 선택
5. 컨테이너 빌드 대기 (처음에는 몇 분 정도 소요될 수 있음)

DevContainer는 Ubuntu 24.04 기반이며 다음을 포함합니다:
- Go 1.25.1
- Git
- golangci-lint (`make setup`을 통해 설치)

컨테이너가 시작되면 `post-create.sh` 스크립트가 자동으로 `make setup`을 실행합니다.

### 로컬 설정

DevContainer 없이 로컬에서 개발하려면 다음 단계를 따르세요:

```bash
# 1. 저장소 클론
git clone https://github.com/DevSymphony/sym-cli.git
cd sym-cli

# 2. 개발 환경 설정
#    Go 종속성 다운로드 및 golangci-lint 설치
make setup

# 3. 빌드 확인
make build

# 4. 테스트 실행하여 모든 것이 작동하는지 확인
make test
```

### Make 타겟

프로젝트는 일반적인 개발 작업에 Make를 사용합니다. `make help`를 실행하여 사용 가능한 모든 타겟을 확인할 수 있습니다.

| 타겟 | 설명 |
|------|------|
| `make setup` | 개발 환경 초기화: Go 종속성 다운로드 및 golangci-lint v2.4.0 설치 |
| `make build` | 현재 플랫폼용 바이너리 빌드. 출력: `bin/sym` |
| `make build-all` | 모든 플랫폼용 빌드 (Linux amd64/arm64, macOS amd64/arm64, Windows amd64) |
| `make test` | 레이스 감지와 함께 모든 테스트 실행 및 커버리지 리포트 생성 (`coverage.html`) |
| `make unit-test` | 커버리지 없이 짧은 모드로 테스트 실행 (빠른 확인용) |
| `make lint` | golangci-lint를 실행하여 코드 품질 검사 |
| `make fmt` | `golangci-lint fmt`를 사용하여 모든 Go 소스 파일 포맷팅 |
| `make tidy` | `go mod tidy`를 실행하여 모듈 종속성 정리 |
| `make clean` | 빌드 아티팩트 제거 (`bin/`, `coverage.out`, `coverage.html`) |
| `make run ARGS="..."` | 지정된 인자로 애플리케이션 빌드 및 실행 |

**일반적인 워크플로우:**

```bash
# 커밋 전: 포맷팅, 린트, 테스트
make fmt && make lint && make test

# 개발 중 빠른 반복
make build && ./bin/sym --help

# 특정 명령 실행
make run ARGS="validate --help"
```

---

## 프로젝트 구조

프로젝트 구조를 이해하면 코드베이스를 탐색하고 어디서 변경해야 하는지 파악하는 데 도움이 됩니다.

```
sym-cli/
├── cmd/sym/                    # 진입점
│   ├── main.go                 # CLI 실행 - 루트 명령 초기화 및 실행
│   └── bootstrap.go            # 린터/LLM 프로바이더 등록을 위한 사이드 이펙트 임포트
│
├── internal/                   # 비공개 패키지 (외부 프로젝트에서 임포트 불가)
│   ├── cmd/                    # Cobra로 구현된 CLI 명령
│   ├── converter/              # 사용자 정책(Schema A)을 코드 정책(Schema B)으로 변환
│   ├── validator/              # 정책에 대한 코드 검증 조율
│   ├── linter/                 # 통합 린터 인터페이스 및 구현체
│   │   ├── eslint/             # JavaScript/TypeScript용 ESLint
│   │   ├── prettier/           # 코드 포맷팅용 Prettier
│   │   ├── pylint/             # Python용 Pylint
│   │   ├── tsc/                # 타입 검사용 TypeScript 컴파일러
│   │   ├── checkstyle/         # Java용 Checkstyle
│   │   └── pmd/                # Java 정적 분석용 PMD
│   ├── llm/                    # 통합 LLM 프로바이더 인터페이스
│   │   ├── claudecode/         # Claude Code CLI 프로바이더
│   │   ├── geminicli/          # Gemini CLI 프로바이더
│   │   └── openaiapi/          # OpenAI API 프로바이더
│   ├── mcp/                    # AI 도구 통합을 위한 Model Context Protocol 서버
│   ├── server/                 # 웹 대시보드 HTTP 서버
│   ├── importer/               # 외부 문서에서 컨벤션 추출
│   ├── policy/                 # 정책 파일 로드, 저장 및 템플릿 관리
│   ├── roles/                  # RBAC (역할 기반 접근 제어) 관리
│   └── util/                   # 공유 유틸리티
│       ├── config/             # 설정 파일 관리
│       ├── env/                # 환경 변수 및 .env 파일 처리
│       └── git/                # Git 작업 (diff, 변경된 파일 감지)
│
├── pkg/                        # 공개 패키지 (외부 프로젝트에서 임포트 가능)
│   └── schema/                 # 정책 스키마 타입 정의 (UserPolicy, CodePolicy)
│
├── docs/                       # 문서
│   ├── ARCHITECTURE.md         # 아키텍처 및 패키지 의존성
│   ├── COMMAND.md              # CLI 명령 참조
│   ├── CONVENTION_MANAGEMENT.md # 컨벤션/카테고리 관리 가이드 (MCP/Dashboard)
│   └── CONTRIBUTING.md         # 이 파일
│
└── tests/                      # 테스트 픽스처 및 통합 테스트
    └── testdata/               # 테스트 데이터 파일
```

**주요 아키텍처 개념:**

- **Schema A (UserPolicy)**: 자연어 규칙이 포함된 사람 친화적인 정책 형식
- **Schema B (CodePolicy)**: 검증 엔진용 기계 판독 가능한 정책 형식
- **Converter**: 규칙 해석을 위해 LLM을 사용하여 Schema A → Schema B 변환
- **Validator**: 린터 및 LLM 기반 검증을 통해 변환된 규칙 실행
- **Registry 패턴**: 린터와 LLM 프로바이더가 `init()` 함수를 통해 자체 등록

---

## 패키지 README.md 가이드

모든 패키지에는 목적, 구조 및 API를 문서화하는 README.md가 있어야 합니다. 이는 다른 기여자가 모든 코드를 읽지 않고도 패키지를 이해하는 데 도움이 됩니다.

### 섹션 설명

각 README.md에는 다음 섹션이 포함되어야 합니다:

#### 1. 패키지 이름 및 설명

패키지 이름을 제목으로 시작하고, 패키지가 하는 일에 대한 한 줄 설명을 추가합니다.

#### 2. 패키지 구조

각 파일의 역할에 대한 간단한 설명과 함께 디렉토리 트리를 표시합니다. 이를 통해 독자는 패키지가 어떻게 구성되어 있는지 빠르게 파악할 수 있습니다.

#### 3. 의존성

**패키지 사용자**: 이 패키지를 임포트하는 모든 다른 패키지를 나열합니다. 파일 경로와 *왜* 사용하는지 설명을 포함합니다. 이는 변경 시 무엇이 깨질 수 있는지 아는 데 도움이 됩니다.

**패키지 의존성**: 이 패키지가 임포트하는 모든 내부 패키지를 나열합니다. 표준 라이브러리나 외부 종속성이 아닌, 이 저장소의 패키지만 포함합니다.

#### 4. Public/Private API

**Public API**: 내보낸(대문자로 시작하는) 모든 타입, 함수 및 메서드를 문서화합니다. 포함할 내용:
- 타입 이름과 정의 파일
- 함수 시그니처
- 각각이 하는 일에 대한 간단한 설명

**Private API**: 주요 내부 함수를 나열합니다. 전체 시그니처는 필요 없습니다 - 함수 이름과 목적에 대한 간단한 설명만 포함합니다.

#### 5. 참고 문헌 - 선택 사항

해당되는 경우 외부 문서, 관련 스펙 또는 설계 문서 링크를 포함합니다.

### 템플릿

새 패키지 README를 작성할 때 이 템플릿을 사용하세요:

````markdown
# 패키지 이름

패키지의 목적과 기능에 대한 간결한 설명.

## 패키지 구조

```
package/
├── file1.go          # file1에 대한 간단한 설명
├── file2.go          # file2에 대한 간단한 설명
├── subpackage/       # 서브패키지에 대한 간단한 설명
│   └── ...
└── file_test.go      # 테스트
```

## 의존성

### 패키지 사용자

이 패키지를 임포트하는 모든 패키지와 사용 목적을 나열합니다.

| 위치 | 목적 |
|------|------|
| `internal/cmd/validate.go` | 정책 규칙에 대한 코드 검사에 validator 사용 |
| `internal/mcp/server.go` | MCP 도구를 통해 검증 노출 |

### 패키지 의존성

이 패키지가 의존하는 모든 내부 패키지를 나열합니다.

```
this-package
├── internal/linter      # 정적 분석 도구 실행용
├── internal/llm         # LLM 기반 검증용
├── internal/roles       # RBAC 권한 검사용
└── pkg/schema           # 정책 타입 정의용
```

## Public/Private API

### Public API

#### 타입

| 타입 | 파일 | 설명 |
|------|------|------|
| `Validator` | validator.go | 메인 검증기 조율자 |
| `ValidationResult` | validator.go | 위반 사항 및 메타데이터 포함 |

#### 함수

| 함수 | 시그니처 | 설명 |
|------|----------|------|
| `NewValidator` | `func NewValidator(policy *schema.CodePolicy, verbose bool) *Validator` | 새 검증기 인스턴스 생성 |
| `ValidateChanges` | `func (v *Validator) ValidateChanges(ctx context.Context, changes []Change) (*ValidationResult, error)` | 정책에 대한 코드 변경 검증 |

### Private API

외부 사용을 위해 내보내지 않는 내부 함수:

- `groupRulesByEngine()` - 검증 엔진 타입별 규칙 그룹화
- `buildExecutionUnits()` - 병렬 실행 단위 생성
- `mergeResults()` - 여러 검증기의 결과 결합

### 참고 문헌

- [외부 도구 문서](https://example.com)
- 관련 내부 문서
````

---

## 커밋 컨벤션

우리는 간소화된 Conventional Commits 형식을 따릅니다. 일관된 커밋 메시지는 프로젝트 히스토리를 이해하고 변경 로그를 생성하기 쉽게 만듭니다.

### 형식

```
<type>: <subject>

[선택적 본문]
```

- **type**: 변경 카테고리 (아래 표 참조)
- **subject**: 변경 내용의 짧은 설명 (명령형, 소문자, 마침표 없음)
- **body**: 선택적 상세 설명 (제목만으로 충분하지 않을 때)

### 커밋 타입

| 타입 | 사용 시점 | 예시 |
|------|----------|------|
| `feat` | 새 기능 추가 | `feat: add OpenAI provider support` |
| `fix` | 버그 수정 | `fix: handle nil pointer in validator` |
| `docs` | 문서만 변경 | `docs: update API reference` |
| `style` | 포맷팅, 공백 (코드 변경 없음) | `style: fix indentation in converter` |
| `refactor` | 버그 수정도 기능 추가도 아닌 코드 변경 | `refactor: extract git operations to util package` |
| `test` | 테스트 추가 또는 업데이트 | `test: add unit tests for RBAC validation` |
| `chore` | 빌드 프로세스, 종속성, 도구 | `chore: update golangci-lint to v2.4.0` |

### 규칙

1. **제목 줄**: 소문자로 시작, 끝에 마침표 없음, 50자 이내
2. **명령형**: "add feature"로 작성, "added feature"나 "adds feature"가 아님
3. **본문**: 변경 사항이 제목 이상의 설명을 필요로 할 때 사용
4. **범위**: 범위(예: `feat(llm):`)를 사용하지 않음 - 단순하게 유지

### 예시

**간단한 커밋 (제목만):**
```
feat: add retry logic for API calls
```

**본문이 있는 커밋 (복잡한 변경용):**
```
feat: add exponential backoff for LLM API calls

- Retry up to 3 times with exponential backoff
- Initial delay: 1 second, max delay: 10 seconds
- Configurable via LLM_RETRY_COUNT environment variable
```

**버그 수정:**
```
fix: prevent panic when policy file is empty

Return early with descriptive error instead of attempting
to parse nil content.
```

---

## Pull Request 가이드

이 섹션에서는 이 프로젝트의 pull request 생성 및 작성 방법을 설명합니다.

### PR 제목

PR 제목은 커밋 메시지와 동일한 형식을 따라야 합니다:

```
<type>: <subject>
```

"Squash and Merge" 사용 시 제목이 커밋 메시지가 되므로 설명적으로 작성하세요.

**좋은 예시:**
- `feat: add Gemini CLI provider`
- `fix: resolve race condition in parallel validation`
- `docs: add contribution guide`
- `refactor: simplify linter registry initialization`

**나쁜 예시:**
- `Update code` (너무 모호함)
- `Fixed bug` (타입 접두사 누락, 설명적이지 않음)
- `feat: Add New Feature for Better Performance` (모든 단어를 대문자로 하지 않음)

### PR 본문 형식

PR 설명에 이 템플릿을 사용하세요:

```markdown
## Summary
- 이 PR이 무엇을 하고 왜 하는지에 대한 간단한 설명
- "어떻게"가 아닌 "무엇"과 "왜"에 집중

## Changes
- 수행된 구체적인 변경 사항 목록
- 한 줄에 하나씩
- 영향받는 파일이나 컴포넌트를 구체적으로 명시
```

**Summary에 포함할 내용:**
- 해결하는 문제 또는 추가하는 기능
- 이 변경이 필요한 이유
- 리뷰어가 알아야 할 중요한 컨텍스트

**Changes에 포함할 내용:**
- 추가된 새 파일 또는 패키지
- 수정된 기능
- 삭제되거나 더 이상 사용되지 않는 코드
- 설정 변경

### PR 예시

**제목:** `feat: add Claude API provider`

**본문:**
```markdown
## Summary
- Add support for Claude API as a new LLM provider
- Enables users without Claude CLI to use the API directly
- Includes automatic retry with exponential backoff

## Changes
- Add `internal/llm/claudeapi/` package with provider implementation
- Implement `RawProvider` interface with streaming support
- Add API key validation in provider initialization
- Register provider in `cmd/sym/bootstrap.go`
- Add unit tests for provider (85% coverage)
- Update `internal/llm/README.md` with new provider documentation
```

### 병합 전략

모든 pull request에 **Squash and Merge**를 사용합니다.

**의미:**
- PR의 모든 커밋이 단일 커밋으로 결합됨
- PR 제목이 최종 커밋 메시지가 됨
- 개별 커밋 히스토리는 PR에 보존되지만 main 브랜치에는 포함되지 않음

**이유:**
- main 브랜치 히스토리를 깔끔하고 읽기 쉽게 유지
- 각 PR = 하나의 논리적 변경 = 하나의 커밋
- 필요시 전체 기능을 쉽게 되돌릴 수 있음

**병합 전 확인사항:**
1. 모든 CI 검사 통과
2. 코드 리뷰 및 승인 완료
3. PR 제목이 커밋 컨벤션을 따름
4. PR 설명이 완성됨

---

## 확장 가이드

Symphony CLI는 확장 가능하게 설계되었습니다. 새로운 LLM 프로바이더나 린터를 추가할 수 있습니다.

### LLM 프로바이더 추가

새 프로바이더를 추가하려면:
1. `internal/llm/<provider-name>/` 패키지 생성
2. `RawProvider` 인터페이스 구현
3. `init()`에서 `llm.RegisterProvider()` 호출
4. `cmd/sym/bootstrap.go`에 빈 임포트 추가

상세 가이드와 코드 예제는 [`internal/llm/README.md`](../internal/llm/README.md)를 참조하세요.

### 린터 추가

새 린터를 추가하려면:
1. `internal/linter/<name>/` 패키지 생성
2. `Linter` 인터페이스 구현 (도구 실행)
3. `Converter` 인터페이스 구현 (규칙 변환)
4. `init()`에서 `linter.Global().RegisterTool()` 호출
5. `cmd/sym/bootstrap.go`에 빈 임포트 추가

상세 가이드와 코드 예제는 [`internal/linter/README.md`](../internal/linter/README.md)를 참조하세요.

---

## 테스트 가이드

이 섹션에서는 Symphony CLI의 테스트 작성 및 실행 방법을 설명합니다.

### 테스트 실행

Make 타겟을 사용하여 테스트 실행:

```bash
# 레이스 감지 및 커버리지와 함께 모든 테스트 실행
# coverage.html 파일 생성
make test

# 커버리지 없이 빠른 테스트 실행 (개발 반복용으로 더 빠름)
make unit-test

# 특정 패키지 테스트 실행
go test -v ./internal/validator/...

# 특정 테스트 함수 실행
go test -v ./internal/validator/... -run TestValidateChanges

# 자세한 출력과 캐싱 없이 테스트 실행
go test -v -count=1 ./...
```

### 테스트 작성 가이드라인

테스트 작성 시 다음 가이드라인을 따르세요:

1. **파일 명명**: 테스트 파일은 `_test.go`로 끝나야 함
2. **함수 명명**: 테스트 함수는 `Test`로 시작해야 함 (예: `TestValidateChanges`)
3. **패키지**: 테스트는 테스트 대상 코드와 같은 패키지에 있어야 함
4. **어설션**: 읽기 쉬운 어설션을 위해 `github.com/stretchr/testify` 사용

### 테스트 구조

명확한 테스트 구조를 위해 Arrange-Act-Assert 패턴 사용:

```go
func TestValidator_ValidateChanges(t *testing.T) {
    // Arrange: 테스트 데이터 및 종속성 설정
    policy := &schema.CodePolicy{
        Rules: []schema.Rule{
            {ID: "1", Category: "naming"},
        },
    }
    validator := NewValidator(policy, false)
    changes := []Change{
        {File: "test.go", Content: "package main"},
    }

    // Act: 테스트 대상 코드 실행
    result, err := validator.ValidateChanges(context.Background(), changes)

    // Assert: 결과 검증
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.Empty(t, result.Violations)
}
```

### 테이블 기반 테스트

여러 시나리오 테스트에 테이블 기반 테스트 사용:

```go
func TestParseOutput(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []Violation
        wantErr  bool
    }{
        {
            name:     "유효한 JSON 출력",
            input:    `[{"file": "test.go", "line": 10, "message": "error"}]`,
            expected: []Violation{{File: "test.go", Line: 10, Message: "error"}},
            wantErr:  false,
        },
        {
            name:     "빈 출력",
            input:    `[]`,
            expected: []Violation{},
            wantErr:  false,
        },
        {
            name:    "잘못된 JSON",
            input:   `not json`,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseOutput(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 모킹

종속성 모킹이 필요한 단위 테스트용:

```go
// 종속성용 인터페이스 정의
type LLMProvider interface {
    Execute(ctx context.Context, prompt string) (string, error)
}

// 모의 구현 생성
type mockProvider struct {
    response string
    err      error
}

func (m *mockProvider) Execute(ctx context.Context, prompt string) (string, error) {
    return m.response, m.err
}

// 테스트에서 사용
func TestConverter_ConvertRule(t *testing.T) {
    mock := &mockProvider{
        response: `{"rule": "value"}`,
        err:      nil,
    }

    converter := NewConverter(mock)
    result, err := converter.ConvertRule(context.Background(), rule)

    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 커버리지

`make test` 실행 후 커버리지 데이터가 `coverage.html`에 저장됩니다.

```bash
# 브라우저에서 열기 (macOS)
open coverage.html
```

---

## 참고 문서

| 문서 | 설명 |
|------|------|
| [docs/COMMAND.md](COMMAND.md) | 예제가 포함된 완전한 CLI 명령 참조 |
| [internal/llm/README.md](../internal/llm/README.md) | 상세 LLM 프로바이더 구현 가이드 |
| [internal/linter/README.md](../internal/linter/README.md) | 상세 린터 통합 가이드 |
