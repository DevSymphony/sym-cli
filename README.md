# Symphony

**LLM-Friendly Convention Linter for AI Coding Tools**

Symphony는 AI 개발환경(IDE, MCP 기반 LLM Tooling)을 위한 정책 기반 코드 컨벤션 검사기입니다.
간단한 설정만으로 프로젝트 규칙을 일관되게 적용하고, LLM 코드 생성 품질을 극대화할 수 있습니다.

---

## 목차

- [Symphony](#symphony)
  - [목차](#목차)
  - [주요 기능](#주요-기능)
  - [빠른 시작](#빠른-시작)
  - [MCP 설정](#mcp-설정)
  - [사용 가능한 MCP 도구](#사용-가능한-mcp-도구)
    - [`query_conventions`](#query_conventions)
    - [`validate_code`](#validate_code)
    - [`list_category`](#list_category)
    - [`add_category`](#add_category)
    - [`edit_category`](#edit_category)
    - [`remove_category`](#remove_category)
    - [`import_convention`](#import_convention)
  - [컨벤션 파일](#컨벤션-파일)
  - [요구사항](#요구사항)
  - [지원 플랫폼](#지원-플랫폼)
  - [라이선스](#라이선스)

---

## 주요 기능

- 자연어로 컨벤션 정의
- LLM이 MCP를 통해 필요한 컨벤션만 추출하여 컨텍스트에 포함
- LLM이 MCP를 통해 코드 변경사항에 대한 컨벤션 준수 여부를 검사
- 카테고리 기반 규칙 분류 및 관리
- RBAC 기반 접근 제어

---

## 빠른 시작

```bash
# 1. CLI 설치
npm install -g @dev-symphony/sym

# 2. 프로젝트 초기화 (.sym/ 폴더 생성 + MCP 설정)
sym init

# 3. 대시보드 실행 및 컨벤션 편집
sym dashboard

# 4. MCP 서버를 LLM IDE 내부에서 사용
```

---

## MCP 설정

`sym init` 명령은 MCP 서버 구성을 자동으로 설정합니다.
만약 수동으로 설정하고 싶다면 아래를 `~/.config/.../config.json` 등에 추가하세요.

```json
{
  "mcpServers": {
    "symphony": {
      "command": "npx",
      "args": ["-y", "@dev-symphony/sym@latest", "mcp"]
    }
  }
}
```

---

## 사용 가능한 MCP 도구

### `query_conventions`

- 프로젝트 컨벤션을 조회합니다.
- 카테고리, 파일 목록, 언어 등의 파라미터는 모두 optional입니다.

### `validate_code`

- 코드가 정의된 규칙을 따르는지 검사합니다.
- 필수 파라미터: `files`

### `list_category`

- 프로젝트에 정의된 카테고리 목록을 조회합니다.
- 파라미터 없음

### `add_category`

- 새 카테고리를 추가합니다 (배치 지원).
- 필수 파라미터: `categories` (배열)

### `edit_category`

- 기존 카테고리를 편집합니다 (배치 지원).
- 필수 파라미터: `edits` (배열)

### `remove_category`

- 카테고리를 삭제합니다 (배치 지원).
- 필수 파라미터: `names` (배열)

### `import_convention`

- 외부 문서(텍스트, 마크다운, 코드 파일)에서 컨벤션을 추출합니다.
- LLM을 사용하여 코딩 규칙을 자동으로 인식하고 정책에 추가합니다.
- 필수 파라미터: `path`
- 선택 파라미터: `mode` (`append` 또는 `clear`, 기본값: `append`)

---

## 컨벤션 파일

Symphony는 프로젝트 컨벤션을 **정책 파일(`.sym/user-policy.json`)**로 관리합니다.
아래 명령으로 대시보드를 열어 쉽게 편집할 수 있습니다.

```bash
sym dashboard
```

예시 정책 파일:

```json
{
  "version": "1.0.0",
  "rules": [
    {
      "say": "Functions should be documented",
      "category": "documentation"
    },
    {
      "say": "Lines should be less than 100 characters",
      "category": "formatting",
      "params": { "max": 100 }
    }
  ]
}
```

---

## 요구사항

- Node.js >= 16.0.0
- Policy file: `.sym/user-policy.json`

---

## 지원 플랫폼

- macOS (Intel, Apple Silicon)
- Linux (x64, ARM64)
- Windows (x64)

---

## 라이선스

MIT
