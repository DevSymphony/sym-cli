# mcp

MCP (Model Context Protocol) 서버 구현

AI 코딩 도구(Claude Code, Cursor 등)와 stdio를 통해 통신하며,
컨벤션 조회와 코드 검증 기능을 제공합니다.

## 패키지 구조

```
mcp/
├── server.go       # MCP 서버 구현 (NewServer, Start, 도구 핸들러)
├── server_test.go  # query_conventions 테스트
└── README.md
```

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `internal/cmd/mcp.go` | `sym mcp` CLI 명령어 |

### 패키지 의존성

```
              ┌───────────┐
              │    mcp    │
              └─────┬─────┘
    ┌───────┬───────┼───────┬───────┬────────┐
    ▼       ▼       ▼       ▼       ▼        ▼
┌─────────┐ ┌─────┐ ┌───────┐ ┌─────┐ ┌─────────┐ ┌────────┐
│converter│ │ llm │ │policy │ │roles│ │validator│ │importer│
└─────────┘ └─────┘ └───────┘ └─────┘ └─────────┘ └────────┘
                                │
                        ┌───────┴───────┐
                        ▼               ▼
                  ┌──────────┐   ┌────────────┐
                  │ util/git │   │ pkg/schema │
                  └──────────┘   └────────────┘
```

## Public / Private API

### Public API

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `Server` | server.go:78 | MCP 서버 인스턴스 |
| `RPCError` | server.go:184 | JSON-RPC 에러 타입 |
| `QueryConventionsInput` | server.go:190 | query_conventions 입력 스키마 |
| `ValidateCodeInput` | server.go:196 | validate_code 입력 스키마 |
| `ListCategoryInput` | server.go:201 | list_category 입력 스키마 |
| `CategoryItem` | server.go:206 | 카테고리 항목 (배치용) |
| `CategoryEditItem` | server.go:212 | 카테고리 편집 항목 (배치용) |
| `AddCategoryInput` | server.go:225 | add_category 입력 스키마 |
| `EditCategoryInput` | server.go:230 | edit_category 입력 스키마 |
| `RemoveCategoryInput` | server.go:235 | remove_category 입력 스키마 |
| `ImportConventionsInput` | server.go:240 | import_convention 입력 스키마 |
| `QueryConventionsRequest` | server.go:330 | 컨벤션 조회 요청 |
| `ConventionItem` | server.go:250 | 컨벤션 항목 |
| `ValidateCodeRequest` | server.go:411 | 검증 요청 |
| `ViolationItem` | server.go:416 | 위반 항목 |
| `ValidationResultRecord` | server.go:720 | 검증 결과 레코드 |
| `ValidationHistory` | server.go:731 | 검증 이력 |

#### Functions

| 함수 | 파일 | 설명 |
|------|------|------|
| `ConvertPolicyWithLLM(userPath, codePath)` | server.go:25 | LLM으로 정책 변환 |
| `NewServer(configPath)` | server.go:86 | 서버 생성 |

#### Methods

| 메서드 | 파일 | 설명 |
|--------|------|------|
| `(*Server) Start()` | server.go:95 | MCP 서버 시작 |

### Private API

| 함수/메서드 | 설명 |
|-------------|------|
| `runStdioWithSDK(ctx)` | MCP SDK로 stdio 서버 실행 |
| `handleQueryConventions(params)` | 컨벤션 조회 핸들러 |
| `filterConventions(req)` | 컨벤션 필터링 |
| `isRuleRelevant(rule, req)` | 규칙 관련성 확인 |
| `handleValidateCode(ctx, session, params)` | 코드 검증 핸들러 |
| `containsAny(haystack, needles)` | 배열 교집합 확인 |
| `getValidationPolicy()` | 검증용 정책 반환 |
| `needsConversion(codePolicyPath)` | 변환 필요 여부 확인 |
| `extractSourceRuleID(id)` | 원본 규칙 ID 추출 |
| `convertUserPolicy(userPath, codePath)` | 정책 변환 래퍼 |
| `getRBACInfo()` | RBAC 정보 생성 |
| `saveValidationResults(result, violations, hasErrors)` | 검증 결과 저장 |
| `handleListCategory()` | 카테고리 목록 핸들러 |
| `handleAddCategory(input)` | 카테고리 추가 핸들러 |
| `handleEditCategory(input)` | 카테고리 편집 핸들러 |
| `handleRemoveCategory(input)` | 카테고리 삭제 핸들러 |
| `handleImportConventions(ctx, input)` | import_convention 핸들러 |
| `saveUserPolicy()` | 정책 파일 저장 |

## 참고 문헌

- [MCP 도구 스키마](../../docs/COMMAND.md#mcp-도구-스키마) - query_conventions, validate_code, list_category, add_category, edit_category, remove_category, import_convention 입력/출력 스펙
- [MCP 통합 가이드](../../docs/COMMAND.md#mcp-통합) - 지원 도구 및 등록 방법
