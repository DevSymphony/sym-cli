# engine

다양한 검증 엔진을 구현합니다.

pattern, length, style, ast, llm, typechecker 등의 엔진을 제공하며, 엔진 레지스트리를 통해 통합 관리합니다.

## 서브패키지

- `core`: 엔진 인터페이스 및 공통 타입 정의
- `registry`: 엔진 등록 및 검색 시스템
- `pattern`: 정규식 패턴 매칭 엔진 (→ adapter/eslint)
- `length`: 라인/파일 길이 검증 엔진 (→ adapter/eslint)
- `style`: 코드 스타일 검증 엔진 (→ adapter/eslint, adapter/prettier)
- `ast`: AST 구조 검증 엔진 (→ adapter/eslint, adapter/checkstyle, adapter/pmd)
- `llm`: LLM 기반 검증 엔진 (→ llm)
- `typechecker`: 타입 체킹 엔진 (→ adapter/tsc)

**사용자**: adapter, validator
**의존성**: adapter, llm
