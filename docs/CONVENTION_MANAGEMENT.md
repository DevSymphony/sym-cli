# 컨벤션/카테고리 관리 가이드

이 문서는 Symphony에서 **컨벤션(규칙)** 및 **카테고리**를 관리하는 방법을 정리합니다.

- **MCP 방식**: Cursor/Claude Code 등 LLM 도구가 MCP 도구를 호출해 규칙/카테고리를 조회·변경
- **Dashboard 방식**: 로컬 웹 대시보드에서 직접 편집
- **CLI 방식**: `sym` command를 사용해 규칙/카테고리를 조회·변경

> 중요: 컨벤션/카테고리를 추가/편집/삭제한 뒤에는 **변환(`convert`)을 실행해** `.sym/code-policy.json` 및 린터 설정 파일을 갱신하는 것을 권장합니다.

---

## 공통 개념

- **원본 정책(A Schema)**: `.sym/user-policy.json`
  - 사람이 읽고 편집하기 쉬운 정책(카테고리/규칙)
- **파생 정책(B Schema)**: `.sym/code-policy.json`
  - 검증/린팅에 쓰이는 변환 결과
- **파생 산출물**: `.sym/.eslintrc.json`, `.sym/.prettierrc.json`, `.sym/.pylintrc` 등

---

## 1) MCP를 사용한 컨벤션/카테고리 관리

### 준비

- 프로젝트에 Symphony가 초기화돼 있어야 합니다.

```bash
sym init
```

- MCP 클라이언트(Cursor/Claude Code 등)에 Symphony MCP 서버가 등록돼 있어야 합니다.
  - 이 저장소는 프로젝트 루트의 `.mcp.json`에 예시를 둡니다.

### 사용 가능한 MCP 도구

- **조회**
  - `list_category`: 카테고리 목록 조회
  - `list_convention`: 규칙(컨벤션) 목록 조회
- **카테고리 변경**
  - `add_category`, `edit_category`, `remove_category`
- **컨벤션 변경**
  - `add_convention`, `edit_convention`, `remove_convention`
- **변환**
  - `convert`: user-policy.json → code-policy.json + 린터 설정 생성/갱신

### 기본 흐름(권장)

1. `list_category`로 카테고리 확인
2. 필요 시 `add_category`/`edit_category`로 카테고리 정리
3. `list_convention`으로 규칙 확인
4. 필요 시 `add_convention`/`edit_convention`/`remove_convention`으로 규칙 수정
5. **`convert` 실행**(파생 정책/린터 설정 갱신)
6. (선택) `validate_code`로 Git 변경사항 검증

### 예시 Payload

아래 JSON은 MCP 도구 호출 시 전달하는 입력 예시입니다.

#### 카테고리 추가(`add_category`)

```json
{
  "categories": [
    {"name": "accessibility", "description": "Accessibility rules (WCAG, ARIA, etc.)"}
  ]
}
```

#### 카테고리 편집(`edit_category`)

```json
{
  "edits": [
    {"name": "security", "description": "Security rules (updated)"},
    {"name": "style", "new_name": "formatting"}
  ]
}
```

#### 카테고리 삭제(`remove_category`)

```json
{
  "names": ["deprecated-category"]
}
```

> 참고: 규칙이 참조 중인 카테고리는 삭제할 수 없습니다(먼저 규칙의 카테고리를 변경하거나 규칙을 삭제해야 함).

#### 컨벤션 조회(`list_convention`)

```json
{
  "category": "security",
  "languages": ["go", "python"]
}
```

#### 컨벤션 추가(`add_convention`)

```json
{
  "conventions": [
    {
      "id": "SEC-001",
      "say": "Use parameterized queries for database operations",
      "category": "security",
      "languages": ["go", "python"],
      "severity": "error"
    }
  ]
}
```

#### 컨벤션 편집(`edit_convention`)

```json
{
  "edits": [
    {"id": "SEC-001", "severity": "warning"},
    {"id": "SEC-001", "say": "Use parameterized queries (prepared statements)"}
  ]
}
```

#### 컨벤션 삭제(`remove_convention`)

```json
{
  "ids": ["SEC-001"]
}
```

#### 변환(`convert`)

```json
{
  "input_path": ".sym/user-policy.json",
  "output_dir": ".sym"
}
```

---

## 2) Dashboard를 사용한 컨벤션/카테고리 관리

### 대시보드 실행

```bash
# 기본 포트 8787
sym dashboard

# 또는 별칭
sym dash
```

브라우저에서 `http://localhost:8787`로 접속합니다.

### 카테고리 관리

- **추가**: 새 카테고리 이름/설명을 입력해 생성
- **편집**: 이름/설명 변경
- **삭제**: 해당 카테고리를 참조하는 규칙이 있으면 삭제 불가

### 컨벤션(규칙) 관리

- **추가**: 규칙 ID, 설명(say), 카테고리, 언어, 심각도 등을 입력
- **편집**: 기존 규칙의 속성 수정
- **삭제**: 규칙 삭제

### 변환

대시보드에서 저장을 누르면 convert할지 요청을 합니다.

---

## 3) CLI를 사용한 컨벤션/카테고리 관리

CLI 방식은 `sym` 명령어로 **카테고리/컨벤션(규칙)을 조회·추가·편집·삭제**하고, 마지막에 `convert`/`validate`로 파생 산출물 생성 및 검증까지 한 번에 처리하는 방법입니다.

### 준비

- 프로젝트가 초기화돼 있어야 합니다.

```bash
sym init
```

- (선택) 현재 사용 중인 정책 파일 경로를 확인합니다.

```bash
sym policy path
```

### 기본 흐름(권장)

1. 카테고리 조회/정리
2. 컨벤션(규칙) 조회/정리
3. **`sym convert` 실행**(파생 정책/린터 설정 갱신)
4. (선택) **`sym validate` 실행**(Git 변경사항 검증)

### 카테고리 관리

```bash
# 목록
sym category list

# 추가
sym category add accessibility "Accessibility rules (WCAG, ARIA, etc.)"

# 편집(이름/설명)
sym category edit security --description "Updated security rules"
sym category edit old-name --name new-name

# 삭제(참조 중인 규칙이 있으면 삭제 불가)
sym category remove deprecated-category
```

> 팁: 여러 항목을 한 번에 처리해야 한다면 `-f <file.json>` 형태의 파일 입력 옵션을 사용할 수 있습니다.

### 컨벤션(규칙) 관리

```bash
# 목록(필터)
sym convention list
sym convention list --category security
sym convention list --language go

# 추가
sym convention add SEC-001 "Use parameterized queries" --category security --languages go,python --severity error

# 편집
sym convention edit SEC-001 --severity warning
sym convention edit SEC-001 --new-id SEC-001-v2

# 삭제
sym convention remove SEC-001
```

### 변환/검증

컨벤션/카테고리를 변경한 뒤에는 파생 정책 및 린터 설정이 최신이 되도록 변환을 실행하는 것을 권장합니다.

```bash
# 파생 정책/린터 설정 갱신
sym convert

# 정책 파일을 직접 지정하거나, 출력 디렉토리를 바꾸고 싶다면
sym convert -i user-policy.json
sym convert -i user-policy.json -o ./custom-dir
```

Git 변경사항 기준으로 정책 위반을 검증하려면:

```bash
sym validate
sym validate --staged
sym validate --policy custom-policy.json
sym validate --timeout 60
```

---

## Import로 컨벤션 관리 (문서 → 정책 병합)

Import는 팀 가이드/보안 규칙/코딩 표준 문서처럼 **이미 존재하는 문서**를 기준으로, LLM이 컨벤션을 추출해 `.sym/user-policy.json`에 병합하는 방식입니다.

### 권장 흐름(사용자 관점)

1. 문서 준비: 예) `docs/team-standards.md`
2. Import 실행(아래 3가지 방법 중 택1)
3. `convert` 실행(파생 정책/린터 설정 갱신)
4. (선택) `validate` 또는 MCP `validate_code`로 Git 변경사항 검증

### (A) MCP로 Import (`import_convention`)

LLM IDE(Cursor/Claude Code 등)에서 Symphony MCP 도구를 사용할 수 있다면 다음 흐름을 권장합니다:

1. `import_convention`으로 문서를 정책에 반영
2. `convert`로 파생 산출물 갱신
3. `validate_code`로 변경사항 검증

#### 예시 Payload

```json
{
  "path": "docs/team-standards.md",
  "mode": "append"
}
```

> 예: “`docs/team-standards.md`를 컨벤션으로 반영해줘”라고 요청하면, LLM이 `import_convention` → `convert` → (필요 시) `validate_code` 순으로 실행하는 형태를 기대할 수 있습니다.

### (B) Dashboard에서 Import

대시보드(`sym dash` 또는 `./bin/sym dash`)에서도 Import UI를 통해 문서를 선택해 정책에 병합할 수 있습니다.
Import 후에는 저장을 누르면 convert할지 사용자에게 요청을 합니다.

### (C) CLI로 Import (`sym import`)

```bash
# 기본: append (기존 유지 + 새 항목 추가)
sym import docs/team-standards.md

# 기존 컨벤션을 비우고 새로 구성하려면 (주의)
sym import docs/team-standards.md --mode clear

# Import 이후 파생 정책/린터 설정 갱신
sym convert
```
