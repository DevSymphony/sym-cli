# LLM 패키지

LLM 프로바이더를 위한 통합 인터페이스입니다.

## 파일 구조

```
internal/llm/
├── llm.go           # Provider, RawProvider 인터페이스, Config, ResponseFormat, ModelInfo, APIKeyConfig, ProviderInfo
├── registry.go      # 프로바이더 레지스트리 및 유틸리티 함수
├── wrapper.go       # parsedProvider (자동 파싱 래퍼)
├── config.go        # LoadConfig, LoadConfigFromDir, Config.Validate
├── parser.go        # 응답 파싱 (비공개)
├── claudecode/      # Claude Code CLI 프로바이더
├── geminicli/       # Gemini CLI 프로바이더
└── openaiapi/       # OpenAI API 프로바이더
```

## 사용법

```go
import "github.com/DevSymphony/sym-cli/internal/llm"

// 1. 설정 로드
cfg := llm.LoadConfig()

// 2. 프로바이더 생성
provider, err := llm.New(cfg)
if err != nil {
    return err // CLI 미설치 또는 API 키 누락
}

// 3. 프롬프트 실행
response, err := provider.Execute(ctx, prompt, llm.JSON)
```

### 설정

설정 파일: `.sym/config.json`

```json
{
  "llm": {
    "provider": "claudecode",
    "model": "sonnet"
  }
}
```

OpenAI API 사용 시 `.sym/.env`에 API 키 추가 필요:

```bash
OPENAI_API_KEY=sk-...
```

### 응답 형식

| 형식 | 설명 |
|------|------|
| `llm.Text` | 원시 텍스트 반환 |
| `llm.JSON` | LLM 응답에서 JSON 추출 |
| `llm.XML` | LLM 응답에서 XML 추출 |

`llm.JSON`과 `llm.XML`은 LLM이 서두 텍스트와 함께 JSON/XML을 반환할 때 자동으로 구조화된 데이터를 추출합니다.

## 프로바이더 목록

| 이름 | 유형 | 기본 모델 | 설치 방법 |
|------|------|-----------|-----------|
| `claudecode` | CLI | sonnet | `npm i -g @anthropic-ai/claude-cli` |
| `geminicli` | CLI | gemini-2.5-flash | `npm i -g @google/gemini-cli` |
| `openaiapi` | API | gpt-4o-mini | `OPENAI_API_KEY` 환경 변수 설정 |

### 프로바이더 상태 확인

```go
providers := llm.ListProviders()
for _, p := range providers {
    fmt.Printf("%s: available=%v\n", p.Name, p.Available)
}
```

## 새 프로바이더 추가

### 1단계: 디렉토리 생성

```
internal/llm/<provider>/
└── provider.go
```

### 2단계: RawProvider 인터페이스 구현

```go
package myprovider

import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/llm"
)

type Provider struct {
    model string
}

// 컴파일 타임 검사: Provider가 RawProvider 인터페이스를 구현하는지 확인
var _ llm.RawProvider = (*Provider)(nil)

func (p *Provider) Name() string {
    return "myprovider"
}

func (p *Provider) ExecuteRaw(ctx context.Context, prompt string, format llm.ResponseFormat) (string, error) {
    // 프롬프트 실행 후 원시 응답 반환
    response := callLLM(prompt)
    return response, nil  // 수동 파싱 불필요
}

func (p *Provider) Close() error {
    return nil
}
```

> **참고**: `var _ llm.RawProvider = (*Provider)(nil)` 라인은 `Provider`가 `RawProvider` 인터페이스를 구현하는지 확인하는 컴파일 타임 검사입니다. 메서드가 누락되었거나 시그니처가 잘못된 경우 컴파일에 실패합니다.

### 3단계: init()에서 등록

```go
func init() {
    llm.RegisterProvider("myprovider", newProvider, llm.ProviderInfo{
        Name:         "myprovider",
        DisplayName:  "My Provider",
        DefaultModel: "default-model-v1",
        Available:    checkAvailability(), // CLI 존재 또는 API 키 설정 여부 확인
        Path:         cliPath,             // CLI 경로 (API의 경우 빈 문자열)
        Models: []llm.ModelInfo{
            {ID: "model-v1", DisplayName: "Model V1", Description: "표준 모델", Recommended: true},
            {ID: "model-v2", DisplayName: "Model V2", Description: "고급 모델", Recommended: false},
        },
        APIKey: llm.APIKeyConfig{
            Required:   true,                    // CLI 기반 프로바이더는 false로 설정
            EnvVarName: "MY_PROVIDER_API_KEY",   // 환경 변수 이름
            Prefix:     "mp-",                   // 검증용 예상 접두사 (선택사항)
        },
    })
}

func newProvider(cfg llm.Config) (llm.RawProvider, error) {
    // CLI 미설치 또는 API 키 누락 시 에러 반환
    if !isAvailable() {
        return nil, fmt.Errorf("provider not available")
    }

    model := cfg.Model
    if model == "" {
        model = "default-model-v1"
    }

    return &Provider{model: model}, nil
}
```

### 4단계: 부트스트랩에 import 추가

```go
// cmd/sym/bootstrap.go
import (
    _ "github.com/DevSymphony/sym-cli/internal/llm/myprovider"
)
```

### 핵심 규칙

- 컴파일 타임 검사 추가: `var _ llm.RawProvider = (*Provider)(nil)`
- `RawProvider.ExecuteRaw()` 구현 - 파싱은 래퍼에서 자동 처리
- CLI 미설치 또는 API 키 누락 시 명확한 에러 메시지 반환
- init()에서 가용성을 확인하여 ProviderInfo.Available 설정
- `Models`에 최소 하나의 모델을 `Recommended: true`로 정의
- CLI 기반 프로바이더는 `APIKey.Required: false` 설정 (자체적으로 인증 처리)
- 기존 프로바이더(claudecode, openaiapi) 패턴 참조
