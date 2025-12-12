# policy

정책 파일의 로드, 저장, 검증 및 템플릿 관리를 담당하는 패키지입니다.

UserPolicy(A 스키마)와 CodePolicy(B 스키마) 파일을 로드하고, 정책 유효성 검증, 저장, 다양한 프레임워크 템플릿 제공 기능을 수행합니다.

## 패키지 구조

```
internal/policy/
├── loader.go        # 정책 파일 로더 (Loader 구조체)
├── manager.go       # 정책 관리 함수 (경로, 로드, 저장, 검증)
├── defaults.go      # defaults.languages 자동 업데이트 함수
├── templates.go     # 템플릿 관리 (embed.FS 기반)
├── README.md
└── templates/       # 내장 정책 템플릿 (7개)
    ├── demo-template.json      # 샘플 자바 정책
    ├── go-template.json        # Go 마이크로서비스
    ├── node-template.json      # Node.js 백엔드
    ├── python-template.json    # Python (PEP 8, Django/Flask)
    ├── react-template.json     # React 프로젝트
    ├── typescript-template.json # TypeScript 라이브러리
    └── vue-template.json       # Vue.js 프로젝트
```

## 의존성

### 패키지 사용자

| 패키지 | 파일 | 사용 목적 |
|--------|------|-----------|
| cmd | policy.go | 정책 경로 표시, 검증 CLI 명령 |
| cmd | init.go | 프로젝트 초기화 시 기본 정책 생성 |
| cmd | convention.go | 컨벤션 추가/편집 시 언어 자동 업데이트 |
| roles | rbac.go | RBAC 검증을 위한 정책 로드 |
| server | server.go | 대시보드 REST API (정책 CRUD, 템플릿) |
| mcp | server.go | MCP 서버 정책 로드, 변환, add/edit_convention 시 언어 자동 업데이트 |
| importer | importer.go | Import 시 언어 자동 업데이트 |

### 패키지 의존성

| 패키지 | 사용 목적 |
|--------|-----------|
| internal/util/git | Git 저장소 루트 경로 획득 (GetRepoRoot) |
| pkg/schema | UserPolicy, CodePolicy 스키마 타입 |

## Public/Private API

### Public API

#### Types

**Loader** (loader.go:12)
```go
type Loader struct{}
```
정책 파일 로더. `LoadUserPolicy`, `LoadCodePolicy` 메서드 제공.

**Template** (templates.go:18)
```go
type Template struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Language    string `json:"language"`
    Framework   string `json:"framework,omitempty"`
}
```
템플릿 메타데이터.

#### Functions

| 함수 | 파일 | 설명 |
|------|------|------|
| `NewLoader(verbose bool) *Loader` | loader.go:16 | 새 로더 인스턴스 생성 |
| `(*Loader).LoadUserPolicy(path) (*UserPolicy, error)` | loader.go:21 | A 스키마 로드 |
| `(*Loader).LoadCodePolicy(path) (*CodePolicy, error)` | loader.go:36 | B 스키마 로드 |
| `GetPolicyPath(customPath) (string, error)` | manager.go:16 | 정책 파일 전체 경로 반환 |
| `LoadPolicy(customPath) (*UserPolicy, error)` | manager.go:31 | 정책 로드 (없으면 빈 정책 반환) |
| `SavePolicy(policy, customPath) error` | manager.go:58 | 정책 저장 (검증 후) |
| `ValidatePolicy(policy) error` | manager.go:84 | 정책 구조 유효성 검증 |
| `PolicyExists(customPath) (bool, error)` | manager.go:133 | 정책 파일 존재 여부 |
| `GetTemplates() ([]Template, error)` | templates.go:26 | 템플릿 목록 반환 |
| `GetTemplate(name) (*UserPolicy, error)` | templates.go:81 | 특정 템플릿 로드 |
| `UpdateDefaultsLanguages(policy, rules)` | defaults.go:9 | 규칙에서 언어 추출하여 defaults.languages에 추가 |

### Private API

| 변수/함수 | 파일 | 설명 |
|-----------|------|------|
| `defaultPolicyPath` | manager.go:13 | 기본 경로: `.sym/user-policy.json` |
| `templateFiles` | templates.go:15 | embed.FS (내장 템플릿) |
