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
      - [query\_conventions](#query_conventions)
      - [validate\_code](#validate_code)
      - [list\_category](#list_category)
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
├── category                # 카테고리 목록 조회
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

### sym category

**설명**: 사용 가능한 모든 컨벤션 카테고리와 설명을 표시합니다.

user-policy.json에 정의된 카테고리를 표시합니다. `sym init` 실행 시 7개의 기본 카테고리(security, style, documentation, error_handling, architecture, performance, testing)가 생성됩니다. 사용자는 이 카테고리를 수정, 삭제하거나 새로운 카테고리를 추가할 수 있습니다.

**문법**:
```
sym category
```

**예시**:
```bash
# 카테고리 목록 조회
sym category
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

  • error_handling
    Error handling and exception management rules

  • architecture
    Code structure and architecture rules

  • performance
    Performance optimization rules

  • testing
    Testing rules (coverage, test patterns, etc.)
```

**사용자 정의 카테고리**:

user-policy.json에 `category` 필드를 추가하여 사용자 정의 카테고리를 추가하거나 기존 카테고리 설명을 변경할 수 있습니다:

```json
{
  "version": "1.0",
  "category": [
    {"name": "security", "description": "보안 관련 규칙 (인증, 인가, 취약점 방지 등)"},
    {"name": "naming", "description": "네이밍 컨벤션 규칙 (변수, 함수, 클래스 등)"}
  ],
  "rules": [...]
}
```

**관련 파일**: `internal/cmd/category.go`

---

### sym mcp

**설명**: MCP(Model Context Protocol) 서버를 시작합니다. LLM 기반 코딩 도구가 stdio를 통해 컨벤션을 쿼리하고 코드를 검증할 수 있습니다.

**제공되는 MCP 도구**:
- `query_conventions`: 주어진 컨텍스트에 대한 컨벤션 쿼리
- `validate_code`: 코드의 컨벤션 준수 여부 검증
- `list_category`: 사용 가능한 카테고리 목록 조회

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

#### query_conventions

코딩 전 프로젝트 컨벤션을 쿼리합니다.

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

Use query_conventions with a specific category to get rules for that category.
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

