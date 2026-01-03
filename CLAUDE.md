# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 빌드 및 테스트 명령어

```bash
# 빌드
make build              # 현재 플랫폼 빌드
make build-all          # 전체 플랫폼 빌드 (darwin/linux/windows)

# 테스트
make test               # 전체 테스트
make test-cover         # 커버리지 포함 테스트
go test ./internal/handler/... -v  # 단일 패키지 테스트

# 실행
make run                # 개발 실행 (go run)

# 정리
make clean              # 빌드 파일 정리
```

## 아키텍처

Slack 채널 모니터링 → ClickUp 태스크 자동 생성 서비스. Clean Architecture 적용.

```text
cmd/monitor/main.go     # 엔트리포인트 (DI 구성)
internal/
├── config/             # 설정 로더 (config.ini → 환경변수)
├── domain/             # 도메인 모델 (Message, Event)
├── monitor/            # 폴링 기반 모니터 서비스
├── handler/            # 이벤트 핸들러 (Chain of Responsibility)
│   ├── LogHandler      # 로그 출력
│   └── ForwardHandler  # ClickUp 전송 + 히스토리 저장
├── slack/              # Slack API 클라이언트 (인터페이스 분리)
├── clickup/            # ClickUp API 클라이언트 (인터페이스 분리)
└── history/            # 히스토리 저장소 (Store 인터페이스)
```

**핵심 흐름**: `monitor.Service` → 폴링 → `domain.Event` 생성 → `EventHandler.Handle()` 호출

**인터페이스 패턴**: `slack.Client`, `clickup.Client`, `handler.EventHandler`, `history.Store` 인터페이스로 테스트 용이성 확보

## 언어 정책

- 주석, 문서, 로그: **한국어**
- 변수/함수명: 영어
- 커밋 메시지: 한국어 권장 (타입: `기능`, `수정`, `문서`, `리팩터`, `설정`)

## 테스트 규칙

- 패키지: `_test` 접미사 없이 동일 패키지에 작성
- 외부 의존성은 인터페이스로 모킹
