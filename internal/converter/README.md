# converter

UserPolicy(A Schema)를 CodePolicy(B Schema)로 변환합니다.

자연어 규칙을 구조화된 검증 규칙으로 변환하고 ESLint, Prettier, Checkstyle, PMD 등의 linter 설정 파일을 자동 생성합니다.

## 서브패키지

- `linters`: 각 linter별 설정 파일 생성기 (ESLint, Prettier, Checkstyle, PMD)
- `linters/registry`: Linter 변환기 등록 및 검색 시스템

**사용자**: cmd, mcp
**의존성**: llm, schema
