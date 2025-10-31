# 저장소 가이드라인

## 프로젝트 구조 및 모듈 구성
- 명확한 최상위 레이아웃을 사용하세요:
  - `src/` — 애플리케이션 및 라이브러리 코드
  - `tests/` — `src/` 레이아웃을 반영한 자동화 테스트
  - `scripts/` — 개발/빌드/테스트 보조 스크립트
  - `assets/` — 정적 파일(이미지, JSON, 스키마)
  - `docs/` — 설계 노트 및 ADR
- 모듈은 작고 응집력 있게 유지하고, 필요 시 레이어보다 도메인 기준으로 그룹화하세요.

- Go 프로젝트 구조(뱅크샐러드 권장 예시):
  - 루트에 `go.mod` 유지(모듈 경로는 소문자·단수형 권장).
  - 최소 폴더만 사용해 단순성을 유지합니다.
    - `cmd/` — 초기화/의존성 주입 등 실행 진입점
    - `config/` — 환경 변수 및 설정
    - `client/` — 외부 서비스/서드파티 클라이언트
    - `server/` — 서버 구현(미들웨어/핸들러/DB 등)
      - `server/handler/` — RPC/엔드포인트별 핸들러 파일
      - `server/db/mysql/` — 자동 생성된 MySQL 모델 등
      - `server/server.go` — 라우팅/미들웨어/핸들러 배선
  - 필요 시 `internal/`, `api/`, 각 패키지의 `testdata/` 사용 가능하나, 과도한 폴더 분리는 지양합니다.
  - 패키지/파일명은 소문자 단수형, 불필요한 언더스코어/대문자/축약 지양.
  - 파일은 멀티 모듈 형태로 각 기능별로 구성되어 있어야 해

## 빌드, 테스트, 개발 명령어
- 재현성을 위해 Make 타깃 또는 스크립트를 선호하세요:
  - `make dev` 또는 `./scripts/dev` — 자동 리로드로 로컬에서 앱 실행
  - `make test` 또는 `./scripts/test` — 전체 테스트 스위트 실행
  - `make lint` 또는 `./scripts/lint` — 린터/포매터 실행
  - `make build` 또는 `./scripts/build` — 릴리스 아티팩트 생성
- 서비스 사전 요구사항(DB, 큐, 환경 변수)은 `docs/`에 문서화하세요.

- Go 전용 권장 명령(가능하면 Make 타깃으로 래핑):
  - 개발: `go run ./cmd/<app>` 또는 해당 패키지.
  - 테스트: `go test ./... -race -cover`(필요 시 `-shuffle=on`).
  - 린트/정적 분석: `golangci-lint run`, `go vet ./...`.
  - 보안 점검(선택): `govulncheck ./...`.
  - 빌드: `go build -trimpath -ldflags "-s -w" ./cmd/<app>`.

## 코딩 스타일 및 네이밍 규칙
- 들여쓰기: 공백 4칸. 라인 길이: 100.
- 네이밍: 모듈/파일은 `snake_case`, 클래스는 `PascalCase`, 함수/변수는 공개 여부에 따른 MixedCase (앞글자의 대소문자 여부),
- 함수는 약 50줄 이하로 유지하고, 순수하며 테스트 가능한 유닛을 선호하세요.

- Go 스타일/프랙티스(뱅크샐러드 기준):
  - 포맷팅/린트: `gofmt`/`gofumpt` + `goimports` 적용, `golangci-lint`와 `go vet` 필수.
  - 인자 순서: `ctx context.Context`를 항상 맨 앞에 두고, 그 다음 DB/서비스 클라이언트.
    무거운 인자(slice/map)는 앞쪽, 가벼운 인자(userID, now 등)는 뒤쪽에 배치.
  - 임포트 정렬: 표준 라이브러리 / 서드파티 / 사내(로컬) 순으로 세 그룹, 공백 줄로 구분.
    `gci` 또는 `goimports-reviser` 사용 권장.
  - 네이밍: 복수 결과는 `listXxxs`(단수는 `getXxx`) 사용. 모호한 단어(information, details, summary) 지양.
    상수는 camelCase(`defaultPageSize`), SCREAMING_SNAKE_CASE 사용 지양.
    패키지명은 `core/util/helper/common/infra` 등 범용명 피하고, 구체적으로(`parser`, `timeconv`, `testutil`).
  - 파일 내 선언 순서: interface → type → const → var → new func → 공개 리시버 메서드 → 비공개 리시버 메서드 → 공개 함수 → 비공개 함수.
    테스트 함수 네이밍은 `TestPublicFunc`, `Test_privateFunc`, `TestType_Method` 패턴 준수.
  - 에러/패닉: 런타임(요청 처리 중)엔 `panic`/`fatal` 사용 금지. 초기화(main 등)에서만 허용.
    패닉 가능 함수는 `MustXxx` 접두사 사용하고 테스트/초기화에서만 호출.
    서버에는 recovery 미들웨어/인터셉터 체인 구성.
  - 컨텍스트: `context.Background()`를 기본(top-level)로, `context.TODO()`는 미정/미적용 시에만.
    `Context`는 첫 인자(`ctx`)로 전달하고 타임아웃/취소를 전파.
  - 시간/타임존: 함수 인자는 `time.Duration` 사용. 환경값은 조기에 Duration으로 변환.
    타임존은 초기화 시 `time.LoadLocation`으로 로드하여 재사용(`MustLoadKST()`).
  - 문자열/유니코드: 문자열은 `range`로 rune 단위 순회, 길이는 `utf8.RuneCountInString`으로 계산.
  - 동시성: goroutine은 누수 없게 설계, 에러 집계/취소 전파에 `x/sync/errgroup` 활용.
    복수 에러가 필요하면 `go-multierror` 등 사용 고려.
  - 리시버 선택: 작은 불변 타입은 값 리시버, 그 외 포인터 리시버. 리시버 식별자는 짧고 일관되게.
  - 코멘트: Godoc 스타일의 완전한 문장으로 공개 식별자/패키지 문서화.
  - 리플렉션/원숭이패치: 핸들러 경로에서 `reflect` 사용 자제. 사이드 이펙트는 의존성 주입으로 대체.
  - 함수 옵션: 선택적 인자는 Functional Options 패턴 사용.
  - 이니셜리즘: ID, URL, HTTP, JSON, XML 등은 대문자 유지(예: `userID`, `serviceURL`).
  - 인터페이스: 소비자(사용자) 패키지에서 정의하고, 작고 응집력 있게 유지. 불필요한 공개 인터페이스 지양, 가능한 한 구체 타입을 인자/반환값으로 사용.
  - 에러 처리: `fmt.Errorf("%s: %w", op, err)`로 감싸기(wrap) 및 `errors.Is/As` 사용. 센티넬 에러는 `var`로 선언하고 문자열 비교 지양. 에러 메시지는 소문자로 시작, 말미에 구두점 생략.
  - 로깅: 구조적 로깅 사용(zap/zerolog 등). 라이브러리 레이어에선 로깅 최소화하고 에러 반환 선호. 요청 단위 상관키(request-id/user-id) 포함, 비밀/PII는 로그 금지. 레벨/필드 일관성 유지.
  - 컨텍스트 추가 수칙: `Context`를 구조체에 저장하지 않기, nil 컨텍스트 금지, 타입 지정 키 사용, 값은 작게 유지, 데드라인/취소 전파, 선택적 인자 전달용으로 사용 금지.
  - 제로 값: 타입은 제로 값이 안전하고 유용하도록 설계. 생성자(`NewXxx`)는 불변식 설정에 사용하되 기본 사용에 강제하지 않기.

## 테스트 가이드라인
- `tests/`에서 `src/` 구조를 반영하세요(예: `src/foo/service.py` → `tests/foo/test_service.py`).
- Python: pytest(`tests/test_*.py`). Node: Jest/Vitest(`**/*.test.ts`).
- 변경된 코드의 라인 커버리지는 80% 이상을 목표로 하고, 엣지 케이스와 에러 경로를 포함하세요.
- 기본적으로 빠른 단위 테스트를 작성하고, 느림/통합 테스트는 명시적으로 표시하세요.

- Go 테스트 지침(뱅크샐러드 기준):
  - 테이블 주도 테스트 + `t.Run` 서브테스트. 가능하면 `t.Parallel()`로 병렬화.
  - 어설션은 `stretchr/testify`의 `assert`/`require`를 선호. `suite` 패키지는 사용하지 않음.
  - 결정적 테스트: map 순회 의존 로직 회피. JSON 비교는 `assert.JSONEq` 또는 `cmp.Diff` 사용.
    직렬화는 하드코딩 문자열 대신 헬퍼(`mustMarshal`)로 생성.
  - 공용 헬퍼는 `t.Helper()`. 테스트 데이터는 `testdata/` 디렉터리에 보관.
  - 벤치마크: `BenchmarkXxx(b *testing.B)`, 예제: `ExampleXxx()`로 문서/검증 겸용.
  - 커버리지: `go test ./... -race -coverprofile=coverage.out`.
  - 테스트 로깅: 테스트 로그는 캡처하거나 비활성화하여 노이즈 최소화. 필요한 경우 구조를 기준으로 단언.
  - 에러 단언: `require.Error`와 `errors.Is/As`로 센티넬/래핑 에러를 검증.

## 커밋 및 PR 가이드라인
- Conventional Commits 사용: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.
- 커밋은 작고 집중적으로, 명령형 제목과 간단한 근거 설명 본문을 포함하세요.
- PR에는 요약, 관련 이슈 링크, 필요 시 스크린샷/로그, 그리고 영향 항목 체크리스트(마이그레이션, 설정, 문서)를 포함하세요.

## 보안 및 구성
- 시크릿은 절대 커밋하지 마세요. `.env.example`을 사용하고 필요한 변수를 문서화하세요.
- 모든 입력을 검증하고 정제하세요. 파라미터화된 쿼리를 선호하세요.
- 토큰/키의 권한은 최소화하고 정기적으로 교체하세요.

- 네트워크/서버 설정(권장):
  - HTTP 서버: `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`를 합리적으로 설정.
  - HTTP 클라이언트: `Client.Timeout` 설정 및 `Transport` 튜닝(`IdleConnTimeout`, `MaxIdleConns`, `MaxIdleConnsPerHost`). 요청 단위 컨텍스트 데드라인 지정.
  - gRPC: recovery/로깅/메트릭 인터셉터 구성, keepalive/메시지 크기 제한 설정, 호출 측에서 데드라인 강제.
  - 재시도: 멱등(idempotent) 호출에 한해 백오프/지터 포함 재시도. 컨텍스트 취소/데드라인 전파.

- Go 보안/정적 분석:
  - `go vet`, `golangci-lint`, `staticcheck`(선택), `govulncheck`를 CI에 포함.
  - `exec.Command`는 인자 분리로 쉘 인젝션 방지. `http.Server`에 합리적 타임아웃 설정.
  - 크립토는 `crypto/rand` 사용(비밀번호/토큰 등). `math/rand`는 비보안용에 한정.

## 에이전트 전용 메모
- 변경 범위를 좁게 유지하고, 관련 없는 코드 리팩터링은 하지 마세요.
- 검색에는 `rg`를 선호하고, 파일은 250줄 이하의 청크로 읽으세요.
- 새로 생성하는 파일에도 이 문서의 지침을 따르세요.

## 테스트
테스트는 반드시 작성해야 하고 