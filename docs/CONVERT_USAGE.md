# Convert 명령어 사용 가이드

## 빠른 시작

자연어 규칙을 linter 설정으로 변환:

```bash
# 모든 지원 linter로 변환 (출력: <git-root>/.sym)
sym convert -i user-policy.json --targets all

# JavaScript/TypeScript만
sym convert -i user-policy.json --targets eslint

# Java만
sym convert -i user-policy.json --targets checkstyle,pmd
```

## 기본 출력 디렉토리

**중요**: convert 명령어는 Git 저장소 루트에 `.sym` 디렉토리를 자동 생성하고 모든 파일을 저장합니다.

### 디렉토리 구조

```
your-project/
+-- .git/
+-- .sym/                      # 자동 생성
|   +-- .eslintrc.json        # ESLint 설정
|   +-- checkstyle.xml        # Checkstyle 설정
|   +-- pmd-ruleset.xml       # PMD 설정
|   +-- code-policy.json      # 내부 정책
|   +-- conversion-report.json # 변환 리포트
+-- src/
+-- user-policy.json          # 입력 파일
```

### 왜 .sym인가?

- **일관된 위치**: 항상 Git 루트에 있어 찾기 쉬움
- **버전 관리**: `.gitignore`에 추가하여 생성 파일을 Git에서 제외 가능
- **CI/CD 친화적**: 스크립트가 항상 `<git-root>/.sym`에서 설정 찾음

### 커스텀 출력 디렉토리

다른 위치가 필요한 경우:

```bash
sym convert -i user-policy.json --targets all --output-dir ./custom-dir
```

## 사전 요구사항

1. **Git 저장소**: Git 저장소 내에서 명령어 실행
2. **OpenAI API 키** (선택): 더 나은 추론을 위해 `OPENAI_API_KEY` 설정
   ```bash
   export OPENAI_API_KEY=sk-...
   ```

API 키 없이도 폴백 패턴 매칭 사용 (정확도 낮음)

## 사용자 정책 파일

자연어 규칙으로 `user-policy.json` 작성:

```json
{
  "version": "1.0.0",
  "defaults": {
    "severity": "error",
    "autofix": false
  },
  "rules": [
    {
      "say": "클래스 이름은 PascalCase여야 합니다",
      "category": "naming",
      "languages": ["javascript", "typescript", "java"]
    },
    {
      "say": "한 줄은 최대 100자입니다",
      "category": "length"
    },
    {
      "say": "들여쓰기는 4칸 공백을 사용합니다",
      "category": "style"
    }
  ]
}
```

## 명령어 옵션

### 기본 옵션

- `-i, --input`: 입력 사용자 정책 파일 (기본값: `user-policy.json`)
- `--targets`: 타겟 linter (쉼표 구분 또는 `all`)
  - `eslint` - JavaScript/TypeScript
  - `checkstyle` - Java
  - `pmd` - Java
  - `all` - 모든 지원 linter

### 고급 옵션

- `--output-dir`: 커스텀 출력 디렉토리 (기본값: `<git-root>/.sym`)
- `--openai-model`: OpenAI 모델 (기본값: `gpt-4o`)
- `--confidence-threshold`: 최소 신뢰도 (기본값: `0.7`)
  - 범위: 0.0 ~ 1.0
  - 낮은 값 = 더 많은 규칙 변환, 더 많은 경고
- `--timeout`: API 타임아웃 초 (기본값: `30`)
- `-v, --verbose`: 상세 로깅 활성화

### 레거시 모드

내부 `code-policy.json`만 생성:

```bash
sym convert -i user-policy.json -o code-policy.json
```

## 예제 워크플로우

### JavaScript/TypeScript 프로젝트

```bash
# 1. user-policy.json 작성
cat > user-policy.json <<EOF
{
  "rules": [
    {
      "say": "클래스 이름은 PascalCase여야 합니다",
      "category": "naming"
    },
    {
      "say": "문자열에는 작은따옴표를 사용합니다",
      "category": "style"
    }
  ]
}
EOF

# 2. ESLint 설정으로 변환
sym convert -i user-policy.json --targets eslint

# 3. 생성된 설정 사용
npx eslint --config .sym/.eslintrc.json src/
```

### Java 프로젝트

```bash
# 1. Java 규칙으로 user-policy.json 작성
sym convert -i user-policy.json --targets checkstyle,pmd

# 2. Checkstyle 사용
java -jar checkstyle.jar -c .sym/checkstyle.xml src/

# 3. PMD 사용
pmd check -R .sym/pmd-ruleset.xml -d src/
```

### 다중 언어 프로젝트

```bash
# 모든 언어를 한 번에 변환
sym convert -i user-policy.json --targets all

# 생성되는 파일:
# - .sym/.eslintrc.json (JS/TS)
# - .sym/checkstyle.xml (Java)
# - .sym/pmd-ruleset.xml (Java)
```

## 출력 파일

### 생성 파일

1. **`.eslintrc.json`**: ESLint 설정
   - 규칙: naming, length, style, complexity
   - 원본 "say" 텍스트를 주석으로 포함

2. **`checkstyle.xml`**: Checkstyle 설정
   - 모듈: TypeName, LineLength, Indentation 등
   - params에서 속성 설정

3. **`pmd-ruleset.xml`**: PMD 규칙셋
   - PMD 카테고리에 대한 규칙 참조
   - severity에서 priority 매핑

4. **`code-policy.json`**: 내부 검증 정책
   - `sym validate` 명령어에서 사용
   - 구조화된 규칙 정의

5. **`conversion-report.json`**: 변환 리포트
   - linter별 통계
   - 경고 및 에러
   - 신뢰도 점수

### 변환 리포트

리포트를 통해 변환 결과 확인:

```json
{
  "timestamp": "2025-10-30T21:13:00+09:00",
  "total_rules": 5,
  "targets": ["eslint", "checkstyle", "pmd"],
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

## 신뢰도와 경고

### 신뢰도 점수

규칙에 신뢰도 점수 (0.0-1.0) 부여:
- **높음 (0.8-1.0)**: 강한 매칭, 규칙이 잘 작동
- **중간 (0.6-0.8)**: 좋은 매칭, 조정 필요할 수 있음
- **낮음 (0.4-0.6)**: 불확실, 생성된 규칙 확인 필요
- **매우 낮음 (0.0-0.4)**: 패턴 매칭 폴백만 사용

### 낮은 신뢰도 경고

규칙의 신뢰도가 낮을 때:
```
warning: eslint: Rule 2: low confidence (0.40 < 0.70): 한 줄은 최대 100자입니다
```

**대응 방법:**
1. 설정 파일에서 생성된 규칙 검토
2. 필요 시 수동 조정
3. 사용자 규칙에서 더 구체적인 `category`와 `params` 제공
4. 더 나은 정확도를 위해 폴백 대신 OpenAI API 사용

### 임계값 조정

```bash
# 더 엄격하게 (경고 줄이기, 일부 규칙 누락 가능)
sym convert -i user-policy.json --targets all --confidence-threshold 0.8

# 더 관대하게 (경고 많아지고 규칙 많아짐)
sym convert -i user-policy.json --targets all --confidence-threshold 0.5
```

## 문제 해결

### "not in a git repository"

**문제**: Git 저장소 오류로 명령어 실패

**해결**:
```bash
# Git 초기화
git init

# 또는 커스텀 출력 디렉토리 사용
sym convert --targets all --output-dir ./linter-configs
```

### "OPENAI_API_KEY not set"

**문제**: API 키 누락 경고

**해결**:
```bash
# API 키 설정 (권장)
export OPENAI_API_KEY=sk-your-key-here

# 또는 폴백 모드 수용 (정확도 낮음)
# 변환은 작동하지만 패턴 매칭만 사용
```

### 생성된 규칙이 작동하지 않음

**문제**: Linter가 생성된 설정을 거부

**해결**:
1. Linter 버전 호환성 확인
2. conversion-report.json에서 에러 확인
3. 문제가 되는 규칙 수동 조정
4. user-policy.json과 함께 이슈 보고

### 변환이 느림

**문제**: 변환에 너무 오래 걸림

**해결**:
```bash
# 타임아웃 늘리기
sym convert --targets all --timeout 60

# 더 빠른 모델 사용
sym convert --targets all --openai-model gpt-4o

# 또는 규칙을 작은 배치로 분할
```

## CI/CD 통합

### GitHub Actions

```yaml
name: Validate Code Conventions

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Convert policies
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          ./sym convert -i user-policy.json --targets all

      - name: Run ESLint
        run: npx eslint --config .sym/.eslintrc.json src/

      - name: Run Checkstyle
        run: java -jar checkstyle.jar -c .sym/checkstyle.xml src/
```

### GitLab CI

```yaml
validate:
  script:
    - ./sym convert -i user-policy.json --targets all
    - npx eslint --config .sym/.eslintrc.json src/
  only:
    - merge_requests
```

## 팁과 모범 사례

### 더 나은 규칙 작성

1. **구체적으로**: "클래스 이름은 PascalCase여야 합니다" vs "좋은 이름 사용"
2. **params 포함**: `params` 필드에 숫자 값 제공
3. **category 설정**: 올바른 엔진 타입 선택에 도움
4. **언어 지정**: 필요 시 특정 언어 타겟팅

### 생성된 파일 관리

```bash
# .gitignore에 추가
echo ".sym/" >> .gitignore

# user-policy.json은 커밋
git add user-policy.json
git commit -m "Add coding conventions policy"
```

### 설정 공유

```bash
# 팀과 공유
git add .sym/*.{json,xml}
git commit -m "Add generated linter configs"

# 또는 각 머신에서 재생성
# (각 개발자가 실행: sym convert -i user-policy.json --targets all)
```

### 규칙 업데이트

```bash
# 1. user-policy.json 편집
# 2. 설정 재생성
sym convert -i user-policy.json --targets all

# 3. 변경사항 검토
git diff .sym/

# 4. 프로젝트에 적용
npx eslint --config .sym/.eslintrc.json src/
```

## 다음 단계

- [전체 기능 문서](CONVERT_FEATURE.md)
- [기여 가이드](../AGENTS.md)
