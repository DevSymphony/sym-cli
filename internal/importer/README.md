# importer

외부 문서에서 LLM을 사용하여 코딩 컨벤션을 추출하고 정책 파일에 병합합니다.

텍스트, 마크다운, 코드 파일 등 다양한 포맷의 문서에서 컨벤션을 자동으로 인식하여
Symphony 정책 형식으로 변환합니다.

## 패키지 구조

```
importer/
├── types.go       # 타입 정의 (ImportMode, ImportInput, ImportResult 등)
├── reader.go      # 파일 읽기 및 포맷 검증 (31개 포맷 지원, 50KB 제한)
├── extractor.go   # LLM 기반 컨벤션 추출 및 JSON 파싱
├── importer.go    # Import 워크플로우 조율
└── README.md
```

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `internal/cmd/import.go` | `sym import` CLI 명령어 |
| `internal/mcp/server.go` | `import_convention` MCP 도구 |
| `internal/server/server.go` | `/api/import` HTTP 엔드포인트 |

### 패키지 의존성

```
            ┌──────────┐
            │ importer │
            └────┬─────┘
     ┌───────────┼───────────┐
     ▼           ▼           ▼
┌─────────┐ ┌────────┐ ┌───────────┐
│   llm   │ │ policy │ │ pkg/schema│
└─────────┘ └────────┘ └───────────┘
```

## Import 워크플로우

`Import()` 메서드는 6단계 워크플로우를 실행합니다:

```
1. 파일 읽기 (Reader.ReadFile)
      │
      ▼
2. LLM 추출 (Extractor.Extract)
      │
      ▼
3. 기존 정책 로드 (policy.LoadPolicy)
      │
      ▼
4. Import 모드 적용
   ├─ append: 기존 유지
   └─ clear: 기존 삭제
      │
      ▼
5. 고유 ID 할당 및 병합
   └─ 중복 카테고리: 건너뛰기
   └─ 중복 규칙 ID: 접미어 추가 (e.g., SEC-001-2)
      │
      ▼
6. 정책 저장 (policy.SavePolicy)
```

## Public / Private API

### Public API

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `ImportMode` | types.go:6 | Import 모드 (`append` / `clear`) |
| `ImportInput` | types.go:16 | Import 입력 (경로, 모드) |
| `ImportResult` | types.go:22 | Import 결과 (추가된 항목, 경고) |
| `DocumentContent` | types.go:32 | 파싱된 문서 내용 |
| `ExtractedConventions` | types.go:39 | LLM 추출 결과 |
| `Importer` | importer.go:13 | Import 워크플로우 조율자 |
| `Reader` | reader.go:57 | 파일 읽기 담당 |
| `Extractor` | extractor.go:15 | LLM 추출 담당 |

#### Constants

| 상수 | 파일 | 값 | 설명 |
|------|------|-----|------|
| `ImportModeAppend` | types.go:10 | `"append"` | 기존 유지 모드 |
| `ImportModeClear` | types.go:12 | `"clear"` | 기존 삭제 모드 |
| `MaxFileSizeBytes` | reader.go:54 | `50 * 1024` | 최대 파일 크기 (50KB) |

#### Variables

| 변수 | 파일 | 설명 |
|------|------|------|
| `SupportedFormats` | reader.go:12 | 지원 파일 확장자 맵 |

#### Functions

| 함수 | 파일 | 설명 |
|------|------|------|
| `NewImporter(provider, verbose)` | importer.go:20 | Importer 인스턴스 생성 |
| `NewReader(verbose)` | reader.go:62 | Reader 인스턴스 생성 |
| `NewExtractor(provider, verbose)` | extractor.go:21 | Extractor 인스턴스 생성 |
| `IsSupportedFormat(ext)` | reader.go:122 | 지원 포맷 여부 확인 |
| `GetSupportedExtensions()` | reader.go:127 | 지원 확장자 목록 반환 |

#### Methods

| 메서드 | 파일 | 설명 |
|--------|------|------|
| `(*Importer) Import(ctx, input)` | importer.go:29 | Import 워크플로우 실행 |
| `(*Reader) ReadFile(ctx, filePath)` | reader.go:67 | 파일 읽기 및 검증 |
| `(*Extractor) Extract(ctx, doc)` | extractor.go:29 | LLM으로 컨벤션 추출 |

### Private API

#### Importer (importer.go)

| 함수/메서드 | 설명 |
|-------------|------|
| `assignUniqueIDs(existing, extracted, result)` | 고유 ID 생성 및 중복 처리 |
| `generateUniqueID(baseID, existingIDs)` | 고유 규칙 ID 생성 |
| `updateDefaultsLanguages(policy, newRules)` | defaults.languages 업데이트 |

#### Extractor (extractor.go)

| 함수/메서드 | 설명 |
|-------------|------|
| `buildExtractionPrompt(content, filename)` | LLM 프롬프트 생성 |
| `parseExtractionResponse(response, source)` | LLM JSON 응답 파싱 |
| `cleanJSONResponse(response)` | JSON 응답 정리 (마크다운 펜싱 제거) |
| `normalizeCategory(category)` | 카테고리명 정규화 |
| `normalizeSeverity(severity)` | 심각도 정규화 |
| `truncateString(s, maxLen)` | 문자열 자르기 |

## 지원 파일 포맷

총 31개 포맷을 지원합니다:

| 카테고리 | 확장자 |
|----------|--------|
| 텍스트 문서 | `.txt`, `.md`, `.markdown` |
| 코드 파일 | `.go`, `.js`, `.ts`, `.jsx`, `.tsx`, `.py`, `.java`, `.rs`, `.rb`, `.php`, `.c`, `.cpp`, `.h`, `.hpp`, `.cs`, `.swift`, `.kt`, `.scala` |
| 설정/데이터 | `.yaml`, `.yml`, `.json`, `.toml`, `.xml` |
| 웹 파일 | `.html`, `.htm`, `.css`, `.scss`, `.less` |
| 기타 | `.rst`, `.adoc` |

## 사용 예시

```go
import (
    "context"
    "github.com/DevSymphony/sym-cli/internal/importer"
    "github.com/DevSymphony/sym-cli/internal/llm"
)

func example() {
    // LLM 프로바이더 생성
    provider, _ := llm.NewProvider()

    // Importer 생성
    imp := importer.NewImporter(provider, false)

    // Import 실행
    result, err := imp.Import(context.Background(), &importer.ImportInput{
        Path: "coding-standards.md",
        Mode: importer.ImportModeAppend,
    })

    if err != nil {
        // 에러 처리
    }

    fmt.Printf("추가된 카테고리: %d\n", len(result.CategoriesAdded))
    fmt.Printf("추가된 규칙: %d\n", len(result.RulesAdded))
}
```

## 참고 문헌

- [sym import 명령어](/docs/COMMAND.md#sym-import) - CLI 사용법
- [import_convention MCP 도구](/docs/COMMAND.md#import_convention) - MCP 도구 스키마
- [pkg/schema](/pkg/schema/README.md) - UserPolicy, UserRule 타입 정의
