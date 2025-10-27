# Symphony CLI

자연어로 정의된 컨벤션을 검증하는 LLM 친화적 linter

## 개요

Symphony는 개발자들이 자연어로 코드 컨벤션을 정의하고, LLM 기반 코딩 도구들이 작업 시 해당 컨벤션을 자동으로 준수할 수 있도록 지원하는 도구입니다.

기존 linter 도구들이 정적으로 검증 가능한 규칙만 다루는 것과 달리, Symphony는 정성적인 코드 스타일, 아키텍처 레이어 제약, RBAC 정책 등 복잡한 컨벤션을 자연어로 정의하고 검증할 수 있습니다.

## 주요 기능

- 자연어 기반 컨벤션 정의
- 코드 스타일 및 아키텍처 규칙 검증
- RBAC 기반 파일 접근 제어
- JSON 출력을 통한 LLM 도구 연동
- 컨텍스트 기반 컨벤션 추출

## 설치

### 소스에서 빌드

```bash
# 현재 플랫폼용 빌드
make build

# 모든 플랫폼용 빌드 (Linux, macOS, Windows)
make build-all
```

### Go Install

```bash
go install github.com/DevSymphony/sym-cli/cmd/sym@latest
```

### 시스템에 설치

```bash
# 소스 빌드 후 GOPATH/bin에 설치
make install
```

## 사용법

### 1. 컨벤션 정의

`.sym/user-policy.json` 파일에 자연어로 컨벤션을 정의합니다.

```json
{
  "version": "1.0.0",
  "defaults": {
    "languages": ["go", "python"],
    "severity": "error"
  },
  "rules": [
    { "say": "클래스 이름은 PascalCase" },
    { "say": "한 줄은 100자 이하", "params": { "max": 100 } },
    { "say": "domain 레이어에서 UI 모듈을 import할 수 없습니다" }
  ]
}
```

### 2. 정책 변환

자연어 정책을 검증 가능한 형식으로 변환합니다.

```bash
sym convert .sym/user-policy.json -o .sym/policy.json
```

### 3. 코드 검증

작성한 코드가 컨벤션을 준수하는지 검증합니다.

```bash
sym validate .

# 특정 경로 검증
sym validate ./src

# 자동 수정 시도
sym validate . --fix
```

### 4. 컨벤션 내보내기

현재 작업에 필요한 컨벤션만 추출합니다.

```bash
sym export . --context "사용자 인증 기능 추가" --files "src/auth/login.go"
```

### 5. LLM 도구 연동

CLI는 JSON 형식 출력을 지원하여 LLM 도구가 직접 호출할 수 있습니다.

```bash
# JSON 형식으로 검증 결과 출력
sym validate . --format json

# JSON 형식으로 컨벤션 추출
sym export . --format json --context "add authentication"
```

## 프로젝트 구조

```
sym-cli/
├── cmd/sym/              # CLI 진입점
├── internal/
│   ├── cmd/             # Cobra 커맨드 정의
│   ├── policy/          # 정책 로딩/파싱
│   ├── validator/       # 검증 로직
│   └── converter/       # 스키마 변환
├── pkg/
│   └── schema/          # 스키마 타입 정의
├── Makefile
└── README.md
```

## 개발

### 개발 환경 설정

#### DevContainer 사용 (권장)

VS Code와 DevContainer를 사용하면 자동으로 개발 환경이 구성됩니다:

1. VS Code에서 프로젝트 열기
2. "Reopen in Container" 선택
3. 자동으로 Go, 개발 도구 설치 완료

#### 수동 설정

```bash
# 개발 환경 설정 (의존성 다운로드 + 개발 도구 설치)
make setup
```

### 빌드

```bash
make build      # 현재 플랫폼용
make build-all  # 모든 플랫폼용
```

### 테스트

```bash
make test
```

### 코드 품질

```bash
# 포맷팅
make fmt

# 린팅
make lint

# 의존성 정리
make tidy
```

## 컨벤션 스키마

### 사용자 입력 스키마 (A)

사용자가 직접 작성하는 자연어 중심의 간단한 스키마입니다.

- `say`: 자연어로 작성된 규칙 (필수)
- `category`: 규칙 카테고리 (naming, formatting, security 등)
- `languages`: 적용 대상 언어
- `severity`: 심각도 (error, warning, info)
- `params`: 규칙 파라미터

### 검증 스키마 (B)

변환기가 생성하는 정형화된 검증 스키마입니다.

- `check.engine`: 검증 엔진 타입 (pattern, ast, metric 등)
- `when`: 규칙 적용 조건 (selector)
- `remedy`: 자동 수정 설정
- `enforce`: 집행 설정
