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

# 릴리즈
./scripts/release.sh v1.0.0    # 릴리즈 생성 (GitHub Actions 자동 빌드)

# 정리
make clean              # 빌드 파일 정리
```

## 서비스 개요

이 프로젝트는 3개의 독립 서비스로 구성됩니다:

| 서비스 | 설명 | 설정 파일 |
| ------ | ---- | --------- |
| Slack Monitor | Slack 채널 모니터링 → ClickUp 태스크 생성 | config.ini |
| Email Monitor | 이메일 모니터링 → ClickUp 태스크 생성 | config.email.ini |
| AI Worker | ClickUp AI 리스트 모니터링 → **AI 에이전트 자동 실행** | config.aiworker.ini |

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

ClickUp AI 리스트 모니터링 → AI 에이전트 자동 호출 (4개 병렬) → 완료 후 상태 업데이트 및 Slack 알림

```text
┌─────────────────────────────────────────────────────────────────┐
│                      AI Worker Service                          │
├─────────────────────────────────────────────────────────────────┤
│  ClickUp Webhook ──→ Webhook Server ──→ 리스트별 라우팅         │
│                              │                                   │
│           ┌──────────────────┼──────────────────┐               │
│           ▼                  ▼                  ▼               │
│      Worker 1           Worker 2           Worker 3/4           │
│      (AI_01)            (AI_02)            (AI_03/04)           │
│           │                  │                  │               │
│           ▼                  ▼                  ▼               │
│   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐      │
│   │ Claude Code   │  │   OpenCode    │  │   Ampcode     │      │
│   │ (터미널 1)    │  │  (터미널 2)   │  │  (터미널 3/4) │      │
│   └───────┬───────┘  └───────┬───────┘  └───────┬───────┘      │
│           │                  │                  │               │
│           └──────────────────┼──────────────────┘               │
│                              ▼                                   │
│                    Hook Server (완료 수신)                       │
│                              │                                   │
│              ┌───────────────┴───────────────┐                  │
│              ▼                               ▼                  │
│      ClickUp 상태 변경              Slack 알림 전송             │
│      ("개발완료" + 리스트 이동)     (제목, 링크)                │
└─────────────────────────────────────────────────────────────────┘
```

```text
internal/
├── aiworker/           # AI Worker 핵심 모듈
│   ├── aimodel/        # AI 모델 핸들러 (NEW)
│   │   ├── interface.go   # AIModelHandler 인터페이스
│   │   ├── claude.go      # Claude Code 핸들러
│   │   ├── opencode.go    # OpenCode 핸들러
│   │   └── ampcode.go     # Ampcode 핸들러
│   ├── config.go       # 리스트별 Worker 설정
│   ├── queue.go        # 태스크 큐 (FIFO, 동시성 안전)
│   ├── invoker.go      # AI 에이전트 호출 (AppleScript)
│   ├── worker.go       # 개별 Worker (리스트 1개 담당)
│   └── manager.go      # Worker 관리자 (4개 병렬)
├── webhook/            # ClickUp Webhook 서버
│   ├── server.go       # HTTP 서버 (포트: 8080)
│   └── handler.go      # 서명 검증, AI 리스트 필터링
├── hookserver/         # AI 에이전트 Hook 수신
│   ├── server.go       # HTTP 서버 (포트: 8081)
│   └── types.go        # 페이로드 타입
└── claudehook/         # Claude Code 설정 관리
    └── manager.go      # ~/.claude/settings.json 관리
```

**핵심 흐름**:

1. ClickUp Webhook → AI 리스트 필터링 → Worker 큐에 추가
2. Worker: 태스크 조회 → issueformatter → 상태 "작업중" → AI 에이전트 실행
3. AI 에이전트 완료 → Hook Server → 상태 "개발완료" → Slack 알림

**인터페이스 패턴**: `slack.Client`, `clickup.Client`, `handler.EventHandler`, `history.Store`, `aiworker.ClaudeInvoker` 인터페이스로 테스트 용이성 확보

## AI Worker 사용 가이드

### 1. 지원 AI 에이전트

| AI 에이전트 | 설정값 | Hook 시스템 | 자동화 수준 |
|-------------|--------|-------------|-------------|
| **Claude Code** | `claude` | ✅ 내장 HTTP Hook | ⭐⭐⭐ 완전 자동화 |
| **OpenCode** | `opencode` | ✅ 플러그인 이벤트 | ⭐⭐⭐ 완전 자동화 |
| **Ampcode** | `ampcode` | ⚠️ 프롬프트 기반 | ⭐⭐ 부분 자동화 |

### 2. 설정 (config.aiworker.ini)

```ini
# AI 리스트 설정 (4개 병렬 Worker)
# Worker별로 터미널/AI 모델 개별 설정 가능

# Worker 1: Claude + iTerm2
AI_01_LIST_ID=901414115524
AI_01_SRC_PATH=/path/to/project1
AI_01_TERMINAL_TYPE=iterm2     # 개별 터미널 설정 (선택)
AI_01_AI_MODEL_TYPE=claude     # 개별 AI 모델 설정 (선택)

# Worker 2: OpenCode + Warp
AI_02_LIST_ID=901414115581
AI_02_SRC_PATH=/path/to/project2
AI_02_TERMINAL_TYPE=warp
AI_02_AI_MODEL_TYPE=opencode

# Worker 3: 전역 설정 사용 (개별 설정 생략 시)
AI_03_LIST_ID=901414115582
AI_03_SRC_PATH=/path/to/project3

# Worker 4: Ampcode + Terminal
AI_04_LIST_ID=901414115583
AI_04_SRC_PATH=/path/to/project4
AI_04_AI_MODEL_TYPE=ampcode

# 서버 포트
WEBHOOK_PORT=8080              # ClickUp Webhook 수신
HOOK_SERVER_PORT=8081          # AI 에이전트 완료 Hook 수신

# 상태명 (ClickUp 커스텀 상태)
AI_STATUS_WORKING=작업중
AI_STATUS_COMPLETED=개발완료

# 완료된 태스크 이동 목표 리스트
AI_COMPLETED_LIST_ID=901413896178

# 전역 터미널 타입 (Worker별 설정 없을 때 사용)
# - terminal: macOS 기본 Terminal.app
# - warp: Warp 터미널 (AppleScript 창 타겟팅 미지원)
# - iterm2: iTerm2 (AppleScript 완벽 지원, 세션 이름으로 타겟팅 가능)
TERMINAL_TYPE=terminal

# 전역 AI 모델 타입 (Worker별 설정 없을 때 사용)
# - claude: Claude Code (기본값)
# - opencode: OpenCode (oh-my-opencode)
# - ampcode: Ampcode (Sourcegraph)
AI_MODEL_TYPE=claude
```

### 3. AI 모델별 추가 설정

#### Claude Code

별도 설정 불필요. AI Worker 시작 시 자동으로 `~/.claude/settings.json`에 Hook 설정 추가.

```json
{
  "hooks": {
    "Stop": [{"matcher": {}, "hooks": [{"type": "command", "command": "curl ..."}]}]
  }
}
```

#### OpenCode (oh-my-opencode)

`~/.config/opencode/oh-my-opencode.json` 설정 필요:

```json
{
  "agents": {
    "sisyphus": { "model": "google/antigravity-claude-sonnet-4-5-thinking" },
    "plan": { "model": "google/antigravity-claude-sonnet-4-5-thinking" },
    "explore": { "model": "google/antigravity-gemini-3-flash" }
  },
  "categories": {
    "quick": { "model": "google/antigravity-gemini-3-flash" },
    "visual-engineering": { "model": "google/antigravity-claude-sonnet-4-5-thinking" }
  }
}
```

OpenCode Hook 플러그인 (`~/.config/opencode/plugins/ai-worker-hook.ts`):

```typescript
export const AIWorkerHookPlugin = async ({ project, client, $, directory, worktree }) => {
  return {
    event: async ({ event }) => {
      if (event.type === "session.idle") {
        await $`curl -s -X POST http://localhost:8081/hook/stop -H 'Content-Type: application/json' -d '...'`
      }
    },
  }
}
```

`~/.config/opencode/opencode.json`에 플러그인 등록:

```json
{
  "plugin": ["./plugins/ai-worker-hook.ts"]
}
```

#### Ampcode

별도 Hook 시스템 없음. 프롬프트에 `curl` 명령어가 자동 추가되어 작업 완료 시 직접 호출.

### 4. 실행 방법

#### 바이너리 직접 실행 (권장)

```bash
# 도움말 표시
./ai-worker --help

# 포그라운드 실행
./ai-worker

# 백그라운드 실행
./ai-worker --bg

# 상태 확인
./ai-worker --status

# 종료
./ai-worker --stop
```

#### 스크립트 사용 (대안)

```bash
# 포그라운드 실행 (개발 모드)
./scripts/start_aiworker.sh

# 백그라운드 실행 (운영 모드)
./scripts/start_aiworker.sh --bg
```

#### CLI 옵션

| 옵션 | 설명 |
| ------ | ---- |
| `--help`, `-h` | 도움말 표시 |
| `--version`, `-v` | 버전 정보 표시 |
| `--bg` | 백그라운드로 실행 |
| `--status` | 실행 상태 확인 |
| `--stop` | 실행 중인 프로세스 종료 |

#### 로그 확인

| 명령어 | 설명 |
| ------ | ---- |
| `./ai-worker --status` | 상태 및 로그 경로 확인 |
| `tail -f logs/ai-worker.log` | 로그 실시간 확인 |

**로그 로테이션 설정** (자동 적용):

- 파일 위치: `logs/aiworker.log`
- 최대 크기: 100MB
- 보관 개수: 5개
- 보관 기간: 30일
- 압축: gzip

```bash
# ngrok으로 Webhook URL 외부 노출 (별도 터미널)
./scripts/setup_ngrok.sh
# → https://xxxx.ngrok-free.app 형태의 URL 생성

# ClickUp에 Webhook 등록 (최초 1회)
curl -X POST "https://api.clickup.com/api/v2/team/{TEAM_ID}/webhook" \
  -H "Authorization: {CLICKUP_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://xxxx.ngrok-free.app/webhook/clickup",
    "events": ["taskCreated", "taskStatusUpdated"]
  }'
```

### 5. 동작 흐름

```text
1. ClickUp AI 리스트에 태스크 등록
      ↓
2. Webhook으로 AI Worker에 이벤트 전달
      ↓
3. 태스크 상태 → "작업중" 변경
      ↓
4. issueformatter로 프롬프트 생성
      ↓
5. AI 에이전트 실행 (선택된 모델에 따라)
   - Claude Code: plan 모드로 실행
   - OpenCode: TUI 모드로 실행 (ultrawork 키워드 포함)
   - Ampcode: 파이프로 프롬프트 전달
      ↓
6. AI 에이전트 완료 시 Hook 발생
   - Claude Code: 내장 Stop Hook
   - OpenCode: 플러그인 session.idle 이벤트
   - Ampcode: 프롬프트에 포함된 curl 실행
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

### 6. 테스트

```bash
# Webhook 테스트
./scripts/test_aiworker_webhook.sh

# Hook Server 테스트
./scripts/test_hook_server.sh

# 로그 확인
./scripts/logs.sh aiworker
```

### 7. 주의사항

- **macOS 전용**: AppleScript로 터미널 제어 (Linux/Windows 미지원)
- **ngrok 필수**: ClickUp Webhook은 외부 URL 필요
- **AI 에이전트 설치 필수**: 선택한 AI 에이전트가 PATH에 있어야 함
  - Claude Code: `npm install -g @anthropic-ai/claude`
  - OpenCode: `brew install opencode`
  - Ampcode: `npm install -g @sourcegraph/amp`
- **4개 병렬 제한**: 각 리스트당 1개 Worker, 동시 4개 태스크 처리

### 8. 릴리즈 배포

#### 개발자용: 새 버전 릴리즈

```bash
# 릴리즈 생성 (테스트 → 태그 → GitHub Actions 자동 빌드)
./scripts/release.sh v1.0.0
```

#### 개발자용: 업데이트

```bash
./scripts/update_aiworker.sh           # Git pull + 빌드
./scripts/update_aiworker.sh --release  # GitHub Releases 다운로드
```

---

### 9. 사용자 설치 가이드 (배포 버전)

처음 사용하는 분을 위한 상세 가이드입니다.

#### Step 1: 작업 폴더 생성

```bash
# 홈 디렉토리에 폴더 생성
mkdir -p ~/ai-worker
cd ~/ai-worker
```

#### Step 2: 바이너리 다운로드

**macOS Apple Silicon (M1/M2/M3):**

```bash
curl -L -o ai-worker https://github.com/zime78/SlickWebhook/releases/latest/download/ai-worker-darwin-arm64
chmod +x ai-worker
```

**macOS Intel:**

```bash
curl -L -o ai-worker https://github.com/zime78/SlickWebhook/releases/latest/download/ai-worker-darwin-amd64
chmod +x ai-worker
```

**Linux x86:**

```bash
curl -L -o ai-worker https://github.com/zime78/SlickWebhook/releases/latest/download/ai-worker-linux-amd64
chmod +x ai-worker
```

#### Step 3: 설정 파일 생성

```bash
# 설정 파일 다운로드
curl -L -o config.aiworker.ini https://raw.githubusercontent.com/zime78/SlickWebhook/main/_config.aiworker.ini
```

설정 파일을 열어 필수 항목을 수정합니다:

```bash
nano config.aiworker.ini   # 또는 원하는 편집기 사용
```

> **중요**: `config.aiworker.ini` 파일은 반드시 `ai-worker` 바이너리와 **같은 폴더**에 있어야 합니다.

**필수 설정 항목:**

| 항목 | 설명 |
|------|------|
| `CLICKUP_API_TOKEN` | ClickUp API 토큰 |
| `CLICKUP_TEAM_ID` | ClickUp 팀 ID |
| `SLACK_BOT_TOKEN` | Slack Bot 토큰 |
| `SLACK_NOTIFY_CHANNEL` | Slack 알림 채널 ID |
| `AI_01_LIST_ID` | 모니터링할 ClickUp 리스트 ID |
| `AI_01_SRC_PATH` | AI 에이전트 실행 경로 |
| `AI_MODEL_TYPE` | AI 모델 선택 (claude/opencode/ampcode) |

#### Step 4: 테스트 실행 (포그라운드)

```bash
./ai-worker
```

> 정상 동작하면 "AI Worker 시작..." 메시지가 표시됩니다.
> 종료하려면 `Ctrl+C`를 누르세요.

#### Step 5: 백그라운드 실행 (실제 운영)

```bash
LOG_TO_FILE=1 nohup ./ai-worker > /dev/null 2>&1 &
```

> **명령어 설명:**
>
> - `LOG_TO_FILE=1` : 로그를 파일로 저장 (logs/aiworker.log)
> - `nohup` : 터미널 종료해도 프로세스 유지
> - `> /dev/null 2>&1` : 터미널 출력 숨김 (로그는 파일로 저장됨)
> - `&` : 백그라운드 실행

**로그 확인:**

```bash
tail -f logs/aiworker.log
```

**종료:**

```bash
pkill ai-worker
```

#### Step 6: 업데이트

새 버전이 릴리즈되면:

```bash
# 1) 종료
pkill ai-worker

# 2) 새 버전 다운로드
curl -L -o ai-worker https://github.com/zime78/SlickWebhook/releases/latest/download/ai-worker-darwin-arm64
chmod +x ai-worker

# 3) 재시작
LOG_TO_FILE=1 nohup ./ai-worker > /dev/null 2>&1 &
```

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
| `test_hook_server.sh` | Hook Server (AI 에이전트 Stop) 테스트 |
| `send_slack_test.sh` | Slack 메시지 전송 테스트 |
| `test_clickup_agent_trigger.sh` | ClickUp Agent 트리거 테스트 |

### launchd 설정 파일

| 파일 | 설명 |
| ---- | ---- |
| `com.slickwebhook.monitor.plist` | Slack Monitor launchd 설정 |
| `com.slickwebhook.email.plist` | Email Monitor launchd 설정 |
| `com.slickwebhook.aiworker.plist` | AI Worker launchd 설정 |
