# schema

중앙 데이터 구조를 정의합니다.

UserPolicy(A Schema) 및 CodePolicy(B Schema) 타입을 정의하며, 전체 시스템의 데이터 흐름을 담당합니다.

## 패키지 구조

```
pkg/schema/
├── types.go    # 모든 스키마 타입 정의
└── README.md
```

## 의존성

### 패키지 사용자

| 패키지 | 용도 |
|--------|------|
| `internal/cmd` | CLI 명령어에서 정책 로드/저장 |
| `internal/converter` | A→B 스키마 변환 |
| `internal/llm` | LLM 응답 파싱 |
| `internal/mcp` | MCP 서버 정책 처리 |
| `internal/policy` | 정책 파일 로드/저장 |
| `internal/roles` | RBAC 검증 |
| `internal/server` | 웹 대시보드 API |
| `internal/validator` | 코드 검증 |

### 패키지 의존성

- 없음 (독립 패키지)

## 스키마 관계

```
┌─────────────────────────────────────────────────────────────────┐
│                    A Schema (사용자 친화적)                      │
│  UserPolicy ──┬── UserRBAC ─── UserRole                         │
│               ├── UserDefaults                                  │
│               └── UserRule[]                                    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    converter.Convert()
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    B Schema (시스템 실행용)                      │
│  CodePolicy ──┬── PolicyRBAC ─┬─ PolicyRole                     │
│               │               └─ Permission ─ PermissionCond.   │
│               ├── ProjectInfo                                   │
│               ├── PolicyRule[] ─┬─ Selector                     │
│               │                 └─ Remedy                       │
│               └── EnforceSettings ─ RBACEnforce                 │
└─────────────────────────────────────────────────────────────────┘
```

## A Schema (사용자 입력용)

사용자가 자연어로 컨벤션을 정의하기 위한 간소화된 스키마입니다.

### UserPolicy

메인 정책 구조체입니다.

```go
type UserPolicy struct {
    Version  string        `json:"version,omitempty"`  // 정책 버전 (예: "1.0.0")
    RBAC     *UserRBAC     `json:"rbac,omitempty"`     // 역할 기반 접근 제어
    Defaults *UserDefaults `json:"defaults,omitempty"` // 규칙 기본값
    Rules    []UserRule    `json:"rules"`              // 자연어 규칙 목록
}
```

### UserRBAC

RBAC 설정을 담는 구조체입니다.

```go
type UserRBAC struct {
    Roles map[string]UserRole `json:"roles"` // 역할명 → 역할 정의
}
```

### UserRole

단일 역할의 권한을 정의합니다.

```go
type UserRole struct {
    AllowWrite    []string `json:"allowWrite,omitempty"`    // 쓰기 허용 경로 (glob)
    DenyWrite     []string `json:"denyWrite,omitempty"`     // 쓰기 거부 경로 (glob)
    AllowExec     []string `json:"allowExec,omitempty"`     // 실행 허용 경로 (glob)
    CanEditPolicy bool     `json:"canEditPolicy,omitempty"` // 정책 편집 권한
    CanEditRoles  bool     `json:"canEditRoles,omitempty"`  // 역할 편집 권한
}
```

### UserDefaults

규칙의 기본값을 정의합니다.

```go
type UserDefaults struct {
    Languages       []string `json:"languages,omitempty"`       // 기본 대상 언어
    DefaultLanguage string   `json:"defaultLanguage,omitempty"` // 기본 언어
    Include         []string `json:"include,omitempty"`         // 포함 경로
    Exclude         []string `json:"exclude,omitempty"`         // 제외 경로
    Severity        string   `json:"severity,omitempty"`        // 기본 심각도
    Autofix         bool     `json:"autofix,omitempty"`         // 자동 수정 여부
}
```

### UserRule

자연어로 정의된 단일 규칙입니다.

```go
type UserRule struct {
    ID        string         `json:"id"`                   // 규칙 ID (필수)
    Say       string         `json:"say"`                  // 자연어 규칙 설명
    Category  string         `json:"category,omitempty"`   // 카테고리 (naming, formatting 등)
    Languages []string       `json:"languages,omitempty"`  // 적용 언어
    Include   []string       `json:"include,omitempty"`    // 포함 경로
    Exclude   []string       `json:"exclude,omitempty"`    // 제외 경로
    Severity  string         `json:"severity,omitempty"`   // 심각도 (error, warning, info)
    Autofix   bool           `json:"autofix,omitempty"`    // 자동 수정 여부
    Params    map[string]any `json:"params,omitempty"`     // 추가 파라미터
    Message   string         `json:"message,omitempty"`    // 위반 시 메시지
    Example   string         `json:"example,omitempty"`    // 예시
}
```

## B Schema (시스템 실행용)

린터 및 검증기가 실행하기 위한 정형화된 스키마입니다.

### CodePolicy

메인 검증 정책 구조체입니다.

```go
type CodePolicy struct {
    Version string          `json:"version"`           // 정책 버전
    Project *ProjectInfo    `json:"project,omitempty"` // 프로젝트 메타데이터
    Extends []string        `json:"extends,omitempty"` // 상속할 정책 URI
    RBAC    *PolicyRBAC     `json:"rbac,omitempty"`    // RBAC 설정
    Rules   []PolicyRule    `json:"rules"`             // 정형화된 규칙 목록
    Enforce EnforceSettings `json:"enforce"`           // 집행 설정
}
```

### ProjectInfo

프로젝트 메타데이터입니다.

```go
type ProjectInfo struct {
    Name       string   `json:"name,omitempty"`       // 프로젝트 이름
    Languages  []string `json:"languages,omitempty"`  // 사용 언어
    Frameworks []string `json:"frameworks,omitempty"` // 프레임워크
}
```

### PolicyRBAC / PolicyRole / Permission

정책 스키마용 RBAC 구조체입니다.

```go
type PolicyRBAC struct {
    Roles map[string]PolicyRole `json:"roles"` // 역할명 → 역할 정의
}

type PolicyRole struct {
    Inherits    []string     `json:"inherits,omitempty"` // 상속할 역할
    Permissions []Permission `json:"permissions"`        // 권한 목록
}

type Permission struct {
    Path       string                `json:"path"`                // 경로 (glob)
    Read       bool                  `json:"read"`                // 읽기 권한
    Write      bool                  `json:"write"`               // 쓰기 권한
    Execute    bool                  `json:"execute"`             // 실행 권한
    Conditions *PermissionConditions `json:"conditions,omitempty"` // 조건
}

type PermissionConditions struct {
    Branches []string   `json:"branches,omitempty"` // 적용 브랜치
    Time     *TimeRange `json:"time,omitempty"`     // 시간 범위
}

type TimeRange struct {
    Start string `json:"start,omitempty"` // 시작 시간 (HH:MM:SS)
    End   string `json:"end,omitempty"`   // 종료 시간 (HH:MM:SS)
}
```

### PolicyRule

정형화된 단일 규칙입니다.

```go
type PolicyRule struct {
    ID       string         `json:"id"`                 // 규칙 ID
    Enabled  bool           `json:"enabled"`            // 활성화 여부
    Category string         `json:"category"`           // 카테고리
    Severity string         `json:"severity"`           // 심각도
    Desc     string         `json:"desc,omitempty"`     // 설명
    When     *Selector      `json:"when,omitempty"`     // 적용 조건
    Check    map[string]any `json:"check"`              // 검사 설정 (engine 필드 포함)
    Remedy   *Remedy        `json:"remedy,omitempty"`   // 자동 수정 설정
    Message  string         `json:"message,omitempty"`  // 위반 시 메시지
}
```

### Selector

규칙 적용 조건을 정의합니다.

```go
type Selector struct {
    Languages []string `json:"languages,omitempty"` // 대상 언어
    Include   []string `json:"include,omitempty"`   // 포함 경로
    Exclude   []string `json:"exclude,omitempty"`   // 제외 경로
    Branches  []string `json:"branches,omitempty"`  // 대상 브랜치
    Roles     []string `json:"roles,omitempty"`     // 대상 역할
    Tags      []string `json:"tags,omitempty"`      // 태그
}
```

### Remedy

자동 수정 설정입니다.

```go
type Remedy struct {
    Autofix bool           `json:"autofix"`          // 자동 수정 활성화
    Tool    string         `json:"tool,omitempty"`   // 수정 도구 (prettier, black 등)
    Config  map[string]any `json:"config,omitempty"` // 도구 설정
}
```

### EnforceSettings / RBACEnforce

집행 설정입니다.

```go
type EnforceSettings struct {
    Stages     []string     `json:"stages"`            // 집행 단계 (pre-commit, ci 등)
    FailOn     []string     `json:"fail_on,omitempty"` // 실패 조건 심각도
    RBACConfig *RBACEnforce `json:"rbac,omitempty"`    // RBAC 집행 설정
}

type RBACEnforce struct {
    Enabled     bool     `json:"enabled"`               // RBAC 활성화
    Stages      []string `json:"stages,omitempty"`      // RBAC 집행 단계
    OnViolation string   `json:"on_violation,omitempty"` // 위반 시 동작 (block, warn)
}
```

## 사용 예시

### A Schema 예시

```json
{
  "version": "1.0.0",
  "rbac": {
    "roles": {
      "admin": { "allowWrite": ["**/*"], "canEditPolicy": true },
      "developer": { "allowWrite": ["src/**"], "denyWrite": [".sym/**"] }
    }
  },
  "defaults": {
    "languages": ["typescript", "python"],
    "severity": "warning"
  },
  "rules": [
    { "id": "1", "say": "클래스 이름은 PascalCase", "category": "naming" },
    { "id": "2", "say": "한 줄은 100자 이하", "params": { "max": 100 } }
  ]
}
```

### B Schema 예시

```json
{
  "version": "1.0",
  "rules": [
    {
      "id": "NAMING-CLASS-PASCAL",
      "enabled": true,
      "category": "naming",
      "severity": "warning",
      "check": {
        "engine": "eslint",
        "rule": "@typescript-eslint/naming-convention"
      }
    }
  ],
  "enforce": {
    "stages": ["pre-commit", "ci"],
    "fail_on": ["error"]
  }
}
```
