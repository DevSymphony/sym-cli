# git

Git 저장소의 변경 사항 조회 및 저장소 정보를 제공하는 패키지.

## 패키지 구조

```
git/
├── changes.go       # 변경 사항 조회 및 diff 처리
├── changes_test.go  # 테스트
├── repo.go          # 저장소 정보 조회
└── README.md
```

## 의존성

### 패키지 사용자

| 패키지 | 용도 |
|--------|------|
| `internal/validator` | 검증 대상 파일 및 diff 획득 |
| `internal/cmd/validate.go` | CLI validate 명령어 |
| `internal/mcp/server.go` | MCP validate_code 도구 |
| `internal/roles` | RBAC 권한 검증 |
| `internal/policy` | 정책 관리 |

### 패키지 의존성

없음.

## Public/Private API

### Public API

**Types**

| 타입 | 파일 | 설명 |
|------|------|------|
| `Change` | changes.go:10 | 파일 변경 정보 (FilePath, Status, Diff) |

**Functions**

| 함수 | 파일 | 설명 |
|------|------|------|
| `GetChanges()` | changes.go:18 | 모든 변경 사항 (staged + unstaged + untracked) |
| `GetStagedChanges()` | changes.go:146 | staged 변경만 |
| `ExtractAddedLines(diff)` | changes.go:185 | diff에서 추가된 줄 추출 |
| `GetRepoRoot()` | repo.go:10 | 저장소 루트 경로 |
| `GetCurrentUser()` | repo.go:21 | Git user.name |

### Private API

없음.
