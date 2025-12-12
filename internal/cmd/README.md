# cmd

Symphony CLI 명령어를 구현합니다.

Cobra 프레임워크 기반으로 init, validate, convert, policy, dashboard, my-role, llm, mcp, import, version 등의 명령어를 제공합니다.

## 패키지 구조

```
cmd/
├── root.go              # 루트 명령어, Execute(), 전역 플래그
├── version.go           # sym version 명령어
├── colors.go            # 터미널 포맷팅 유틸리티 (ANSI 색상)
├── init.go              # sym init 명령어 (프로젝트 초기화)
├── dashboard.go         # sym dashboard 명령어 (웹 대시보드)
├── my_role.go           # sym my-role 명령어 (역할 관리)
├── policy.go            # sym policy path|validate 명령어 (정책 관리)
├── validate.go          # sym validate 명령어 (코드 검증)
├── convert.go           # sym convert 명령어 (정책 변환)
├── llm.go               # sym llm status|test|setup 명령어 (LLM 관리)
├── mcp.go               # sym mcp 명령어 (MCP 서버)
├── mcp_register.go      # MCP 서버 등록 헬퍼 함수
├── category.go          # sym category list|add|edit|remove 명령어 (카테고리 관리)
├── convention.go        # sym convention list|add|edit|remove 명령어 (컨벤션 관리)
├── import.go            # sym import 명령어 (외부 문서에서 컨벤션 추출)
├── survey_templates.go  # 커스텀 survey UI 템플릿
└── README.md
```

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `cmd/sym/main.go` | CLI 진입점, `Execute()` 및 `SetVersion()` 호출 |

### 패키지 의존성

```
                              ┌───────────┐
                              │    cmd    │
                              └─────┬─────┘
        ┌──────┬──────┬───────┬─────┼─────┬──────┬──────┬──────┐
        ▼      ▼      ▼       ▼     ▼     ▼      ▼      ▼      ▼
   converter  llm  validator policy roles server  mcp  linter importer
        │      │      │       │     │     │      │      │      │
        └──────┴──────┴───────┴─────┴─────┴──────┴──────┴──────┘
                                    │
                     ┌──────────────┼──────────────┐
                     ▼              ▼              ▼
                util/config     util/git       util/env
                                    │
                                    ▼
                               pkg/schema
```

## Public / Private API

### Public API

#### Functions

| 함수 | 파일 | 설명 |
|------|------|------|
| `Execute()` | root.go:26 | CLI 실행 진입점 |
| `SetVersion(v string)` | version.go:26 | 버전 문자열 설정 (main.go에서 호출) |

### Private API

#### 명령어 변수 (cobra.Command)

| 변수 | 파일 | 설명 |
|------|------|------|
| `rootCmd` | root.go:13 | 루트 명령어 (sym) |
| `versionCmd` | version.go:12 | version 명령어 |
| `initCmd` | init.go:18 | init 명령어 |
| `dashboardCmd` | dashboard.go:13 | dashboard 명령어 |
| `myRoleCmd` | my_role.go:14 | my-role 명령어 |
| `policyCmd` | policy.go:13 | policy 명령어 |
| `policyPathCmd` | policy.go:24 | policy path 명령어 |
| `policyValidateCmd` | policy.go:31 | policy validate 명령어 |
| `validateCmd` | validate.go:23 | validate 명령어 |
| `convertCmd` | convert.go:22 | convert 명령어 |
| `llmCmd` | llm.go:18 | llm 명령어 |
| `llmStatusCmd` | llm.go:33 | llm status 명령어 |
| `llmTestCmd` | llm.go:40 | llm test 명령어 |
| `llmSetupCmd` | llm.go:47 | llm setup 명령어 |
| `mcpCmd` | mcp.go:15 | mcp 명령어 |
| `categoryCmd` | category.go:10 | category 명령어 |
| `conventionCmd` | convention.go:44 | convention 명령어 |
| `conventionListCmd` | convention.go:60 | convention list 명령어 |
| `conventionAddCmd` | convention.go:68 | convention add 명령어 |
| `conventionEditCmd` | convention.go:93 | convention edit 명령어 |
| `conventionRemoveCmd` | convention.go:115 | convention remove 명령어 |
| `importCmd` | import.go:18 | import 명령어 |

#### 명령어 실행 함수

| 함수 | 파일 | 설명 |
|------|------|------|
| `runInit(cmd, args)` | init.go:47 | init 명령어 실행 |
| `runDashboard(cmd, args)` | dashboard.go:32 | dashboard 실행 |
| `runMyRole(cmd, args)` | my_role.go:34 | my-role 실행 |
| `runPolicyPath(cmd, args)` | policy.go:49 | policy path 실행 |
| `runPolicyValidate(cmd, args)` | policy.go:92 | policy validate 실행 |
| `runValidate(cmd, args)` | validate.go:55 | validate 실행 |
| `runConvert(cmd, args)` | convert.go:44 | convert 실행 |
| `runLLMStatus(cmd, args)` | llm.go:61 | llm status 실행 |
| `runLLMTest(cmd, args)` | llm.go:107 | llm test 실행 |
| `runLLMSetup(cmd, args)` | llm.go:142 | llm setup 실행 |
| `runMCP(cmd, args)` | mcp.go:37 | mcp 실행 |
| `runCategoryList(cmd, args)` | category.go:138 | category list 실행 |
| `runCategoryAdd(cmd, args)` | category.go:165 | category add 실행 |
| `runCategoryEdit(cmd, args)` | category.go:240 | category edit 실행 |
| `runCategoryRemove(cmd, args)` | category.go:353 | category remove 실행 |
| `runConventionList(cmd, args)` | convention.go:179 | convention list 실행 |
| `runConventionAdd(cmd, args)` | convention.go:215 | convention add 실행 |
| `runConventionEdit(cmd, args)` | convention.go:295 | convention edit 실행 |
| `runConventionRemove(cmd, args)` | convention.go:414 | convention remove 실행 |
| `runImport(cmd, args)` | import.go:50 | import 실행 |

#### 헬퍼 함수 - 초기화

| 함수 | 파일 | 설명 |
|------|------|------|
| `createDefaultPolicy()` | init.go:135 | 기본 정책 파일 생성 |
| `initializeConfigFile()` | init.go:186 | config.json 초기화 |
| `removeExistingCodePolicy()` | init.go:201 | 생성된 파일 정리 (--force) |

#### 헬퍼 함수 - LLM

| 함수 | 파일 | 설명 |
|------|------|------|
| `promptLLMBackendSetup()` | llm.go:198 | LLM 프로바이더 설정 프롬프트 |
| `promptAndSaveAPIKey(provider)` | llm.go:276 | API 키 입력 및 저장 |
| `ensureGitignore(path)` | llm.go:318 | .gitignore 업데이트 |

#### 헬퍼 함수 - MCP 등록

| 함수 | 파일 | 설명 |
|------|------|------|
| `promptMCPRegistration()` | mcp_register.go:64 | MCP 등록 프롬프트 |
| `registerMCP(app)` | mcp_register.go:135 | MCP 서버 등록 |
| `getMCPConfigPath(app)` | mcp_register.go:257 | MCP 설정 파일 경로 |
| `getAppDisplayName(app)` | mcp_register.go:274 | 앱 표시명 반환 |
| `checkNpxAvailable()` | mcp_register.go:288 | npx 설치 여부 확인 |
| `updateSymphonySection(content)` | mcp_register.go:294 | Symphony 섹션 업데이트 |
| `createInstructionsFile(app)` | mcp_register.go:325 | 지시 파일 생성 |
| `getClaudeCodeInstructions()` | mcp_register.go:395 | Claude Code 지시 반환 |
| `getCursorInstructions()` | mcp_register.go:429 | Cursor 지시 반환 |
| `getVSCodeInstructions()` | mcp_register.go:467 | VS Code 지시 반환 |

#### 헬퍼 함수 - 역할

| 함수 | 파일 | 설명 |
|------|------|------|
| `selectNewRole()` | my_role.go:81 | 새 역할 선택 |

#### 헬퍼 함수 - 출력 포맷

| 함수 | 파일 | 설명 |
|------|------|------|
| `printValidationResult(result)` | validate.go:131 | 검증 결과 출력 |
| `runNewConverter(policy)` | convert.go:74 | 새 컨버터 실행 |
| `printImportResults(result)` | import.go:108 | Import 결과 출력 |

#### 터미널 포맷팅 (colors.go)

| 함수 | 파일 | 설명 |
|------|------|------|
| `isTTY()` | colors.go:21 | TTY 여부 확인 |
| `colorize(color, msg)` | colors.go:26 | 색상 적용 |
| `ok(msg)` | colors.go:34 | [OK] 접두어 포맷 |
| `formatError(msg)` | colors.go:40 | [ERROR] 접두어 포맷 |
| `warn(msg)` | colors.go:46 | [WARN] 접두어 포맷 |
| `titleWithDesc(title, desc)` | colors.go:52 | 섹션 제목 포맷 |
| `done(msg)` | colors.go:58 | [DONE] 접두어 포맷 |
| `printOK(msg)` | colors.go:64 | OK 메시지 출력 |
| `printError(msg)` | colors.go:69 | 에러 메시지 출력 |
| `printWarn(msg)` | colors.go:74 | 경고 메시지 출력 |
| `printTitle(title, desc)` | colors.go:79 | 제목 출력 |
| `printDone(msg)` | colors.go:84 | 완료 메시지 출력 |
| `indent(msg)` | colors.go:89 | 들여쓰기 |

#### Survey 템플릿 (survey_templates.go)

| 함수 | 파일 | 설명 |
|------|------|------|
| `useSelectTemplateNoFilter()` | survey_templates.go:53 | 필터 없는 Select 템플릿 적용 |
| `useMultiSelectTemplateNoFilter()` | survey_templates.go:64 | 필터 없는 MultiSelect 템플릿 적용 |

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `MCPRegistrationConfig` | mcp_register.go:22 | MCP 설정 구조 (mcpServers 포맷) |
| `VSCodeMCPConfig` | mcp_register.go:27 | VS Code MCP 설정 구조 |
| `MCPServerConfig` | mcp_register.go:34 | MCP 서버 설정 |
| `VSCodeServerConfig` | mcp_register.go:42 | VS Code 서버 설정 |

## 참고 문헌

- [명령어 레퍼런스](/docs/COMMAND.md) - 상세 CLI 사용법, 플래그, 예시
