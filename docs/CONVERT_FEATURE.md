# Convert 기능 - 다중 타겟 Linter 설정 생성기

## 개요

`convert` 명령어는 자연어 코딩 컨벤션을 LLM 기반 추론을 통해 linter별 설정 파일로 변환합니다.

## 기능

- **LLM 기반 추론**: OpenAI API를 사용하여 자연어 규칙을 분석
- **다중 타겟 지원**: 여러 linter 설정을 동시에 생성
  - **ESLint** (JavaScript/TypeScript)
  - **Checkstyle** (Java)
  - **PMD** (Java)
- **폴백 메커니즘**: LLM 사용 불가 시 패턴 기반 추론
- **신뢰도 점수**: 추론 신뢰도를 추적하고 낮은 신뢰도에 대해 경고
- **1:N 규칙 매핑**: 하나의 사용자 규칙이 여러 linter 규칙을 생성
- **캐싱**: 추론 결과를 캐싱하여 API 호출 최소화

## 아키텍처

```
User Policy (user-policy.json)
       |
   Converter
       |
   LLM Inference (OpenAI API) <- Fallback (Pattern Matching)
       |
   Rule Intent Detection
       |
   Linter Converters
       +-- ESLint Converter -> .eslintrc.json
       +-- Checkstyle Converter -> checkstyle.xml
       +-- PMD Converter -> pmd-ruleset.xml
```

### 패키지 구조

```
internal/
+-- llm/
|   +-- client.go          # OpenAI API 클라이언트
|   +-- inference.go       # 규칙 추론 엔진
|   +-- types.go           # Intent 및 결과 타입
+-- converter/
|   +-- converter.go       # 메인 변환 로직
|   +-- linters/
|       +-- converter.go   # Linter 변환기 인터페이스
|       +-- registry.go    # 변환기 레지스트리
|       +-- eslint.go      # ESLint 변환기
|       +-- checkstyle.go  # Checkstyle 변환기
|       +-- pmd.go         # PMD 변환기
+-- cmd/
    +-- convert.go         # CLI 명령어
```

## 사용법

### 기본 사용

```bash
# 모든 지원 linter로 변환
sym convert -i user-policy.json --targets all --output-dir .linters

# JavaScript/TypeScript만
sym convert -i user-policy.json --targets eslint --output-dir .linters

# Java만
sym convert -i user-policy.json --targets checkstyle,pmd --output-dir .linters
```

### 고급 옵션

```bash
# 특정 OpenAI 모델 사용
sym convert -i user-policy.json \
  --targets all \
  --output-dir .linters \
  --openai-model gpt-4o

# 신뢰도 임계값 조정
sym convert -i user-policy.json \
  --targets eslint \
  --output-dir .linters \
  --confidence-threshold 0.8

# 상세 출력 활성화
sym convert -i user-policy.json \
  --targets all \
  --output-dir .linters \
  --verbose
```

### 레거시 모드

```bash
# 내부 code-policy.json만 생성 (linter 설정 없음)
sym convert -i user-policy.json -o code-policy.json
```

## 설정

### 환경 변수

- `OPENAI_API_KEY`: OpenAI API 키 (LLM 추론에 필요)
  - 미설정 시 폴백 패턴 기반 추론 사용

### 플래그

- `--targets`: 타겟 linter (쉼표로 구분 또는 "all")
- `--output-dir`: 생성 파일 출력 디렉토리
- `--openai-model`: 사용할 OpenAI 모델 (기본값: gpt-4o)
- `--confidence-threshold`: 추론 최소 신뢰도 (기본값: 0.7)
- `--timeout`: API 호출 타임아웃 초 (기본값: 30)
- `--verbose`: 상세 로깅 활성화

## 사용자 정책 스키마

### 예시

```json
{
  "version": "1.0.0",
  "defaults": {
    "severity": "error",
    "autofix": false
  },
  "rules": [
    {
      "id": "naming-class-pascalcase",
      "say": "클래스 이름은 PascalCase여야 합니다",
      "category": "naming",
      "languages": ["javascript", "typescript", "java"],
      "params": {
        "case": "PascalCase"
      }
    },
    {
      "id": "length-max-line",
      "say": "한 줄은 최대 100자입니다",
      "category": "length",
      "params": {
        "max": 100
      }
    }
  ]
}
```

### 지원 카테고리

- `naming`: 식별자 네이밍 컨벤션
- `length`: 크기 제약 (라인/파일/함수 길이)
- `style`: 코드 포맷팅 (들여쓰기, 따옴표, 세미콜론)
- `complexity`: 순환/인지 복잡도
- `security`: 보안 관련 규칙
- `error_handling`: 예외 처리 패턴
- `dependency`: import/의존성 제한

### 지원 엔진 타입

- `pattern`: 네이밍 컨벤션, 금지 패턴, import 제한
- `length`: 라인/파일/함수 길이, 파라미터 수
- `style`: 들여쓰기, 따옴표, 세미콜론, 공백
- `ast`: 순환 복잡도, 중첩 깊이
- `custom`: 기타 카테고리에 맞지 않는 규칙

## 출력 파일

### 생성 파일

1. **`.eslintrc.json`**: JavaScript/TypeScript용 ESLint 설정
2. **`checkstyle.xml`**: Java용 Checkstyle 설정
3. **`pmd-ruleset.xml`**: Java용 PMD 규칙셋
4. **`code-policy.json`**: 내부 검증 정책
5. **`conversion-report.json`**: 상세 변환 리포트

### 변환 리포트 형식

```json
{
  "timestamp": "2025-10-30T19:52:22+09:00",
  "input_file": "user-policy.json",
  "total_rules": 5,
  "targets": ["eslint", "checkstyle", "pmd"],
  "openai_model": "gpt-4o",
  "confidence_threshold": 0.7,
  "linters": {
    "eslint": {
      "rules_generated": 5,
      "warnings": 2,
      "errors": 0
    }
  },
  "warnings": [
    "eslint: Rule 2: low confidence (0.40 < 0.70): 한 줄은 최대 100자입니다"
  ]
}
```

## LLM 추론

### 동작 방식

1. **캐시 확인**: 먼저 규칙이 이전에 추론되었는지 확인
2. **LLM 분석**: 구조화된 프롬프트와 함께 OpenAI API로 규칙 전송
3. **Intent 추출**: JSON 응답을 파싱하여 추출:
   - 엔진 타입 (pattern/length/style/ast)
   - 카테고리 (naming/security 등)
   - 타겟 (identifier/content/import)
   - 범위 (line/file/function)
   - 파라미터 (max, min, case 등)
   - 신뢰도 점수 (0.0-1.0)
4. **폴백**: LLM 실패 시 패턴 매칭 사용
5. **변환**: Intent를 linter별 규칙으로 매핑

### 폴백 추론

LLM 사용 불가 시 패턴 기반 규칙이 감지:
- **네이밍 규칙**: "PascalCase", "camelCase", "name" 등의 키워드
- **길이 규칙**: "line", "length", "max", "characters" 등의 키워드
- **스타일 규칙**: "indent", "spaces", "tabs", "quote" 등의 키워드
- **보안 규칙**: "secret", "password", "hardcoded" 등의 키워드
- **import 규칙**: "import", "dependency", "layer" 등의 키워드

## 테스트

### 단위 테스트

```bash
# 모든 테스트 실행
go test ./...

# LLM 추론 테스트
go test ./internal/llm/...

# Linter 변환기 테스트
go test ./internal/converter/linters/...
```

## 제한사항

### ESLint
- 복잡한 AST 패턴 지원 제한
- 일부 규칙은 커스텀 ESLint 플러그인 필요
- 스타일 규칙이 Prettier와 충돌 가능

### Checkstyle
- 모듈 설정이 복잡할 수 있음
- 일부 규칙은 추가 체크 필요
- 커스텀 패턴 지원 제한

### PMD
- 규칙 참조는 PMD 버전과 일치해야 함
- 속성 설정이 규칙마다 다름
- 일부 카테고리의 커버리지 제한

### LLM 추론
- OpenAI API 키 필요 (비용 발생)
- 복잡한 규칙에 대해 잘못된 해석 가능
- 신뢰도 점수는 추정치
- 네트워크 의존성 및 지연

## 성능

### 벤치마크 (5개 규칙, 캐시 없음)

- **LLM 사용 (gpt-4o)**: 약 5-10초
- **폴백만**: 1초 미만
- **캐시 적용**: 100ms 미만

### 비용 추정

- **gpt-4o**: 규칙당 약 $0.001
- **캐싱**: 반복 규칙에 대해 비용 약 90% 절감

## 기여

새 linter 지원 추가 시:

1. `internal/converter/linters/`에 `LinterConverter` 인터페이스 구현
2. `init()` 함수에서 변환기 등록
3. `*_test.go`에 테스트 추가
4. 이 문서 업데이트

## 라이선스

sym-cli 프로젝트 라이선스와 동일
