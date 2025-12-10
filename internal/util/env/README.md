# env

환경 변수 및 `.env` 파일 관리 패키지

## 패키지 구조

```
env/
└── env.go    # .env 파일 읽기/쓰기
```

## 의존성

### 패키지 사용자

- `internal/cmd/llm.go` - API 키 저장
- `internal/roles/roles.go` - 현재 역할 저장/로드
- `internal/llm/openaiapi/provider.go` - API 키 로드

### 패키지 의존성

- 없음 (외부 의존성만 사용)

## Public API

| API | 설명 |
|-----|------|
| `GetAPIKey(keyName)` | 환경변수 또는 `.sym/.env`에서 API 키 조회 |
| `LoadKeyFromEnvFile(path, key)` | `.env` 파일에서 특정 키 로드 |
| `SaveKeyToEnvFile(path, key, value)` | `.env` 파일에 키-값 저장 |

## Private API

없음
