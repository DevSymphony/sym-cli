# config

설정 관리 패키지 (사용자 전역 설정 + 프로젝트 설정)

## 패키지 구조

```
config/
├── config.go     # 사용자 전역 설정 (~/.config/sym/config.json)
└── project.go    # 프로젝트 레벨 설정 (.sym/config.json)
```

## 의존성

### 패키지 사용자

- `internal/cmd/init.go` - 프로젝트 초기화 시 설정 생성
- `internal/cmd/convert.go` - 정책 경로 로드
- `internal/cmd/llm.go` - LLM 프로바이더 설정
- `internal/cmd/policy.go` - 정책 경로 관리 (사용자 전역 설정)
- `internal/mcp/server.go` - MCP 서버 설정 로드
- `internal/server/server.go` - 대시보드 API 설정
- `internal/llm/config.go` - LLM 설정 로드

### 패키지 의존성

- 없음 (외부 의존성만 사용)

## Public API

### 사용자 전역 설정 (config.go)

| API | 설명 |
|-----|------|
| `Config` | 사용자 전역 설정 구조체 (`PolicyPath` 필드) |
| `LoadConfig()` | `~/.config/sym/config.json` 로드 |
| `SaveConfig(cfg)` | 설정 저장 |

### 프로젝트 설정 (project.go)

| API | 설명 |
|-----|------|
| `ProjectConfig` | 프로젝트 설정 구조체 |
| `LLMConfig` | LLM 설정 구조체 (`Provider`, `Model`) |
| `MCPConfig` | MCP 설정 구조체 (`Tools`) |
| `LoadProjectConfig()` | `.sym/config.json` 로드 |
| `SaveProjectConfig(cfg)` | 설정 저장 |
| `ProjectConfigExists()` | 설정 파일 존재 확인 |
| `GetProjectConfigPath()` | 설정 파일 경로 반환 |
| `GetProjectEnvPath()` | `.env` 파일 경로 반환 |
| `UpdateProjectConfigLLM(provider, model)` | LLM 설정 업데이트 |
| `UpdateProjectConfigMCP(tools)` | MCP 설정 업데이트 |

## Private API

| API | 설명 |
|-----|------|
| `init()` | 전역 설정 경로 초기화 (config.go) |
| `ensureConfigDir()` | 설정 디렉터리 생성 (config.go) |
