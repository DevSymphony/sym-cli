# Linter 패키지

정적 린팅 도구를 위한 통합 인터페이스.

## 파일 구조

```
internal/linter/
├── linter.go        # Linter 인터페이스 (실행)
├── converter.go     # Converter 인터페이스 (규칙 변환)
├── registry.go      # 전역 레지스트리, RegisterTool(), GetLinter()
├── helpers.go       # CleanJSONResponse, DefaultToolsDir, WriteTempConfig
├── subprocess.go    # SubprocessExecutor
├── eslint/          # JavaScript/TypeScript
├── prettier/        # 코드 포맷팅
├── pylint/          # Python
├── tsc/             # TypeScript 타입 검사
├── checkstyle/      # Java 스타일
└── pmd/             # Java 정적 분석

각 린터 서브디렉토리 구성:
├── linter.go      # Linter 구현
├── converter.go   # Converter 구현
├── executor.go    # 도구 실행 로직
├── parser.go      # 출력 파싱
├── register.go    # init() 등록
└── *_test.go      # 단위 테스트 (eslint, prettier, pylint, tsc)
```

## 사용법

```go
import "github.com/DevSymphony/sym-cli/internal/linter"

// 1. 이름으로 린터 가져오기
l, err := linter.Global().GetLinter("eslint")
if err != nil {
    return err
}

// 2. 가용성 확인 및 필요시 설치
if err := l.CheckAvailability(ctx); err != nil {
    if err := l.Install(ctx, linter.InstallConfig{}); err != nil {
        return err
    }
}

// 3. 린터 실행
output, err := l.Execute(ctx, config, files)
if err != nil {
    return err
}

// 4. 출력을 위반 사항으로 파싱
violations, err := l.ParseOutput(output)
```

### Converter 가져오기

```go
converter, ok := linter.Global().GetConverter("eslint")
if !ok {
    return fmt.Errorf("converter not found")
}

// 단일 규칙 변환 (메인 컨버터가 병렬로 호출)
result, err := converter.ConvertSingleRule(ctx, rule, llmProvider)

// 결과로 설정 빌드
config, err := converter.BuildConfig(results)
```

### Registry 메서드

```go
// 핵심 메서드
linter.Global()                          // 싱글톤 Registry 인스턴스 반환
linter.Global().RegisterTool(l, c, cfg)  // 린터, 컨버터, 설정 파일 등록
linter.Global().GetLinter("eslint")      // 이름으로 Linter 가져오기 (없으면 에러 반환)
linter.Global().GetConverter("eslint")   // 이름으로 Converter 가져오기 (bool 반환)
linter.Global().GetConfigFile("eslint")  // 설정 파일명 가져오기 (예: ".eslintrc.json")

// 일괄 조회 메서드
linter.Global().GetAllToolNames()        // []string - 등록된 모든 도구 이름
linter.Global().GetAllConfigFiles()      // []string - 모든 설정 파일명
linter.Global().GetAllConverters()       // []Converter - 등록된 모든 컨버터
linter.Global().BuildLanguageMapping()   // map[string][]string - 언어별 도구 매핑
```

## 린터 목록

| 이름 | 언어 | 설정 파일 |
|------|------|----------|
| `eslint` | JavaScript, TypeScript, JSX, TSX | `.eslintrc.json` |
| `prettier` | JS, TS, JSON, CSS, HTML, Markdown | `.prettierrc` |
| `pylint` | Python | `.pylintrc` |
| `tsc` | TypeScript | `tsconfig.json` |
| `checkstyle` | Java | `checkstyle.xml` |
| `pmd` | Java | `pmd.xml` |

## 새 린터 추가

### 1단계: 디렉토리 생성

```
internal/linter/<name>/
├── linter.go      # Linter 구현
├── converter.go   # Converter 구현
├── executor.go    # 도구 실행 로직
├── parser.go      # 출력 파싱
└── register.go    # init() 등록
```

### 2단계: Linter 인터페이스 구현

```go
package mylinter

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/linter"
)

// 컴파일 타임 검사
var _ linter.Linter = (*Linter)(nil)

type Linter struct {
    ToolsDir string
    executor *linter.SubprocessExecutor
}

func New(toolsDir string) *Linter {
    if toolsDir == "" {
        toolsDir = linter.DefaultToolsDir()
    }
    return &Linter{
        ToolsDir: toolsDir,
        executor: linter.NewSubprocessExecutor(),
    }
}

func (l *Linter) Name() string { return "mylinter" }

func (l *Linter) GetCapabilities() linter.Capabilities {
    return linter.Capabilities{
        Name:                "mylinter",
        SupportedLanguages:  []string{"ruby"},
        SupportedCategories: []string{"pattern", "style"},
        Version:             "1.0.0",
    }
}

func (l *Linter) CheckAvailability(ctx context.Context) error {
    // 도구 바이너리 존재 여부 확인
}

func (l *Linter) Install(ctx context.Context, cfg linter.InstallConfig) error {
    // 도구 설치 (gem install, pip install, npm install 등)
}

func (l *Linter) Execute(ctx context.Context, config []byte, files []string) (*linter.ToolOutput, error) {
    // 도구 실행 및 출력 반환
}

func (l *Linter) ParseOutput(output *linter.ToolOutput) ([]linter.Violation, error) {
    // 도구별 출력을 위반 사항으로 파싱
}
```

### 3단계: Converter 인터페이스 구현

```go
package mylinter

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/linter"
    "github.com/DevSymphony/sym-cli/internal/llm"
    "github.com/DevSymphony/sym-cli/pkg/schema"
)

// 컴파일 타임 검사
var _ linter.Converter = (*Converter)(nil)

type Converter struct{}

func NewConverter() *Converter { return &Converter{} }

func (c *Converter) Name() string { return "mylinter" }

func (c *Converter) SupportedLanguages() []string {
    return []string{"ruby"}
}

func (c *Converter) GetLLMDescription() string {
    return `Ruby 코드 품질 (스타일, 네이밍, 복잡도)
  - 가능: 네이밍 규칙, 줄 길이, 순환 복잡도
  - 불가: 비즈니스 로직, 런타임 동작`
}

func (c *Converter) GetRoutingHints() []string {
    return []string{
        "Ruby 코드 스타일 → mylinter 사용",
        "Ruby 네이밍 규칙 → mylinter 사용",
    }
}

// ConvertSingleRule은 하나의 사용자 규칙을 린터별 데이터로 변환합니다.
// 동시성은 메인 컨버터가 관리하며, 여기서는 처리하지 않습니다.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
    // LLM을 호출하여 규칙 변환
    config, err := c.callLLM(ctx, rule, provider)
    if err != nil {
        return nil, err
    }

    // 이 린터로 규칙을 적용할 수 없으면 nil, nil 반환
    if config == nil {
        return nil, nil
    }

    return &linter.SingleRuleResult{
        RuleID: rule.ID,
        Data:   config,  // 린터별 데이터
    }, nil
}

// BuildConfig는 모든 성공적인 변환 결과로 최종 설정을 조립합니다.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
    if len(results) == 0 {
        return nil, nil
    }

    // 결과로 설정 빌드
    config := buildMyLinterConfig(results)

    content, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return nil, err
    }

    return &linter.LinterConfig{
        Filename: ".mylinter.yml",
        Content:  content,
        Format:   "yaml",
    }, nil
}
```

### 4단계: init()에서 등록

```go
package mylinter

import "github.com/DevSymphony/sym-cli/internal/linter"

func init() {
    _ = linter.Global().RegisterTool(
        New(linter.DefaultToolsDir()),
        NewConverter(),
        ".mylinter.yml",
    )
}
```

### 5단계: Bootstrap에 임포트 추가

```go
// cmd/sym/bootstrap.go
import (
    _ "github.com/DevSymphony/sym-cli/internal/linter/mylinter"
)
```

## 주요 규칙

- 두 인터페이스 모두에 컴파일 타임 검사 추가:
  ```go
  var _ linter.Linter = (*Linter)(nil)
  var _ linter.Converter = (*Converter)(nil)
  ```
- `ConvertSingleRule()`은 하나의 규칙만 처리 - 동시성은 메인 컨버터가 관리
- 규칙을 적용할 수 없으면 `ConvertSingleRule()`에서 `(nil, nil)` 반환 (llm-validator로 폴백)
- LLM 응답에서 마크다운 펜스 제거에 `linter.CleanJSONResponse()` 사용
- 도구가 설치되지 않은 경우 명확한 오류 메시지 반환
- 패턴 참고를 위해 기존 린터 (eslint, pylint) 참조
