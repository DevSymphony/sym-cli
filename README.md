# Symphony (sym)

자연어 기반 코딩 컨벤션 관리 및 검증 도구

## 빠른 시작

### 1. 설치

```bash
npm i -g @dev-symphony/sym
```

### 2. 초기화

```bash
# 프로젝트 초기화 (.sym/ 폴더 생성, MCP 자동 설정)
sym init
```

### 3. 사용

**웹 대시보드로 컨벤션 편집:**
```bash
sym dashboard
# http://localhost:8787 에서 역할/컨벤션 편집
```

## 주요 기능

- 자연어로 코딩 컨벤션 정의 (`.sym/user-policy.json`)
- RBAC 기반 파일 접근 권한 관리
- LLM으로 자연어 규칙을 ESLint/Checkstyle/PMD 설정으로 변환
- MCP 서버로 Claude, Cursor 등 AI 도구와 연동
- 웹 대시보드에서 시각적으로 정책 편집

## MCP 설정

`sym init` 실행 시 MCP가 자동으로 설정됩니다.

수동 설정이 필요한 경우:

```json
{
  "mcpServers": {
    "symphony": {
      "command": "npx",
      "args": ["-y", "@dev-symphony/sym@latest", "mcp"]
    }
  }
}
```

## CLI 명령어

| 명령어 | 설명 |
|--------|------|
| `sym init` | 프로젝트 초기화 (.sym/ 생성) |
| `sym dashboard` | 웹 대시보드 실행 (포트 8787) |
| `sym my-role` | 내 역할 확인 |
| `sym policy validate` | 정책 파일 유효성 검사 |
| `sym convert -i user-policy.json --targets all` | 컨벤션을 linter 설정으로 변환 |
| `sym mcp` | MCP 서버 실행 |

## 정책 파일 예시

`.sym/user-policy.json`:

```json
{
  "version": "1.0.0",
  "rbac": {
    "roles": {
      "admin": { "allowWrite": ["**/*"] },
      "developer": { "allowWrite": ["src/**"], "denyWrite": [".sym/**"] }
    }
  },
  "rules": [
    { "say": "클래스 이름은 PascalCase", "category": "naming" },
    { "say": "한 줄은 100자 이하", "category": "formatting" }
  ]
}
```

`.sym/roles.json`:

```json
{
  "admin": ["alice"],
  "developer": ["bob", "charlie"]
}
```

## 개발

```bash
make setup      # 의존성 설치
make build      # 빌드
make test       # 테스트
make lint       # 린트
```

**필수 도구:** Go 1.21+, Node.js 18+

## 라이선스

MIT License

## 지원

- GitHub Issues: https://github.com/DevSymphony/sym-cli/issues
