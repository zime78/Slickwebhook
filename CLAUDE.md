# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 빌드 및 테스트 명령어

```bash
# 빌드
make build-slack        # Slack Monitor 빌드
make build-email        # Email Monitor 빌드
make build-ai-worker    # AI Worker 빌드
make build-all          # 전체 플랫폼 빌드 (darwin/linux/windows)

# 테스트
make test               # 전체 테스트
make test-cover         # 커버리지 포함 테스트
go test ./internal/handler/... -v  # 단일 패키지 테스트

# 실행
make run-slack          # Slack Monitor 실행
make run-email          # Email Monitor 실행
make run-ai-worker      # AI Worker 실행

# 정리
make clean              # 빌드 파일 정리
```

## 서비스 개요

이 프로젝트는 3개의 독립 서비스로 구성됩니다:

| 서비스 | 설명 | 설정 파일 |
| ------ | ---- | --------- |
| Slack Monitor | Slack 채널 모니터링 → ClickUp 태스크 생성 | config.ini |
| Email Monitor | 이메일 모니터링 → ClickUp 태스크 생성 | config.email.ini |
| AI Worker | ClickUp AI 리스트 모니터링 → Claude Code 자동 실행 | config.email.ini |

## 아키텍처

### Slack/Email Monitor

Slack/Email 채널 모니터링 → ClickUp 태스크 자동 생성 서비스. Clean Architecture 적용.

```text
cmd/
├── slack-monitor/      # Slack Monitor 엔트리포인트
├── email-monitor/      # Email Monitor 엔트리포인트
└── ai-worker/          # AI Worker 엔트리포인트

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

### AI Worker

ClickUp AI 리스트 모니터링 → Claude Code 자동 호출 (4개 병렬) → 완료 후 상태 업데이트 및 Slack 알림

```text
┌─────────────────────────────────────────────────────────────┐
│                      AI Worker Service                       │
├─────────────────────────────────────────────────────────────┤
│  ClickUp Webhook ──→ Webhook Server ──→ 리스트별 라우팅     │
│                              │                               │
│           ┌──────────────────┼──────────────────┐           │
│           ▼                  ▼                  ▼           │
│      Worker 1           Worker 2           Worker 3/4       │
│      (AI_01)            (AI_02)            (AI_03/04)       │
│           │                  │                  │           │
│           ▼                  ▼                  ▼           │
│      Claude Code        Claude Code        Claude Code      │
│      (터미널 1)         (터미널 2)         (터미널 3/4)     │
│           │                  │                  │           │
│           └──────────────────┼──────────────────┘           │
│                              ▼                               │
│                    Hook Server (완료 수신)                   │
│                              │                               │
│              ┌───────────────┴───────────────┐              │
│              ▼                               ▼              │
│      ClickUp 상태 변경              Slack 알림 전송         │
│      ("개발완료")                   (제목, 링크)            │
└─────────────────────────────────────────────────────────────┘
```

```text
internal/
├── aiworker/           # AI Worker 핵심 모듈
│   ├── config.go       # 리스트별 Worker 설정
│   ├── queue.go        # 태스크 큐 (FIFO, 동시성 안전)
│   ├── invoker.go      # Claude Code 호출 (AppleScript)
│   ├── worker.go       # 개별 Worker (리스트 1개 담당)
│   └── manager.go      # Worker 관리자 (4개 병렬)
├── webhook/            # ClickUp Webhook 서버
│   ├── server.go       # HTTP 서버 (포트: 8080)
│   └── handler.go      # 서명 검증, AI 리스트 필터링
├── hookserver/         # Claude Code Hook 수신
│   ├── server.go       # HTTP 서버 (포트: 8081)
│   └── types.go        # 페이로드 타입
└── claudehook/         # Claude Code 설정 관리
    └── manager.go      # ~/.claude/settings.json 관리
```

**핵심 흐름**:

1. ClickUp Webhook → AI 리스트 필터링 → Worker 큐에 추가
2. Worker: 태스크 조회 → issueformatter → 상태 "작업중" → Claude Code 실행
3. Claude Code 완료 → Hook Server → 상태 "개발완료" → Slack 알림

**인터페이스 패턴**: `slack.Client`, `clickup.Client`, `handler.EventHandler`, `history.Store`, `aiworker.ClaudeInvoker` 인터페이스로 테스트 용이성 확보

## AI Worker 사용 가이드

### 1. 설정 (config.email.ini)

```ini
# AI 리스트 설정 (4개 병렬 Worker)
AI_01_LIST_ID=901414115524
AI_01_SRC_PATH=/path/to/project1

AI_02_LIST_ID=901414115581
AI_02_SRC_PATH=/path/to/project2

AI_03_LIST_ID=901414115582
AI_03_SRC_PATH=/path/to/project3

AI_04_LIST_ID=901414115583
AI_04_SRC_PATH=/path/to/project4

# 서버 포트
WEBHOOK_PORT=8080              # ClickUp Webhook 수신
HOOK_SERVER_PORT=8081          # Claude Code 완료 Hook 수신

# 상태명 (ClickUp 커스텀 상태)
AI_STATUS_WORKING=작업중
AI_STATUS_COMPLETED=개발완료

# 완료된 태스크 이동 목표 리스트
AI_COMPLETED_LIST_ID=901413896178
```

### 2. 실행 방법

```bash
# 1) AI Worker 시작
./scripts/start_aiworker.sh

# 2) ngrok으로 Webhook URL 외부 노출 (별도 터미널)
./scripts/setup_ngrok.sh
# → https://xxxx.ngrok-free.app 형태의 URL 생성

# 3) ClickUp에 Webhook 등록 (최초 1회)
curl -X POST "https://api.clickup.com/api/v2/team/{TEAM_ID}/webhook" \
  -H "Authorization: {CLICKUP_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://xxxx.ngrok-free.app/webhook/clickup",
    "events": ["taskCreated", "taskStatusUpdated"]
  }'
```

### 3. 동작 흐름

```text
1. ClickUp AI 리스트에 태스크 등록
      ↓
2. Webhook으로 AI Worker에 이벤트 전달
      ↓
3. 태스크 상태 → "작업중" 변경
      ↓
4. issueformatter로 프롬프트 생성
      ↓
5. Claude Code 실행 (새 터미널, plan 모드)
   - 프롬프트 끝에 "TDD 방식으로 개발 진행." 자동 추가
      ↓
6. Claude Code 완료 시 Stop Hook 발생
      ↓
7. Hook Server가 완료 감지
      ↓
8. 태스크 상태 → "개발완료" 변경
      ↓
9. 완료 리스트로 태스크 이동 (AI_COMPLETED_LIST_ID)
      ↓
10. Slack 알림 전송 (태스크 제목, 링크)
      ↓
11. 터미널 종료 → 다음 태스크 처리
```

### 4. Claude Code Hook 설정

AI Worker 시작 시 자동으로 `~/.claude/settings.json`에 Hook 설정 추가:

```json
{
  "hooks": {
    "Stop": [{
      "matcher": {},
      "hooks": [{
        "type": "command",
        "command": "curl -s -X POST http://localhost:8081/hook/stop ...",
        "timeout": 5000
      }]
    }]
  }
}
```

### 5. 테스트

```bash
# Webhook 테스트
./scripts/test_aiworker_webhook.sh

# Hook Server 테스트
./scripts/test_hook_server.sh

# 로그 확인
./scripts/logs.sh aiworker
```

### 6. 주의사항

- **macOS 전용**: AppleScript로 터미널 제어 (Linux/Windows 미지원)
- **ngrok 필수**: ClickUp Webhook은 외부 URL 필요
- **Claude Code 설치 필수**: `claude` 명령어가 PATH에 있어야 함
- **4개 병렬 제한**: 각 리스트당 1개 Worker, 동시 4개 태스크 처리

## 언어 정책

- 주석, 문서, 로그: **한국어**
- 변수/함수명: 영어
- 커밋 메시지: 한국어 권장 (타입: `기능`, `수정`, `문서`, `리팩터`, `설정`)

## 테스트 규칙

- 패키지: `_test` 접미사 없이 동일 패키지에 작성
- 외부 의존성은 인터페이스로 모킹

## 스크립트 (scripts/)

### 설치 스크립트

| 스크립트 | 설명 |
| -------- | ---- |
| `install_macos.sh` | Slack Monitor macOS 서비스 설치 |
| `install_email_macos.sh` | Email Monitor macOS 서비스 설치 |
| `install_aiworker_macos.sh` | AI Worker macOS 서비스 설치 |

### 시작/중지 스크립트

```bash
# 개발 모드 실행
./scripts/start_slack_monitor.sh
./scripts/start_email_monitor.sh
./scripts/start_aiworker.sh

# 서비스 중지
./scripts/stop_slack_monitor.sh
./scripts/stop_email_monitor.sh
./scripts/stop_aiworker.sh
./scripts/stop_all.sh              # 전체 중지
```

### 관리 스크립트

| 스크립트 | 설명 |
| -------- | ---- |
| `status_all.sh` | 전체 서비스 상태 확인 |
| `logs.sh [service]` | 로그 확인 (slack/email/aiworker/all) |
| `build_all.sh [platform]` | 전체 빌드 (current/darwin/linux/windows/all) |
| `uninstall_all.sh` | 전체 서비스 제거 |
| `setup_ngrok.sh` | ngrok 터널 설정 (AI Worker Webhook용) |

### 테스트 스크립트

| 스크립트 | 설명 |
| -------- | ---- |
| `test_aiworker_webhook.sh` | AI Worker Webhook 테스트 |
| `test_hook_server.sh` | Hook Server (Claude Code Stop) 테스트 |
| `send_slack_test.sh` | Slack 메시지 전송 테스트 |
| `test_clickup_agent_trigger.sh` | ClickUp Agent 트리거 테스트 |

### launchd 설정 파일

| 파일 | 설명 |
| ---- | ---- |
| `com.slickwebhook.monitor.plist` | Slack Monitor launchd 설정 |
| `com.slickwebhook.email.plist` | Email Monitor launchd 설정 |
| `com.slickwebhook.aiworker.plist` | AI Worker launchd 설정 |
