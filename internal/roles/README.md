# roles

RBAC(Role-Based Access Control) 기능을 제공하는 패키지입니다. 역할 정의 관리와 파일 접근 권한 검증을 담당합니다.

## 패키지 구조

```
roles/
├── roles.go    # 역할 파일 관리 (Load/Save), 역할 상태 관리 (Get/Set)
├── rbac.go     # RBAC 권한 검증 (파일 권한 체크, 패턴 매칭)
└── README.md
```

- **roles.go**: `.sym/roles.json` 파일의 읽기/쓰기와 `.sym/.env`의 현재 역할 상태 관리
- **rbac.go**: `user-policy.json`의 AllowWrite/DenyWrite 패턴을 기반으로 파일 수정 권한 검증

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `internal/cmd/init.go` | 초기 역할 생성, `SetCurrentRole("admin")` 설정 |
| `internal/cmd/my_role.go` | 역할 선택 CLI 인터페이스 |
| `internal/cmd/dashboard.go` | 대시보드 시작 전 역할 존재 여부 확인 |
| `internal/server/server.go` | 웹 대시보드 역할 관련 API 엔드포인트 |
| `internal/mcp/server.go` | MCP 서버에서 RBAC 컨텍스트 메시지 생성 |
| `internal/validator/validator.go` | 검증 시 RBAC 파일 권한 확인 |
| `tests/integration/rbac_test.go` | RBAC 통합 테스트 |

### 패키지 의존성

```
        ┌──────────┐
        │  roles   │
        └────┬─────┘
             │
    ┌────────┼────────┐
    ▼        ▼        ▼
┌────────┐ ┌────────┐ ┌────────┐
│util/env│ │util/git│ │ policy │
└────────┘ └────────┘ └───┬────┘
                          │
                          ▼
                   ┌────────────┐
                   │ pkg/schema │
                   └────────────┘
```

| 패키지 | 용도 |
|--------|------|
| `internal/util/env` | `.env` 파일 읽기/쓰기 |
| `internal/util/git` | Git 저장소 루트 경로 탐지 |
| `internal/policy` | `user-policy.json` 로더 |
| `pkg/schema` | `UserPolicy`, `UserRole`, `UserRBAC` 타입 정의 |

## Public/Private API

### Public API

#### Types

| 타입 | 위치 | 설명 |
|------|------|------|
| `Roles` | roles.go:15 | `map[string][]string` - 역할명을 키로, 사용자명 목록을 값으로 가지는 맵 |
| `ValidationResult` | rbac.go:15 | RBAC 검증 결과 (`Allowed bool`, `DeniedFiles []string`) |

#### Functions - 역할 파일 관리

| 함수 | 위치 | 설명 |
|------|------|------|
| `GetRolesPath()` | roles.go:39 | `.sym/roles.json` 파일 경로 반환 |
| `LoadRoles()` | roles.go:48 | `roles.json` 파일을 읽어 `Roles` 맵 반환 |
| `SaveRoles(roles Roles)` | roles.go:71 | `Roles` 맵을 `roles.json` 파일로 저장 |
| `RolesExists()` | roles.go:111 | `roles.json` 파일 존재 여부 확인 |

#### Functions - 역할 상태 관리

| 함수 | 위치 | 설명 |
|------|------|------|
| `GetCurrentRole()` | roles.go:129 | `.sym/.env`에서 현재 선택된 역할 반환 |
| `SetCurrentRole(role string)` | roles.go:140 | 현재 역할을 `.sym/.env`에 저장 |
| `GetAvailableRoles()` | roles.go:151 | `roles.json`에 정의된 모든 역할명 목록 반환 (정렬됨) |
| `IsValidRole(role string)` | roles.go:166 | 역할명이 `roles.json`에 존재하는지 확인 |

#### Functions - 사용자/권한 검증

| 함수 | 위치 | 설명 |
|------|------|------|
| `GetUserRole(username string)` | roles.go:92 | 사용자명으로 역할 조회 (없으면 `"none"` 반환) |
| `LoadUserPolicyFromRepo()` | rbac.go:30 | `user-policy.json` 파일 로드 |
| `ValidateFilePermissionsForRole(role string, files []string)` | rbac.go:132 | 역할의 파일 수정 권한 검증 |

### Private API

| 함수 | 위치 | 설명 |
|------|------|------|
| `getSymDir()` | roles.go:21 | `.sym` 디렉토리 경로 반환 |
| `getEnvPath()` | roles.go:30 | `.sym/.env` 파일 경로 반환 |
| `getUserPolicyPath()` | rbac.go:21 | `user-policy.json` 파일 경로 반환 |
| `matchPattern(pattern, path string)` | rbac.go:48 | glob 패턴 매칭 (`**`, `*` 지원) |
| `checkFilePermission(filePath string, role *schema.UserRole)` | rbac.go:104 | 단일 파일 권한 확인 |
