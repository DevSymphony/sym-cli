# roles

RBAC(Role-Based Access Control) 기능을 제공합니다.

역할 정의 관리와 파일 접근 권한 검증을 담당합니다.

## 패키지 구조

```
roles/
├── roles.go    # 역할 정의 관리 (Load/Save, Get/Set)
├── rbac.go     # RBAC 권한 검증
└── README.md
```

## 의존성

### 패키지 사용자

| 위치 | 용도 |
|------|------|
| `internal/cmd/init.go` | 초기 역할 생성 |
| `internal/cmd/my_role.go` | 역할 선택 CLI |
| `internal/cmd/dashboard.go` | 대시보드 역할 확인 |
| `internal/server/server.go` | 웹 대시보드 역할 API |
| `internal/mcp/server.go` | MCP RBAC 컨텍스트 |
| `internal/validator/validator.go` | 검증 시 RBAC 권한 확인 |

### 패키지 의존성

```
      ┌──────────┐
      │  roles   │
      └────┬─────┘
           │
     ┌─────┼─────┐
     ▼     ▼     ▼
┌────────┐ ┌────────┐ ┌────────────┐
│ envutil│ │  git   │ │   policy   │
└────────┘ └────────┘ └────────────┘
                           │
                           ▼
                    ┌────────────┐
                    │ pkg/schema │
                    └────────────┘
```

## Public / Private API

### Public API

#### Types

| 타입 | 파일 | 설명 |
|------|------|------|
| `Roles` | roles.go:15 | `map[string][]string` - 역할별 사용자 목록 |
| `ValidationResult` | rbac.go:15 | RBAC 검증 결과 |

#### Functions - 역할 파일 관리

| 함수 | 파일 | 설명 |
|------|------|------|
| `GetRolesPath()` | roles.go:39 | .sym/roles.json 경로 반환 |
| `LoadRoles()` | roles.go:48 | roles.json 로드 |
| `SaveRoles(roles)` | roles.go:71 | roles.json 저장 |
| `RolesExists()` | roles.go:110 | roles.json 존재 여부 |

#### Functions - 역할 상태 관리

| 함수 | 파일 | 설명 |
|------|------|------|
| `GetCurrentRole()` | roles.go:129 | 현재 선택된 역할 (.sym/.env) |
| `SetCurrentRole(role)` | roles.go:139 | 현재 역할 설정 |
| `GetAvailableRoles()` | roles.go:149 | 사용 가능한 역할 목록 |
| `IsValidRole(role)` | roles.go:165 | 역할 유효성 검증 |

#### Functions - 사용자/권한 검증

| 함수 | 파일 | 설명 |
|------|------|------|
| `GetUserRole(username)` | roles.go:91 | 사용자의 역할 조회 |
| `LoadUserPolicyFromRepo()` | rbac.go:30 | user-policy.json 로드 |
| `ValidateFilePermissionsForRole(role, files)` | rbac.go:132 | 역할별 파일 권한 검증 |

### Private API

| 함수 | 파일 | 설명 |
|------|------|------|
| `getSymDir()` | roles.go:21 | .sym 디렉토리 경로 |
| `getEnvPath()` | roles.go:30 | .sym/.env 경로 |
| `getUserPolicyPath()` | rbac.go:21 | user-policy.json 경로 |
| `matchPattern(pattern, path)` | rbac.go:48 | glob 패턴 매칭 |
| `checkFilePermission(file, role)` | rbac.go:104 | 단일 파일 권한 확인 |
