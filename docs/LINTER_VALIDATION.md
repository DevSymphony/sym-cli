# Linter 설정 검증

## 목적

convert 기능은 자연어 코딩 컨벤션에서 linter별 설정을 생성합니다. 이 설정은 Git으로 추적되는 코드 변경사항을 검증하는 데 사용됩니다.

## 지원 Linter

### JavaScript/TypeScript
- **ESLint**: JS/TS 코드 스타일, 패턴, 모범 사례 검증
- 출력: `.sym/.eslintrc.json`

### Java
- **Checkstyle**: Java 코드 포맷팅 및 스타일 검증
- 출력: `.sym/checkstyle.xml`
- **PMD**: Java 코드 품질 검증 및 코드 스멜 감지
- 출력: `.sym/pmd-ruleset.xml`

### 향후 지원 예정
- **SonarQube**: 다중 언어 정적 분석
- **LLM Validator**: 전통적인 linter로 표현할 수 없는 커스텀 규칙

## 엔진 할당

`code-policy.json`의 각 규칙에는 검증 도구를 지정하는 `engine` 필드가 있습니다:

- `eslint`: ESLint 설정으로 변환된 규칙
- `checkstyle`: Checkstyle 모듈로 변환된 규칙
- `pmd`: PMD 규칙셋으로 변환된 규칙
- `sonarqube`: 향후 지원 예정
- `llm-validator`: LLM 분석이 필요한 복잡한 규칙

## 예제 워크플로우

1. `user-policy.json`에 **컨벤션 정의**
2. linter 설정으로 **변환**:
   ```bash
   sym convert -i user-policy.json --targets eslint,checkstyle,pmd
   ```
3. Git 변경사항에 **linter 실행**:
   ```bash
   # JavaScript/TypeScript
   eslint --config .sym/.eslintrc.json src/**/*.{js,ts}

   # Java
   checkstyle -c .sym/checkstyle.xml src/**/*.java
   pmd check -R .sym/pmd-ruleset.xml -d src/
   ```

## 코드 정책 스키마

생성된 `code-policy.json` 내용:
```json
{
  "version": "1.0.0",
  "rules": [
    {
      "id": "naming-class-pascalcase",
      "engine": "eslint",
      "check": {...}
    },
    {
      "id": "security-no-secrets",
      "engine": "llm-validator",
      "check": {...}
    }
  ]
}
```

`engine: "llm-validator"` 규칙은 전통적인 linter로 체크할 수 없으며 커스텀 LLM 기반 검증이 필요합니다.

## 테스트

### 통합 테스트 실행

```bash
# 모든 통합 테스트
go test ./tests/integration/... -v

# 특정 엔진 테스트
go test ./tests/integration/... -v -run TestPatternEngine
go test ./tests/integration/... -v -run TestLengthEngine
go test ./tests/integration/... -v -run TestStyleEngine
go test ./tests/integration/... -v -run TestASTEngine
go test ./tests/integration/... -v -run TestTypeChecker
```
