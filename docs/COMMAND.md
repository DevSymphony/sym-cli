# Symphony CLI 명령어 레퍼런스

Symphony (`sym`)는 코드 컨벤션 관리와 RBAC(역할 기반 접근 제어)를 위한 CLI 도구입니다. LLM 프로바이더와 통합하여 자연어 정책을 린터 설정으로 변환하고, 코드 변경사항을 검증합니다.

---

## 목차

- [Symphony CLI 명령어 레퍼런스](#symphony-cli-명령어-레퍼런스)
  - [목차](#목차)
  - [개요](#개요)
  - [빌드 및 설치](#빌드-및-설치)
  - [전역 플래그](#전역-플래그)
  - [명령어 계층 구조](#명령어-계층-구조)
  - [명령어 상세](#명령어-상세)
    - [sym init](#sym-init)
    - [sym dashboard](#sym-dashboard)
    - [sym my-role](#sym-my-role)
    - [sym policy](#sym-policy)
      - [sym policy path](#sym-policy-path)
      - [sym policy validate](#sym-policy-validate)
    - [sym convert](#sym-convert)
    - [sym validate](#sym-validate)
    - [sym import](#sym-import)
    - [sym category](#sym-category)
    - [sym mcp](#sym-mcp)
    - [sym llm](#sym-llm)
      - [sym llm status](#sym-llm-status)
      - [sym llm test](#sym-llm-test)
      - [sym llm setup](#sym-llm-setup)
    - [sym version](#sym-version)
    - [sym completion](#sym-completion)
    - [sym help](#sym-help)
  - [설정 파일](#설정-파일)
    - [디렉토리 구조](#디렉토리-구조)
    - [config.json](#configjson)
    - [.env](#env)
    - [roles.json](#rolesjson)
  - [MCP 통합](#mcp-통합)
    - [지원 도구](#지원-도구)
    - [MCP 도구 스키마](#mcp-도구-스키마)
      - [list\_convention](#list_convention)
      - [validate\_code](#validate_code)
      - [list\_category](#list_category)
      - [add\_category](#add_category)
      - [edit\_category](#edit_category)
      - [remove\_category](#remove_category)
      - [import\_convention](#import_convention)
      - [add\_convention](#add_convention)
      - [edit\_convention](#edit_convention)
      - [remove\_convention](#remove_convention)
      - [convert](#convert)
    - [등록 방법](#등록-방법)
  - [LLM 프로바이더](#llm-프로바이더)
    - [지원 프로바이더](#지원-프로바이더)
    - [설정 방법](#설정-방법)
      - [Claude Code](#claude-code)
      - [Gemini CLI](#gemini-cli)
      - [OpenAI API](#openai-api)
    - [상태 확인](#상태-확인)

---

## 개요

Symphony CLI는 다음 기능을 제공합니다:

- **자연어 정책 정의**: 사용자가 자연어로 코딩 컨벤션을 정의
- **자동 변환**: 자연어 정책을 ESLint, Prettier, Pylint 등 린터 설정으로 변환
- **RBAC 지원**: 역할 기반 파일 접근 권한 관리
- **MCP 서버**: Claude Code, Cursor 등 AI 코딩 도구와 통합
- **LLM 검증**: 정적 린터로 검사할 수 없는 복잡한 규칙을 LLM으로 검증

---

## 빌드 및 설치

```bash
# 저장소 클론
git clone https://github.com/anthropics/symphony.git
cd symphony

# 빌드
go build -o bin/sym ./cmd/sym

# PATH에 추가 (선택사항)
export PATH=$PATH:$(pwd)/bin

# 또는 직접 설치
go install ./cmd/sym
```

---

## 전역 플래그

모든 명령어에서 사용 가능한 플래그입니다.

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--help` | `-h` | bool | `false` | 명령어 도움말 표시 |
| `--verbose` | `-v` | bool | `false` | 상세 출력 활성화 |

---

## 명령어 계층 구조

```
sym
├── init                    # 프로젝트 초기화
├── dashboard (dash)        # 웹 대시보드 실행
├── my-role                 # 역할 확인/변경
├── policy                  # 정책 관리
│   ├── path               # 정책 파일 경로 관리
│   └── validate           # 정책 파일 유효성 검사
├── convert                 # 정책 → 린터 설정 변환
├── validate                # Git 변경사항 검증
├── import                  # 외부 문서에서 컨벤션 추출
├── category                # 카테고리 관리
├── convention              # 컨벤션(규칙) 관리
├── mcp                     # MCP 서버 실행
├── llm                     # LLM 프로바이더 관리
│   ├── status             # 현재 설정 확인
│   ├── test               # 연결 테스트
│   └── setup              # 설정 안내
├── version                 # 버전 출력
├── completion              # 쉘 자동완성 스크립트 생성
└── help                    # 명령어 도움말
```

---

## 명령어 상세

### sym init

**설명**: 현재 디렉토리에 Symphony를 초기화합니다. `.sym` 디렉토리와 기본 설정 파일을 생성합니다.

**수행 작업**:
1. `.sym/roles.json` 생성 (기본 역할: admin, developer, viewer)
2. `.sym/user-policy.json` 생성 (기본 카테고리 7개 + RBAC 설정)
3. `.sym/config.json` 생성 (기본 설정)
4. 역할을 admin으로 설정 (대시보드에서 변경 가능)
5. MCP 서버 등록 (선택적)
6. LLM 백엔드 설정 (선택적)

**동작 조건**:
- `--force` 없음: `.sym` 디렉토리가 존재하지 않을 때만 동작
- `--force` 있음: 기존 설정을 덮어쓰고, 생성된 파일(code-policy.json, 린터 설정)을 정리

**문법**:
```
sym init [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--force` | `-f` | bool | `false` | 기존 Symphony 설정 덮어쓰기 |
| `--skip-mcp` | - | bool | `false` | MCP 서버 등록 프롬프트 건너뛰기 |
| `--skip-llm` | - | bool | `false` | LLM 백엔드 설정 프롬프트 건너뛰기 |

**예시**:
```bash
# 전체 초기화
sym init

# 기존 설정 덮어쓰기 (재초기화)
sym init --force

# MCP 등록 없이 초기화
sym init --skip-mcp

# LLM 설정 없이 초기화
sym init --skip-llm
```

**관련 파일**: `internal/cmd/init.go`

---

### sym dashboard

**설명**: 역할과 정책을 관리하기 위한 로컬 웹 서버를 시작합니다.

**별칭**: `dash`

**기능**:
- 역할 선택
- 역할 권한 관리
- 코딩 정책 및 규칙 편집

**문법**:
```
sym dashboard [flags]
sym dash [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--port` | `-p` | int | `8787` | 대시보드 실행 포트 |

**예시**:
```bash
# 기본 포트(8787)에서 시작
sym dashboard

# 사용자 지정 포트에서 시작
sym dashboard --port 3000

# 별칭 사용
sym dash
```

**관련 파일**: `internal/cmd/dashboard.go`

---

### sym my-role

**설명**: 현재 선택된 역할을 확인하거나 변경합니다.

**문법**:
```
sym my-role [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--json` | - | bool | `false` | JSON 형식으로 출력 (스크립팅용) |
| `--select` | - | bool | `false` | 대화형으로 새 역할 선택 |

**예시**:
```bash
# 현재 역할 확인
sym my-role

# JSON으로 출력
sym my-role --json

# 역할 변경 (대화형)
sym my-role --select
```

**관련 파일**: `internal/cmd/my_role.go`

---

### sym policy

**설명**: 코딩 컨벤션 및 정책 설정을 관리하는 상위 명령어입니다.

**문법**:
```
sym policy <subcommand> [flags]
```

#### sym policy path

**설명**: 정책 파일 경로를 확인하거나 설정합니다.

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--set` | - | string | `""` | 새 정책 파일 경로 설정 |

**예시**:
```bash
# 현재 정책 경로 확인
sym policy path

# 새 정책 경로 설정
sym policy path --set ./custom/my-policy.json
```

#### sym policy validate

**설명**: 정책 파일의 구문 및 구조를 검증합니다.

**예시**:
```bash
sym policy validate
```

**관련 파일**: `internal/cmd/policy.go`

---

### sym convert

**설명**: 사용자가 작성한 자연어 정책(Schema A)을 린터별 설정 파일과 내부 검증 스키마(Schema B)로 변환합니다.

LLM을 사용하여 자연어 규칙을 분석하고, 언어 기반 라우팅으로 적절한 린터 규칙에 매핑합니다.

**지원 린터**:
- ESLint
- Prettier
- Pylint
- TSC (TypeScript Compiler)
- Checkstyle (Java)
- PMD (Java)

**문법**:
```
sym convert [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--input` | `-i` | string | `""` | 입력 사용자 정책 파일 (기본값: .sym/config.json의 policy_path) |
| `--output-dir` | `-o` | string | `""` | 린터 설정 출력 디렉토리 (기본값: .sym) |

**예시**:
```bash
# 정책 변환 (출력: .sym 디렉토리)
sym convert -i user-policy.json

# 사용자 지정 출력 디렉토리
sym convert -i user-policy.json -o ./custom-dir
```

**출력 파일**:
- `.sym/code-policy.json` - 변환된 정책 (Schema B)
- `.sym/.eslintrc.json` - ESLint 설정
- `.sym/.prettierrc.json` - Prettier 설정
- `.sym/.pylintrc` - Pylint 설정
- 등

**관련 파일**: `internal/cmd/convert.go`

---

### sym validate

**설명**: Git 변경사항을 코딩 컨벤션에 대해 LLM으로 검증합니다.

code-policy.json에서 `llm-validator`를 엔진으로 사용하는 규칙들을 검사합니다. 이는 일반적으로 정적 린터로 검사할 수 없는 복잡한 규칙(보안, 아키텍처 등)입니다.

**기본 동작**: 모든 커밋되지 않은 변경사항 검증
- 스테이지된 변경사항 (git add)
- 스테이지되지 않은 변경사항 (수정됨)
- 추적되지 않은 파일 (새 파일)

**문법**:
```
sym validate [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--policy` | `-p` | string | `""` | code-policy.json 경로 (기본값: .sym/code-policy.json) |
| `--staged` | - | bool | `false` | 스테이지된 변경사항만 검증 (기본값: 모든 커밋되지 않은 변경사항) |
| `--timeout` | - | int | `30` | 규칙당 검사 타임아웃 (초) |

**예시**:
```bash
# 모든 커밋되지 않은 변경사항 검증 (기본)
sym validate

# 스테이지된 변경사항만 검증
sym validate --staged

# 사용자 지정 정책 파일 사용
sym validate --policy custom-policy.json

# 타임아웃 설정
sym validate --timeout 60
```

**관련 파일**: `internal/cmd/validate.go`

---

### sym import

**설명**: 외부 문서에서 코딩 컨벤션을 추출하여 user-policy.json에 추가합니다.

LLM을 사용하여 텍스트, 마크다운, 코드 파일 등에서 코딩 규칙을 자동으로 인식하고 Symphony 정책 형식으로 변환합니다.

**지원 포맷**:
- 텍스트 문서: `.txt`, `.md`, `.markdown`
- 코드 파일: `.go`, `.js`, `.ts`, `.jsx`, `.tsx`, `.py`, `.java`, `.rs`, `.rb`, `.php`, `.c`, `.cpp`, `.h`, `.hpp`, `.cs`, `.swift`, `.kt`, `.scala`
- 설정/데이터: `.yaml`, `.yml`, `.json`, `.toml`, `.xml`
- 웹 파일: `.html`, `.htm`, `.css`, `.scss`, `.less`
- 기타: `.rst`, `.adoc`

**파일 크기 제한**: 50KB

**문법**:
```
sym import <file> [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--mode` | `-m` | string | `append` | Import 모드: `append` (기존 유지, 새 항목 추가) 또는 `clear` (기존 삭제 후 추가) |

**Import 모드**:
- `append` (기본값): 기존 카테고리와 규칙을 유지하고 새 항목을 추가합니다. 중복 카테고리는 건너뛰고, 중복 규칙 ID는 접미어를 추가합니다 (예: `SEC-001-2`).
- `clear`: 기존 모든 카테고리와 규칙을 삭제한 후 새 항목을 추가합니다. 사용자 확인이 필요합니다.

**예시**:
```bash
# 마크다운 문서에서 컨벤션 추출 (append 모드, 기본값)
sym import coding-standards.md

# 텍스트 파일에서 컨벤션 추출
sym import team-guidelines.txt

# 기존 컨벤션 삭제 후 새로 추가 (clear 모드)
sym import new-rules.md --mode clear
```

**Import 프로세스**:
1. 파일 읽기 및 형식 검증
2. LLM을 사용하여 코딩 컨벤션 추출
3. 카테고리와 규칙에 고유 ID 생성
4. 기존 user-policy.json과 병합
5. 정책 파일 저장

**출력 예시**:
```
[Import Conventions] Processing: coding-standards.md
Mode: append

[OK] Processed: /path/to/coding-standards.md

[OK] Added 2 categories:
    • security: Security rules for safe coding
    • performance: Performance optimization guidelines

[OK] Added 5 rules:
    • [SEC-001] Use parameterized queries for database operations (security)
    • [SEC-002] Sanitize all user inputs before processing (security)
    • [PERF-001] Avoid N+1 queries in database operations (performance)
    • [PERF-002] Use pagination for large data sets (performance)
    • [PERF-003] Cache frequently accessed data (performance)

[DONE] Import complete
```

**관련 파일**: `internal/cmd/import.go`

---

### sym category

**설명**: 컨벤션 카테고리를 관리합니다.

user-policy.json에 정의된 카테고리를 조회, 추가, 편집, 삭제할 수 있습니다. `sym init` 실행 시 7개의 기본 카테고리(security, style, documentation, error_handling, architecture, performance, testing)가 생성됩니다.

**서브커맨드**:
- `list` - 카테고리 목록 조회
- `add` - 새 카테고리 추가
- `edit` - 기존 카테고리 편집
- `remove` - 카테고리 삭제

**관련 파일**: `internal/cmd/category.go`

---

#### sym category list

**설명**: 모든 카테고리와 설명을 표시합니다.

**문법**:
```
sym category list
```

**출력 예시**:
```
[Convention Categories] 7 categories available

  • security
    Security rules (authentication, authorization, vulnerability prevention, etc.)

  • style
    Code style and formatting rules

  • documentation
    Documentation rules (comments, docstrings, etc.)
```

---

#### sym category add

**설명**: 새 카테고리를 추가합니다.

**문법**:
```
sym category add <name> <description>
sym category add -f <file.json>
```

**플래그**:
- `-f, --file` - 배치 추가를 위한 JSON 파일

**예시**:
```bash
# 단일 추가
sym category add accessibility "Accessibility rules (WCAG, ARIA, etc.)"

# 배치 추가
sym category add -f categories.json
```

**배치 파일 형식** (`categories.json`):
```json
[
  {"name": "security", "description": "Security rules"},
  {"name": "performance", "description": "Performance rules"}
]
```

---

#### sym category edit

**설명**: 기존 카테고리의 이름 또는 설명을 변경합니다.

**문법**:
```
sym category edit <name> [--name <new-name>] [--description <desc>]
sym category edit -f <file.json>
```

**플래그**:
- `--name` - 새 카테고리 이름
- `--description` - 새 설명
- `-f, --file` - 배치 편집을 위한 JSON 파일

**예시**:
```bash
# 설명만 변경
sym category edit security --description "Updated security rules"

# 이름 변경 (관련 규칙도 자동 업데이트)
sym category edit old-name --name new-name

# 이름과 설명 모두 변경
sym category edit security --name sec --description "Security conventions"

# 배치 편집
sym category edit -f edits.json
```

**배치 파일 형식** (`edits.json`):
```json
[
  {"name": "security", "new_name": "sec"},
  {"name": "performance", "description": "New description"}
]
```

**참고**: 카테고리 이름 변경 시 해당 카테고리를 참조하는 모든 규칙이 자동으로 업데이트됩니다.

---

#### sym category remove

**설명**: 카테고리를 삭제합니다.

**문법**:
```
sym category remove <name> [names...]
sym category remove -f <file.json>
```

**플래그**:
- `-f, --file` - 삭제할 카테고리 이름이 담긴 JSON 파일

**예시**:
```bash
# 단일 삭제
sym category remove deprecated-category

# 다중 삭제
sym category remove cat1 cat2 cat3

# 배치 삭제
sym category remove -f names.json
```

**배치 파일 형식** (`names.json`):
```json
["cat1", "cat2", "cat3"]
```

**참고**: 규칙이 참조하고 있는 카테고리는 삭제할 수 없습니다. 먼저 해당 규칙을 삭제하거나 다른 카테고리로 변경해야 합니다.

---

### sym convention

**설명**: 컨벤션(규칙)을 관리합니다.

user-policy.json에 정의된 규칙을 조회, 추가, 편집, 삭제할 수 있습니다. 규칙은 코딩 컨벤션의 구체적인 내용을 정의합니다.

**서브커맨드**:
- `list` - 컨벤션 목록 조회
- `add` - 새 컨벤션 추가
- `edit` - 기존 컨벤션 편집
- `remove` - 컨벤션 삭제

**관련 파일**: `internal/cmd/convention.go`

---

#### sym convention list

**설명**: 모든 컨벤션을 표시합니다.

**문법**:
```
sym convention list [--category <name>] [--language <lang>]
```

**플래그**:
- `--category`, `-c` - 특정 카테고리로 필터링
- `--language`, `-l` - 특정 언어로 필터링

**출력 예시**:
```
[Conventions] 5 rules available

  • [SEC-001] security (error)
    Use parameterized queries for database operations
    languages: go, python

  • [PERF-001] performance (warning)
    Avoid N+1 queries in database operations
```

---

#### sym convention add

**설명**: 새 컨벤션을 추가합니다.

**문법**:
```
sym convention add <id> <say> [flags]
sym convention add -f <file.json>
```

**플래그**:
- `-f, --file` - 배치 추가를 위한 JSON 파일
- `--category` - 카테고리 (선택)
- `--languages` - 언어 목록 (쉼표로 구분)
- `--severity` - 심각도: error, warning, info (기본값: warning)
- `--autofix` - 자동 수정 활성화
- `--message` - 위반 시 표시할 메시지
- `--example` - 코드 예시
- `--include` - 포함할 파일 패턴 (쉼표로 구분)
- `--exclude` - 제외할 파일 패턴 (쉼표로 구분)

**예시**:
```bash
# 단일 추가
sym convention add SEC-001 "Use parameterized queries" --category security --languages go,python --severity error

# 배치 추가
sym convention add -f conventions.json
```

**배치 파일 형식** (`conventions.json`):
```json
[
  {
    "id": "SEC-001",
    "say": "Use parameterized queries",
    "category": "security",
    "languages": ["go", "python"],
    "severity": "error"
  }
]
```

**참고**: 컨벤션에 포함된 언어는 자동으로 `defaults.languages`에 추가됩니다.

---

#### sym convention edit

**설명**: 기존 컨벤션을 편집합니다.

**문법**:
```
sym convention edit <id> [flags]
sym convention edit -f <file.json>
```

**플래그**:
- `-f, --file` - 배치 편집을 위한 JSON 파일
- `--new-id` - 새 ID
- `--say` - 새 설명
- `--category` - 새 카테고리
- `--languages` - 새 언어 목록
- `--severity` - 새 심각도
- `--autofix` - 자동 수정 활성화/비활성화
- `--message` - 새 메시지
- `--example` - 새 예시
- `--include` - 새 포함 패턴
- `--exclude` - 새 제외 패턴

**예시**:
```bash
# 심각도 변경
sym convention edit SEC-001 --severity warning

# ID 변경
sym convention edit SEC-001 --new-id SEC-001-v2

# 배치 편집
sym convention edit -f edits.json
```

**배치 파일 형식** (`edits.json`):
```json
[
  {"id": "SEC-001", "severity": "warning"},
  {"id": "PERF-001", "new_id": "PERF-001-v2"}
]
```

---

#### sym convention remove

**설명**: 컨벤션을 삭제합니다.

**문법**:
```
sym convention remove <id> [ids...]
sym convention remove -f <file.json>
```

**플래그**:
- `-f, --file` - 삭제할 컨벤션 ID가 담긴 JSON 파일

**예시**:
```bash
# 단일 삭제
sym convention remove SEC-001

# 다중 삭제
sym convention remove SEC-001 PERF-001 PERF-002

# 배치 삭제
sym convention remove -f ids.json
```

**배치 파일 형식** (`ids.json`):
```json
["SEC-001", "PERF-001", "PERF-002"]
```

---

### sym mcp

**설명**: MCP(Model Context Protocol) 서버를 시작합니다. LLM 기반 코딩 도구가 stdio를 통해 컨벤션을 쿼리하고 코드를 검증할 수 있습니다.

**제공되는 MCP 도구**:
- `list_convention`: 프로젝트 컨벤션 조회
- `validate_code`: 코드의 컨벤션 준수 여부 검증
- `list_category`: 사용 가능한 카테고리 목록 조회
- `add_category`: 카테고리 추가 (배치 지원)
- `edit_category`: 카테고리 편집 (배치 지원)
- `remove_category`: 카테고리 삭제 (배치 지원)
- `import_convention`: 외부 문서에서 컨벤션 추출
- `add_convention`: 컨벤션(규칙) 추가 (배치 지원)
- `edit_convention`: 컨벤션(규칙) 편집 (배치 지원)
- `remove_convention`: 컨벤션(규칙) 삭제 (배치 지원)
- `convert`: user-policy.json → code-policy.json + 린터 설정 생성/갱신 (권장: 규칙/카테고리 변경 후 실행)

**통신 방식**: stdio (Claude Desktop, Claude Code, Cursor 등 MCP 클라이언트와 통합)

**문법**:
```
sym mcp [flags]
```

**플래그**:

| 플래그 | 단축 | 타입 | 기본값 | 설명 |
|--------|------|------|--------|------|
| `--config` | `-c` | string | `""` | 정책 파일 경로 (code-policy.json) |

**예시**:
```bash
# 자동 감지로 MCP 서버 시작
sym mcp

# 특정 설정 파일로 시작
sym mcp --config code-policy.json
```

**관련 파일**: `internal/cmd/mcp.go`, `internal/mcp/server.go`

---

### sym llm

**설명**: LLM 프로바이더 설정을 관리하는 상위 명령어입니다.

**지원 프로바이더**:
- `claudecode`: Claude Code CLI ('claude'가 PATH에 필요)
- `geminicli`: Gemini CLI ('gemini'가 PATH에 필요)
- `openaiapi`: OpenAI API (OPENAI_API_KEY 필요)

**문법**:
```
sym llm <subcommand> [flags]
```

#### sym llm status

**설명**: 현재 LLM 프로바이더 설정과 가용성을 표시합니다.

**예시**:
```bash
sym llm status
```

#### sym llm test

**설명**: LLM 프로바이더가 정상 작동하는지 테스트 요청을 보냅니다.

**예시**:
```bash
sym llm test
```

#### sym llm setup

**설명**: LLM 프로바이더 설정 방법에 대한 안내를 표시합니다.

**예시**:
```bash
sym llm setup
```

**관련 파일**: `internal/cmd/llm.go`

---

### sym version

**설명**: sym CLI의 버전 번호를 출력합니다.

**문법**:
```
sym version
```

**예시**:
```bash
sym version
```

**관련 파일**: `internal/cmd/version.go`

---

### sym completion

**설명**: 지정된 쉘에 대한 자동완성 스크립트를 생성합니다.

**지원 쉘**:
- bash
- zsh
- fish
- powershell

**문법**:
```
sym completion <shell>
```

**예시**:
```bash
# Bash 자동완성 설정
sym completion bash > /etc/bash_completion.d/sym

# Zsh 자동완성 설정
sym completion zsh > "${fpath[1]}/_sym"

# Fish 자동완성 설정
sym completion fish > ~/.config/fish/completions/sym.fish

# PowerShell 자동완성 설정
sym completion powershell > sym.ps1
```

---

### sym help

**설명**: 명령어에 대한 도움말을 표시합니다.

**문법**:
```
sym help [command]
```

**예시**:
```bash
# 전체 도움말
sym help

# 특정 명령어 도움말
sym help init
sym help convert
```

---

## 설정 파일

### 디렉토리 구조

```
.sym/
├── config.json           # 프로젝트 설정 (LLM, MCP, 정책 경로)
├── .env                  # API 키 (gitignored)
├── roles.json            # 역할 정의
├── user-policy.json      # 자연어 정책 (Schema A)
├── code-policy.json      # 변환된 정책 (Schema B)
└── validation-results.json  # 검증 이력 (최근 50개)
```

### config.json

프로젝트별 설정 파일입니다.

```json
{
  "llm": {
    "provider": "claudecode",
    "model": "sonnet"
  },
  "mcp": {
    "tools": ["vscode", "claude-code", "cursor"]
  },
  "policy_path": ".sym/user-policy.json"
}
```

### .env

API 키를 저장합니다 (gitignored).

```
OPENAI_API_KEY=sk-...
```

### roles.json

역할 정의 파일입니다.

```json
{
  "roles": {
    "admin": {
      "permissions": [
        {"path": "**", "read": true, "write": true, "execute": true}
      ]
    },
    "developer": {
      "permissions": [
        {"path": "src/**", "read": true, "write": true, "execute": false},
        {"path": "config/**", "read": true, "write": false, "execute": false}
      ]
    },
    "viewer": {
      "permissions": [
        {"path": "**", "read": true, "write": false, "execute": false}
      ]
    }
  }
}
```

---

## MCP 통합

Symphony는 다음 AI 코딩 도구에 MCP 서버로 등록될 수 있습니다.

### 지원 도구

| 도구 | 설정 위치 | 형식 |
|------|----------|------|
| Claude Code | `.mcp.json` (프로젝트 루트) | mcpServers |
| Cursor | `.cursor/mcp.json` | mcpServers (type: "stdio") |
| VS Code Copilot | `.vscode/mcp.json` | servers |
| Claude Desktop | 전역 설정 (OS별) | mcpServers |
| Cline | 전역 설정 (VS Code 확장 저장소) | mcpServers |

### MCP 도구 스키마

#### list_convention

코딩 전 프로젝트 컨벤션을 조회합니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `category` | string | 아니오 | 카테고리 필터. 'all' 또는 비워두면 모든 카테고리. 옵션: security, style, documentation, error_handling, architecture, performance, testing |
| `languages` | []string | 아니오 | 언어 필터. 비워두면 모든 언어. 예: go, javascript, typescript, python, java |

#### validate_code

Git 변경사항을 프로젝트 컨벤션에 대해 검증합니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `role` | string | 아니오 | 검증용 RBAC 역할 (선택) |

#### list_category

사용 가능한 모든 카테고리와 설명을 반환합니다.

user-policy.json에 정의된 카테고리를 반환합니다. 카테고리가 없으면 `sym init`을 실행하라는 안내 메시지를 반환합니다.

**입력 스키마**:

파라미터 없음 (모든 카테고리 반환)

**출력 예시**:
```
Available categories (7):

• security
  Security rules (authentication, authorization, vulnerability prevention, etc.)

• style
  Code style and formatting rules

• documentation
  Documentation rules (comments, docstrings, etc.)

• error_handling
  Error handling and exception management rules

• architecture
  Code structure and architecture rules

• performance
  Performance optimization rules

• testing
  Testing rules (coverage, test patterns, etc.)

Use list_convention with a specific category to get rules for that category.
```

#### add_category

카테고리를 추가합니다 (배치 모드).

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `categories` | array | 예 | `{name, description}` 객체 배열 |

**예시**:
```json
{
  "categories": [
    {"name": "accessibility", "description": "Accessibility rules (WCAG, ARIA)"},
    {"name": "i18n", "description": "Internationalization rules"}
  ]
}
```

#### edit_category

카테고리를 편집합니다 (배치 모드). 이름 변경 시 규칙 참조도 자동 업데이트됩니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `edits` | array | 예 | `{name, new_name?, description?}` 객체 배열 |

**예시**:
```json
{
  "edits": [
    {"name": "security", "description": "Updated security rules"},
    {"name": "style", "new_name": "formatting"}
  ]
}
```

#### remove_category

카테고리를 삭제합니다 (배치 모드). 규칙이 참조하는 카테고리는 삭제할 수 없습니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `names` | array | 예 | 삭제할 카테고리 이름 배열 |

**예시**:
```json
{
  "names": ["deprecated-category", "unused-category"]
}
```

#### import_convention

외부 문서에서 코딩 컨벤션을 추출하여 user-policy.json에 추가합니다.

LLM을 사용하여 텍스트, 마크다운, 코드 파일 등에서 코딩 규칙을 자동으로 인식하고 카테고리와 규칙을 생성합니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `path` | string | 예 | Import할 파일 경로 |
| `mode` | string | 아니오 | Import 모드: `append` (기본값, 기존 유지) 또는 `clear` (기존 삭제 후 추가) |

**예시**:
```json
{
  "path": "/path/to/coding-standards.md",
  "mode": "append"
}
```

**출력 예시**:
```
Import completed successfully.

Categories added (2):
• security: Security rules for safe coding
• performance: Performance optimization guidelines

Rules added (3):
• [SEC-001] Use parameterized queries for database operations (security)
• [PERF-001] Avoid N+1 queries in database operations (performance)
• [PERF-002] Use pagination for large data sets (performance)
```

#### add_convention

컨벤션(규칙)을 추가합니다 (배치 모드).

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `conventions` | array | 예 | `{id, say, category?, languages?, severity?, autofix?, message?, example?, include?, exclude?}` 객체 배열 |

**예시**:
```json
{
  "conventions": [
    {
      "id": "SEC-001",
      "say": "Use parameterized queries for database operations",
      "category": "security",
      "languages": ["go", "python"],
      "severity": "error"
    },
    {
      "id": "PERF-001",
      "say": "Avoid N+1 queries in database operations",
      "category": "performance"
    }
  ]
}
```

**참고**: 컨벤션에 포함된 언어는 자동으로 `defaults.languages`에 추가됩니다.

#### edit_convention

컨벤션(규칙)을 편집합니다 (배치 모드).

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `edits` | array | 예 | `{id, new_id?, say?, category?, languages?, severity?, autofix?, message?, example?, include?, exclude?}` 객체 배열 |

**예시**:
```json
{
  "edits": [
    {"id": "SEC-001", "severity": "warning"},
    {"id": "PERF-001", "new_id": "PERF-001-v2", "say": "Updated description"}
  ]
}
```

**참고**: 편집된 컨벤션에 새 언어가 추가되면 자동으로 `defaults.languages`에 추가됩니다.

#### remove_convention

컨벤션(규칙)을 삭제합니다 (배치 모드).

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `ids` | array | 예 | 삭제할 컨벤션 ID 배열 |

**예시**:
```json
{
  "ids": ["SEC-001", "PERF-001"]
}
```

#### convert

user-policy.json(Schema A)에서 code-policy.json(Schema B) 및 린터 설정 파일을 생성/갱신합니다.

**입력 스키마**:

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `input_path` | string | 아니오 | user-policy.json 경로 (기본값: config 또는 `.sym/user-policy.json`) |
| `output_dir` | string | 아니오 | 출력 디렉토리 (기본값: `.sym`) |

**예시**:
```json
{
  "input_path": ".sym/user-policy.json",
  "output_dir": ".sym"
}
```

### 등록 방법

```bash
# 초기화 시 대화형으로 MCP 등록 (권장)
sym init

# MCP 등록을 건너뛰려면
sym init --skip-mcp
```

---

## LLM 프로바이더

### 지원 프로바이더

| 프로바이더 ID | 표시 이름 | API 키 필요 | CLI 요구사항 |
|--------------|----------|-------------|-------------|
| `claudecode` | Claude Code | 아니오 | 'claude' PATH에 있어야 함 |
| `geminicli` | Gemini CLI | 아니오 | 'gemini' PATH에 있어야 함 |
| `openaiapi` | OpenAI API | 예 (OPENAI_API_KEY) | 없음 |

### 설정 방법

#### Claude Code

1. Claude Code CLI 설치:
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```

2. Claude Code 인증:
   ```bash
   claude auth login
   ```

3. Symphony 설정:
   ```json
   // .sym/config.json
   {
     "llm": {
       "provider": "claudecode",
       "model": "sonnet"
     }
   }
   ```

#### Gemini CLI

1. Gemini CLI 설치 (Google Cloud SDK 필요)

2. Symphony 설정:
   ```json
   // .sym/config.json
   {
     "llm": {
       "provider": "geminicli",
       "model": "gemini-pro"
     }
   }
   ```

#### OpenAI API

1. API 키 설정:
   ```bash
   # .sym/.env
   OPENAI_API_KEY=sk-...
   ```

2. Symphony 설정:
   ```json
   // .sym/config.json
   {
     "llm": {
       "provider": "openaiapi",
       "model": "gpt-4"
     }
   }
   ```

### 상태 확인

```bash
# 현재 설정 확인
sym llm status

# 연결 테스트
sym llm test

# 설정 안내 보기
sym llm setup
```

