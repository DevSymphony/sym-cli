# adapter

외부 검증 도구를 표준 인터페이스로 통합하는 어댑터 레이어입니다.

ESLint, Prettier, TypeScript Compiler(TSC), Checkstyle, PMD 등의 도구를 subprocess로 실행하고 결과를 파싱합니다.

## 서브패키지

- `eslint`: JavaScript/TypeScript 린터 어댑터
- `prettier`: 코드 포매터 어댑터
- `tsc`: TypeScript 타입 체커 어댑터
- `checkstyle`: Java 스타일 체커 어댑터
- `pmd`: Java 정적 분석 도구 어댑터
- `registry`: 어댑터 등록 및 검색 시스템

**사용자**: engine
**의존성**: engine/core
