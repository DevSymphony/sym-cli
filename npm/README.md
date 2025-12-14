# Symphony

**LLM-Friendly Convention Linter for AI Coding Tools**

Symphony는 AI 개발환경(IDE, MCP 기반 LLM Tooling)을 위한 정책 기반 코드 컨벤션 검사기입니다.
간단한 설정만으로 프로젝트 규칙을 일관되게 적용하고, LLM 코드 생성 품질을 향상시킬 수 있습니다.

---

## 목차

- [Symphony](#symphony)
  - [목차](#목차)
  - [주요 기능](#주요-기능)
  - [빠른 시작](#빠른-시작)
  - [컨벤션 관리](#컨벤션-관리)
  - [MCP 설정](#mcp-설정)
  - [사용 가능한 MCP 도구](#사용-가능한-mcp-도구)
    - [`list_convention`](#list_convention)
    - [`validate_code`](#validate_code)
    - [`list_category`](#list_category)
    - [`add_category`](#add_category)
    - [`edit_category`](#edit_category)
    - [`remove_category`](#remove_category)
    - [`import_convention`](#import_convention)
    - [`add_convention`](#add_convention)
    - [`edit_convention`](#edit_convention)
    - [`remove_convention`](#remove_convention)
    - [`convert`](#convert)
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

# 3. 컨벤션 관리
# LLM IDE를 통해 기존 문서나 자연어로 컨벤션을 생성, 관리합니다. '컨벤션 관리' 부분을 참고해 주세요.

# 4. 컨벤션 적용
# LLM IDE에서 작업하면, 작업 전 컨벤션을 자동으로 가져오고, 작업 후 컨벤션을 자동으로 검증합니다.
```

---

## 컨벤션 관리

컨벤션은 아래 3가지 방식으로 관리할 수 있습니다:

- **MCP 도구(권장)**: `list_*`, `add_*`, `edit_*`, `remove_*`, `import_convention`, `convert`
- **Dashboard**: `sym dash`로 웹에서 편집
- **CLI 명령어**: `sym category|convention|import|convert`

권장사항: **LLM IDE(Cursor/Claude Code 등)를 사용한다면 MCP 기반 관리**를 권장합니다.

예시 문장: "`docs/team-standards.md`를 컨벤션에 반영해줘."

자세한 내용은 [`docs/CONVENTION_MANAGEMENT.md`](docs/CONVENTION_MANAGEMENT.md)를 참고하세요.

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

### `list_convention`

- 프로젝트 컨벤션을 조회합니다.
- 카테고리, 언어 등의 파라미터는 모두 optional입니다.

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

### `add_convention`

- 새 컨벤션(규칙)을 추가합니다 (배치 지원).
- 필수 파라미터: `conventions` (배열)
- 컨벤션에 포함된 언어는 자동으로 `defaults.languages`에 추가됩니다.

### `edit_convention`

- 기존 컨벤션을 편집합니다 (배치 지원).
- 필수 파라미터: `edits` (배열)

### `remove_convention`

- 컨벤션을 삭제합니다 (배치 지원).
- 필수 파라미터: `ids` (배열)

### `convert`

- user-policy.json(Schema A)에서 code-policy.json(Schema B) 및 린터 설정 파일을 생성/갱신합니다.
- 컨벤션/카테고리를 추가/편집/삭제한 뒤 실행하는 것을 권장합니다.

---

## 요구사항

- Node.js >= 16.0.0

---

## 지원 플랫폼

- macOS (Intel, Apple Silicon)
- Linux (x64, ARM64)
- Windows (x64)

---

## 라이선스

MIT
