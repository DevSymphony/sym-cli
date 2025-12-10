# server

Symphony 정책 편집기를 위한 웹 대시보드 HTTP 서버를 제공합니다.
역할 기반 접근 제어(RBAC), 정책 관리 REST API, 정적 파일 서빙 기능을 포함합니다.

## 패키지 구조

```
internal/server/
├── server.go              # 메인 서버 구현 (HTTP 핸들러, CORS, 권한 검사)
├── README.md              # 패키지 문서
└── static/                # 임베디드 웹 에셋 (go:embed)
    ├── index.html         # 웹 UI HTML (정책 편집기)
    ├── policy-editor.js   # 클라이언트 JavaScript (상태 관리, API 통신)
    ├── styles/            # CSS 파일
    │   ├── input.css      # Tailwind 소스
    │   └── output.css     # 컴파일된 CSS
    ├── icons/             # SVG 아이콘 (10개)
    └── fonts/             # Pretendard 폰트 (Regular, Medium, Bold)
```

## 의존성

### 패키지 사용자

| 패키지 | 파일 | 용도 |
|--------|------|------|
| internal/cmd | dashboard.go | `sym dashboard` 명령어에서 서버 인스턴스 생성 및 실행 |

### 패키지 의존성

| 패키지 | 용도 |
|--------|------|
| internal/converter | 정책 변환 (`/api/policy/convert` 엔드포인트) |
| internal/llm | LLM 프로바이더 생성 (정책 변환용) |
| internal/policy | 정책 파일 로드/저장, 템플릿 관리 |
| internal/roles | 역할 관리, 현재 역할 조회/설정 |
| internal/util/config | 프로젝트 설정(config.json) 로드/저장 |
| pkg/schema | UserPolicy 타입 정의 |

**외부 의존성:**
- `github.com/pkg/browser` - 서버 시작 시 브라우저 자동 열기

## Public/Private API

### Public API

| 함수 | 시그니처 | 설명 |
|------|----------|------|
| NewServer | `NewServer(port int) (*Server, error)` | 지정된 포트로 서버 인스턴스 생성 |
| Start | `(s *Server) Start() error` | HTTP 서버 시작, 라우트 등록, 브라우저 열기 |

**Server 구조체:**
```go
type Server struct {
    port int
}
```

### Private API

**미들웨어:**
- `corsMiddleware(next http.Handler) http.Handler` - CORS 헤더 추가 (모든 origin 허용)

**권한 검사:**
- `hasPermissionForRole(role, permission string) (bool, error)` - config에서 정책 로드 후 권한 확인
- `hasPermissionForRoleWithPolicy(role, permission string, policy *schema.UserPolicy) (bool, error)` - 제공된 정책으로 권한 확인
- `checkPermissionForRole(userRole, permission string, policy *schema.UserPolicy) (bool, error)` - 권한 검사 핵심 로직

**HTTP 핸들러 (14개):**

| 핸들러 | 라우트 | 설명 |
|--------|--------|------|
| handleGetMe | GET /api/me | 현재 역할 및 권한 조회 |
| handleSelectRole | POST /api/select-role | 역할 선택/변경 |
| handleAvailableRoles | GET /api/available-roles | 사용 가능한 역할 목록 |
| handleRoles | /api/roles | GET/POST 라우터 |
| handleGetRoles | GET /api/roles | roles.json 조회 |
| handleUpdateRoles | POST /api/roles | roles.json 업데이트 |
| handleProjectInfo | GET /api/project-info | 프로젝트 이름 조회 |
| handlePolicy | /api/policy | GET/POST 라우터 |
| handleGetPolicy | GET /api/policy | 정책 파일 조회 |
| handleSavePolicy | POST /api/policy | 정책 파일 저장 |
| handlePolicyPath | /api/policy/path | GET/POST 라우터 |
| handleSetPolicyPath | POST /api/policy/path | 정책 파일 경로 변경 |
| handlePolicyTemplates | GET /api/policy/templates | 템플릿 목록 조회 |
| handlePolicyTemplateDetail | GET /api/policy/templates/{name} | 특정 템플릿 상세 |
| handleUsers | GET /api/users | 모든 사용자 목록 |
| handleConvert | POST /api/policy/convert | 정책 변환 실행 |

## 참고 문헌

- [Server REST API Reference](../../docs/server-api.md) - 엔드포인트 상세 문서 (요청/응답 스키마, 권한 모델)
