# LLM Validator

## Overview

LLM Validator는 전통적인 linter로 검증할 수 없는 복잡한 코딩 규칙을 LLM을 사용해 검증하는 도구입니다.

## 사용 목적

일부 코딩 컨벤션은 정적 분석 도구로 검사하기 어렵습니다:

- **보안 규칙**: "하드코딩된 API 키나 비밀번호를 사용하지 마세요"
- **아키텍처 규칙**: "레이어 간 의존성을 준수하세요"
- **복잡도 규칙**: "순환 복잡도를 10 이하로 유지하세요"
- **비즈니스 로직**: "결제 로직에는 항상 로깅을 포함하세요"

이러한 규칙들은 `code-policy.json`에서 `engine: "llm-validator"`로 표시됩니다.

## 작동 방식

1. **Git 변경사항 읽기**:
   - 현재 unstaged 또는 staged 변경사항을 읽습니다
   - 추가된 라인만 추출합니다

2. **LLM 검증**:
   - `engine: "llm-validator"`인 각 규칙에 대해
   - 변경된 코드를 LLM에 전달
   - 규칙 위반 여부를 확인

3. **결과 리포트**:
   - 위반사항을 발견하면 상세 정보 출력
   - 수정 제안 포함

## 사용 방법

### 기본 사용

```bash
# Unstaged 변경사항 검증
sym validate

# Staged 변경사항 검증
sym validate --staged

# 커스텀 policy 파일 사용
sym validate --policy custom-policy.json
```

### 예시 워크플로우

1. 코드 변경
```bash
echo 'const API_KEY = "sk-1234567890"' >> src/config.js
```

2. 검증 실행
```bash
sym validate
```

3. 결과 확인
```
Validating unstaged changes...
Found 1 changed file(s)

=== Validation Results ===
Checked: 2
Passed:  1
Failed:  1

Found 1 violation(s):

1. [error] security-no-hardcoded-secrets
   File: src/config.js
   Hardcoded API key detected | Suggestion: Use environment variables

Error: found 1 violation(s)
```

## 설정

### 환경 변수

- `OPENAI_API_KEY`: OpenAI API 키 (필수)

### 플래그

- `--policy, -p`: code-policy.json 경로 (기본: .sym/code-policy.json)
- `--staged`: staged 변경사항 검증
- `--model`: OpenAI 모델 (기본: gpt-4o-mini)
- `--timeout`: 규칙당 타임아웃 (초, 기본: 30)

## 통합

### Pre-commit Hook

`.git/hooks/pre-commit`:
```bash
#!/bin/bash
sym validate --staged
```

### CI/CD

```yaml
# GitHub Actions
- name: Validate Code Conventions
  run: |
    sym validate --staged
```

## 제한사항

- LLM API 호출 비용 발생
- 네트워크 연결 필요
- 응답 시간이 정적 분석보다 느림

## 최적화 팁

1. **빠른 피드백을 위해 전통적인 linter와 함께 사용**:
   - ESLint, Checkstyle, PMD로 검증 가능한 규칙은 해당 도구 사용
   - LLM validator는 복잡한 규칙에만 사용

2. **변경사항이 많을 때는 주의**:
   - 큰 PR의 경우 API 비용이 증가할 수 있음
   - `--staged`를 사용해 커밋 단위로 검증 권장

3. **적절한 규칙 설정**:
   - 너무 주관적인 규칙은 피하기
   - 명확하고 구체적인 규칙 작성

## 예시 규칙

`code-policy.json`:
```json
{
  "rules": [
    {
      "id": "security-no-secrets",
      "enabled": true,
      "category": "security",
      "severity": "error",
      "desc": "Do not hardcode secrets, API keys, or passwords",
      "check": {
        "engine": "llm-validator",
        "desc": "Do not hardcode secrets, API keys, or passwords"
      }
    },
    {
      "id": "architecture-layer-dependency",
      "enabled": true,
      "category": "architecture",
      "severity": "error",
      "desc": "Presentation layer should not directly access data layer",
      "check": {
        "engine": "llm-validator",
        "desc": "Presentation layer should not directly access data layer"
      }
    }
  ]
}
```
