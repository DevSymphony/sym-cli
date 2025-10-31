# End-to-End Test Scenario

## 개요

이 시나리오는 LLM 기반 코드 검증 기능을 테스트하기 위한 엔드투엔드 테스트입니다.

## 파일 구성

```
tests/scenario/
├── .sym/
│   └── code-policy.json          # 자연어로 작성된 코딩 규칙 (10개)
├── bad_code.go                    # 10가지 위반사항이 있는 코드
├── good_code.go                   # 모든 규칙을 준수하는 코드
└── good_code_test.go              # Table-driven test 예시
```

## 테스트 시나리오

### 1. 사전 준비

```bash
# OpenAI API 키 설정
export OPENAI_API_KEY="your-api-key"

# Symphony CLI 빌드 (프로젝트 루트에서)
go build -o sym ./cmd/sym
```

### 2. 위반 코드 검증 (실패 예상)

```bash
# bad_code.go를 staged 상태로 만들기
git add tests/scenario/bad_code.go

# 검증 실행
./sym validate --staged --policy tests/scenario/.sym/code-policy.json

# 예상 결과: 10개의 위반사항 검출
```

### 3. 정상 코드 검증 (통과 예상)

```bash
# 이전 staging 취소
git restore --staged tests/scenario/bad_code.go

# good_code.go를 staged 상태로
git add tests/scenario/good_code.go tests/scenario/good_code_test.go

# 검증 실행
./sym validate --staged --policy tests/scenario/.sym/code-policy.json

# 예상 결과: ✓ All checks passed!
```

## 검출되어야 할 위반사항 (bad_code.go)

### 1. 보안: 하드코딩된 API 키
- **위치**: Line 10
- **코드**: `const APIKey = "sk-1234567890..."`
- **규칙**: "API 키나 비밀번호를 코드에 하드코딩하면 안됩니다"

### 2. 보안: SQL Injection 취약점
- **위치**: Line 46
- **코드**: `query := "INSERT INTO users ... VALUES ('" + username + "', '" + email + "')"`
- **규칙**: "SQL 쿼리에 사용자 입력을 직접 연결하면 안됩니다"

### 3. 아키텍처: HTTP 핸들러에서 직접 DB 접근
- **위치**: Line 33-50
- **코드**: `HandleCreateUser` 함수 내에서 `db.Query()` 직접 호출
- **규칙**: "데이터베이스 접근은 반드시 repository 패턴을 통해서만"

### 4. 에러 처리: Panic 사용
- **위치**: Lines 17, 40
- **코드**: `panic("negative amount not allowed")`, `panic(err)`
- **규칙**: "프로덕션 코드에서 panic()을 사용하면 안됩니다"

### 5. 에러 처리: 에러 무시
- **위치**: Line 49
- **코드**: `db.Exec(query)` (에러 체크 안함)
- **규칙**: "에러를 반환하는 함수를 호출할 때는 반드시 에러를 체크"

### 6. 코드 품질: Magic Numbers
- **위치**: Lines 20, 58-72
- **코드**: `10000`, `50`, `20`, `10`, `5`, `300`
- **규칙**: "0과 1을 제외한 숫자 리터럴은 의미있는 상수명으로"

### 7. 코드 품질: 과도한 중첩
- **위치**: Lines 55-67
- **코드**: 4단계 중첩 for-if 구조
- **규칙**: "함수는 3단계 이상의 중첩된 제어 구조를 가지면 안됩니다"

### 8. 아키텍처: 함수 내부 의존성 생성
- **위치**: Line 36
- **코드**: `db, err := sql.Open(...)`
- **규칙**: "함수 내부에서 의존성을 직접 생성하면 안됩니다"

### 9. 문서화: Godoc 누락
- **위치**: Line 14
- **코드**: `func ProcessPayment` (주석 없음)
- **규칙**: "모든 exported 함수는 godoc 주석이 있어야 합니다"

### 10. 테스팅: Table-driven test 미사용
- **위치**: 주석으로 표시됨
- **규칙**: "여러 시나리오를 테스트할 때는 table-driven test 패턴"

## 현재 구현된 기능 확인

### ✅ 완료된 기능

1. **validate 명령어**: `sym validate --staged` 동작 확인
2. **Git 통합**: Staged/unstaged changes 추출
3. **LLM Validator**: 자연어 규칙을 LLM으로 검증
4. **Policy 파싱**: code-policy.json 읽기 및 파싱
5. **결과 출력**: 위반사항 포맷팅 및 표시

### 📝 테스트 방법

```bash
# 1. Unit tests (이미 통과)
go test ./internal/validator/... -v

# 2. CLI 빌드 확인
go build -o sym ./cmd/sym
./sym validate --help

# 3. E2E 테스트 (OpenAI API 키 필요)
export OPENAI_API_KEY="sk-..."
git add tests/scenario/bad_code.go
./sym validate --staged --policy tests/scenario/.sym/code-policy.json
```

## 기대 출력 예시

### bad_code.go 검증 시:

```
Validating staged changes...
Found 1 changed file(s)

=== Validation Results ===
Checked: 10
Passed:  0
Failed:  10

Found 10 violation(s):

1. [error] API 키나 비밀번호를 코드에 하드코딩하면 안됩니다
   File: tests/scenario/bad_code.go
   Hardcoded API key detected in constant declaration

2. [error] SQL 쿼리에 사용자 입력을 직접 연결하면 안됩니다
   File: tests/scenario/bad_code.go
   SQL injection vulnerability: string concatenation in query

3. [error] 데이터베이스 접근은 반드시 repository 패턴을 통해서만
   File: tests/scenario/bad_code.go
   Direct database access in HTTP handler

... (계속)
```

### good_code.go 검증 시:

```
Validating staged changes...
Found 2 changed file(s)

=== Validation Results ===
Checked: 10
Passed:  10
Failed:  0

✓ All checks passed!
```

## 추가 테스트 옵션

### 다른 LLM 모델 사용
```bash
./sym validate --staged --model gpt-4
```

### 타임아웃 조정
```bash
./sym validate --staged --timeout 60
```

### Unstaged 변경사항 검증
```bash
./sym validate
```

## 트러블슈팅

### API 키 문제
```bash
echo $OPENAI_API_KEY  # 키 확인
export OPENAI_API_KEY="your-key"
```

### 변경사항 없음
```bash
git status  # 상태 확인
git add <file>  # 파일 staging
```

### 테스트 초기화
```bash
git restore --staged tests/scenario/*.go  # staging 취소
```
