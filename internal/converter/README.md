# converter

UserPolicy(A Schema)를 CodePolicy(B Schema)로 변환하는 핵심 모듈입니다.

LLM을 활용하여 자연어 규칙을 적절한 린터로 라우팅하고, 각 린터의 설정 파일과 `code-policy.json`을 자동 생성합니다.

## 패키지 구조

```
internal/converter/
├── converter.go   # 변환 로직 전체
└── README.md      # 이 문서
```

## 의존성

### 패키지 사용자

| 패키지 | 용도 |
|--------|------|
| `internal/cmd` | CLI `convert` 명령어 |
| `internal/mcp` | MCP 서버의 policy 변환 |
| `internal/server` | 웹 대시보드의 변환 기능 |

### 패키지 의존성

| 패키지 | 용도 |
|--------|------|
| `internal/linter` | 린터 레지스트리 및 Converter 인터페이스 |
| `internal/llm` | LLM Provider 인터페이스 |
| `pkg/schema` | UserPolicy, CodePolicy 스키마 정의 |

## Public/Private API

### Public API

#### Types

| 타입 | 설명 |
|------|------|
| `Converter` | 변환기 인스턴스. LLM Provider와 출력 디렉토리를 포함 |
| `ConvertResult` | 변환 결과. 생성된 파일 목록, CodePolicy, 오류, 경고 포함 |

#### Functions

| 함수 | 시그니처 | 설명 |
|------|----------|------|
| `NewConverter` | `(provider llm.Provider, outputDir string) *Converter` | 새 Converter 인스턴스 생성 |
| `Convert` | `(ctx context.Context, userPolicy *schema.UserPolicy) (*ConvertResult, error)` | UserPolicy를 CodePolicy와 린터 설정으로 변환 |

### Private API

| 함수 | 설명 |
|------|------|
| `routeRulesWithLLM` | LLM을 사용하여 규칙을 적합한 린터로 라우팅 |
| `getAvailableLinters` | 지정된 언어에 대해 사용 가능한 린터 목록 조회 |
| `selectLintersForRule` | 개별 규칙에 적합한 린터 선택 (LLM 활용) |
| `getLinterConverter` | 레지스트리에서 린터 변환기 조회 |
| `buildLinterDescriptions` | LLM 프롬프트용 린터 설명 문자열 생성 |
| `buildRoutingHints` | LLM 프롬프트용 라우팅 힌트 문자열 생성 |
| `convertAllTasks` | 세마포어 기반 병렬 변환 실행 |
| `convertRBAC` | UserRBAC를 PolicyRBAC로 변환 |
