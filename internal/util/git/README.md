# git

Git 변경사항 감지 및 저장소 정보 패키지

## 패키지 구조

```
git/
├── changes.go       # Git 변경사항 감지
├── changes_test.go  # 테스트
└── repo.go          # 저장소 정보 조회
```

## 의존성

### 패키지 사용자

- `internal/cmd/init.go` - 저장소 루트 확인
- `internal/cmd/validate.go` - 변경사항 검증
- `internal/cmd/mcp.go` - 저장소 루트 확인
- `internal/mcp/server.go` - MCP 검증 시 변경사항 조회
- `internal/validator/validator.go` - 변경사항 필터링
- `internal/validator/llm_validator.go` - diff 추출
- `internal/validator/execution_unit.go` - 추가된 라인 추출
- `internal/roles/rbac.go` - 저장소 루트 확인
- `internal/policy/manager.go` - 정책 파일 경로 구성
- `tests/integration/*` - 통합 테스트
- `tests/e2e/*` - E2E 테스트

### 패키지 의존성

- 없음 (외부 의존성만 사용)

## Public API

| API | 설명 |
|-----|------|
| `Change` | Git 변경 정보 구조체 (`FilePath`, `Status`, `Diff`) |
| `GetChanges()` | 모든 미커밋 변경사항 조회 (staged + unstaged + untracked) |
| `GetStagedChanges()` | 스테이징된 변경사항만 조회 |
| `ExtractAddedLines(diff)` | diff에서 추가된 라인만 추출 |
| `GetRepoRoot()` | Git 저장소 루트 경로 |
| `GetCurrentUser()` | 현재 Git 사용자 이름 |

## Private API

없음
