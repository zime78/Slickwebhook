# SlickWebhook

Slack 채널, Email(Gmail) 모니터링과 ClickUp 자동 연동 + **AI 코딩 에이전트 자동화** 도구입니다.

> 📌 **개발 지침**: 모든 문서와 내용은 한국어로 작성합니다.

---

## 🚀 주요 기능

| 기능 | Slack Monitor | Email Monitor | AI Worker |
|------|:-------------:|:-------------:|:---------:|
| 메시지 감지 | ✅ 채널 폴링 | ✅ IMAP 폴링 | ✅ Webhook |
| ClickUp 연동 | ✅ | ✅ | ✅ |
| 히스토리 관리 | ✅ | ✅ (SQLite) | - |
| 발신자 필터 | ✅ 봇 ID | ✅ 이메일 주소 | - |
| Slack 알림 | - | ✅ (선택) | ✅ (완료 시) |
| **AI 에이전트 연동** | - | - | ✅ 자동 실행 |
| 크로스 플랫폼 | ✅ | ✅ | macOS 전용 |

### 🤖 AI Worker 지원 모델

| AI 에이전트 | 지원 | Hook 시스템 | 자동화 수준 |
|-------------|:----:|:-----------:|:-----------:|
| **Claude Code** | ✅ | ✅ 내장 HTTP Hook | ⭐⭐⭐ 완전 자동화 |
| **OpenCode** | ✅ | ✅ 플러그인 이벤트 | ⭐⭐⭐ 완전 자동화 |
| **Ampcode** | ✅ | ⚠️ 프롬프트 기반 | ⭐⭐ 부분 자동화 |

---

## 📐 아키텍처

### Slack/Email Monitor

```
┌─────────────────┐    ┌─────────────────┐
│  Slack Monitor  │    │  Email Monitor  │
│ (slack-monitor) │    │ (email-monitor) │
└────────┬────────┘    └────────┬────────┘
         │                      │
         └──────────┬───────────┘
                    │
         ┌──────────▼──────────┐
         │   Event Handler     │
         │  (공통 이벤트 처리)  │
         └──────────┬──────────┘
                    │
    ┌───────────────┼───────────────┐
    │               │               │
    ▼               ▼               ▼
 ClickUp        History       Slack 알림
 (Task 생성)   (JSON/SQLite)   (Email 전용)
```

### AI Worker

```
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

> 📖 상세 아키텍처는 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)를 참고하세요.

---

## ⚙️ 빠른 시작

### 1. 설정 파일 생성

```bash
# Slack Monitor 설정
cp _config.ini config.ini

# Email Monitor / AI Worker 설정
cp _config.email.ini config.email.ini

# AI Worker 전용 설정 (선택)
cp _config.aiworker.ini config.aiworker.ini
```

> ⚠️ 설정 파일들은 `.gitignore`에 포함되어 Git에 커밋되지 않습니다.

### 2. Slack Monitor

```bash
# 빌드
make build-slack

# 설정 편집 (config.ini)
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_CHANNEL_ID=YOUR_CHANNEL_ID
POLL_INTERVAL=10s
CLICKUP_API_TOKEN=pk_your_token
CLICKUP_LIST_ID=your_list_id

# 실행
./slack-monitor
```

### 3. Email Monitor

```bash
# 빌드
make build-email

# 설정 편집 (config.email.ini)
GMAIL_CLIENT_ID=your-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
GMAIL_USER_EMAIL=your-email@gmail.com
POLL_INTERVAL=30s
FILTER_FROM=jira@atlassian.com
CLICKUP_API_TOKEN=pk_your_token
CLICKUP_LIST_ID=your_list_id

# 실행
./email-monitor
```

> 📧 Gmail OAuth 설정 방법은 [Gmail OAuth 설정 가이드](#-gmail-oauth-설정)를 참고하세요.

### 4. AI Worker (macOS 전용)

```bash
# 빌드
make build-ai-worker

# 설정 편집 (config.aiworker.ini)
AI_01_LIST_ID=901414115524
AI_01_SRC_PATH=/path/to/project1
WEBHOOK_PORT=8080
HOOK_SERVER_PORT=8081

# AI 모델 선택 (claude/opencode/ampcode)
AI_MODEL_TYPE=opencode

# 터미널 타입 (terminal/warp/cmux)
TERMINAL_TYPE=warp

# 실행
./scripts/start_aiworker.sh

# ngrok으로 Webhook URL 외부 노출 (별도 터미널)
./scripts/setup_ngrok.sh
```

> 🤖 AI Worker는 ClickUp AI 리스트의 태스크를 감지하여 선택한 AI 에이전트를 자동 실행합니다.

---

## 🤖 AI 모델 설정

### 지원 AI 에이전트

| 설정값 | AI 에이전트 | 실행 명령 | 특징 |
|--------|-------------|----------|------|
| `claude` | Claude Code | `claude --permission-mode plan` | 가장 안정적, 내장 Hook |
| `opencode` | OpenCode (oh-my-opencode) | `opencode --prompt "..."` | TUI 모드, 병렬 에이전트 |
| `ampcode` | Ampcode (Sourcegraph) | `cat prompt \| amp` | 경량, Hook 미지원 |

### 설정 예시

```ini
# config.aiworker.ini

# AI 모델 선택 (기본: claude)
AI_MODEL_TYPE=opencode

# 터미널 타입 (terminal/warp/cmux, 기본: terminal)
TERMINAL_TYPE=warp
```

### OpenCode 설정 (oh-my-opencode)

OpenCode 사용 시 추가 설정이 필요합니다:

```bash
# oh-my-opencode 설정
~/.config/opencode/oh-my-opencode.json
```

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

### OpenCode Hook 플러그인

AI Worker와 OpenCode 연동을 위한 플러그인이 함께 제공됩니다:

```bash
# 플러그인 위치
~/.config/opencode/plugins/ai-worker-hook.ts

# opencode.json에 플러그인 등록
{
  "plugin": [
    "./plugins/ai-worker-hook.ts"
  ]
}
```

플러그인이 감지하는 이벤트:

- `session.idle`: 세션 완료/대기 → Stop Hook 전송
- `session.error`: 에러 발생 → 에러 알림 전송
- `permission.updated`: 권한 요청 → Plan 모드 Hook 전송

---

## 📦 파일 구조

```text
SlickWebhook/
├── cmd/
│   ├── slack-monitor/         # Slack Monitor 진입점
│   ├── email-monitor/         # Email Monitor 진입점
│   └── ai-worker/             # AI Worker 진입점
├── internal/
│   ├── clickup/               # ClickUp API 클라이언트 (공통)
│   ├── config/                # 설정 로더 (공통)
│   ├── domain/                # 도메인 모델 (공통)
│   ├── handler/               # 이벤트 핸들러 (공통)
│   ├── history/               # 히스토리 저장소 (JSON)
│   ├── store/                 # 처리된 메시지 저장소 (SQLite)
│   ├── monitor/               # Slack 모니터 서비스
│   ├── slack/                 # Slack API 클라이언트
│   ├── emailmonitor/          # Email 모니터 서비스
│   ├── gmail/                 # Gmail IMAP 클라이언트
│   ├── aiworker/              # AI Worker 핵심 모듈
│   │   ├── aimodel/           # AI 모델 핸들러 (NEW)
│   │   │   ├── interface.go   # AIModelHandler 인터페이스
│   │   │   ├── claude.go      # Claude Code 핸들러
│   │   │   ├── opencode.go    # OpenCode 핸들러
│   │   │   └── ampcode.go     # Ampcode 핸들러
│   │   ├── config.go          # Worker 설정
│   │   ├── invoker.go         # AI 도구 실행기
│   │   ├── manager.go         # Worker 관리자
│   │   └── worker.go          # 개별 Worker
│   ├── webhook/               # ClickUp Webhook 서버
│   ├── hookserver/            # Claude Code Hook 수신
│   ├── claudehook/            # Claude Code 설정 관리
│   └── issueformatter/        # 이슈 → AI 프롬프트 변환
├── docs/                      # 문서
│   ├── ARCHITECTURE.md        # 아키텍처 문서
│   └── CONTRIBUTING.md        # 기여 가이드
├── scripts/                   # 유틸리티 스크립트
├── _config.ini                # Slack Monitor 설정 템플릿
├── _config.email.ini          # Email Monitor 설정 템플릿
├── _config.aiworker.ini       # AI Worker 설정 템플릿 (NEW)
├── Makefile                   # 빌드/테스트 명령
└── go.mod
```

---

## 🛠️ Makefile 명령어

### 빌드

| 명령어 | 설명 |
|--------|------|
| `make build-slack` | Slack Monitor 빌드 |
| `make build-email` | Email Monitor 빌드 |
| `make build-ai-worker` | AI Worker 빌드 |
| `make build-all` | 전체 플랫폼 빌드 (darwin/linux/windows) |

### 실행 및 테스트

| 명령어 | 설명 |
|--------|------|
| `make run-slack` | Slack Monitor 실행 |
| `make run-email` | Email Monitor 실행 |
| `make run-ai-worker` | AI Worker 실행 |
| `make test` | 테스트 실행 |
| `make test-cover` | 커버리지 포함 테스트 |

### 서비스 관리 (macOS)

| 명령어 | 설명 |
|--------|------|
| `make install` | macOS 백그라운드 서비스 설치 |
| `make uninstall` | macOS 서비스 제거 |
| `make status` | 서비스 상태 확인 |
| `make restart` | 서비스 재시작 |

---

## 📜 스크립트 (scripts/)

### 설치 스크립트

| 스크립트 | 설명 |
|----------|------|
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
|----------|------|
| `status_all.sh` | 전체 서비스 상태 확인 |
| `logs.sh [service]` | 로그 확인 (slack/email/aiworker/all) |
| `build_all.sh [platform]` | 전체 빌드 (current/darwin/linux/windows/all) |
| `uninstall_all.sh` | 전체 서비스 제거 |
| `setup_ngrok.sh` | ngrok 터널 설정 (AI Worker Webhook용) |

### 테스트 스크립트

| 스크립트 | 설명 |
|----------|------|
| `test_aiworker_webhook.sh` | AI Worker Webhook 테스트 |
| `test_hook_server.sh` | Hook Server (Claude Code Stop) 테스트 |
| `send_slack_test.sh` | Slack 메시지 전송 테스트 |
| `test_clickup_agent_trigger.sh` | ClickUp Agent 트리거 테스트 |

---

## 📋 환경변수

### Slack Monitor (config.ini)

| 변수명 | 필수 | 설명 |
|--------|:----:|------|
| `SLACK_BOT_TOKEN` | ✅ | Slack Bot 토큰 |
| `SLACK_CHANNEL_ID` | ✅ | 모니터링할 채널 ID |
| `POLL_INTERVAL` | | 폴링 간격 (기본: `10s`) |
| `FILTER_BOT_ONLY` | | 봇 메시지만 처리 (`true`/`false`) |
| `ALLOWED_BOT_IDS` | | 허용할 봇 ID (콤마 구분) |

### Email Monitor (config.email.ini)

| 변수명 | 필수 | 설명 |
|--------|:----:|------|
| `GMAIL_CLIENT_ID` | ✅ | Google OAuth Client ID |
| `GMAIL_CLIENT_SECRET` | ✅ | Google OAuth Client Secret |
| `GMAIL_REFRESH_TOKEN` | ✅ | OAuth Refresh Token |
| `GMAIL_USER_EMAIL` | ✅ | 모니터링할 Gmail 주소 |
| `POLL_INTERVAL` | | 폴링 간격 (기본: `30s`) |
| `LOOKBACK_DURATION` | | 시작 시 과거 이메일 조회 기간 (기본: `0`) |
| `RETENTION_DAYS` | | 처리된 이메일 DB 보관 기간 (기본: `90`) |
| `FILTER_FROM` | | 포함할 발신자 (콤마 구분) |
| `FILTER_EXCLUDE` | | 제외할 발신자 (콤마 구분) |
| `FILTER_EXCLUDE_SUBJECT` | | 제외할 제목 키워드 (콤마 구분) |
| `FILTER_LABEL` | | 모니터링할 라벨 (기본: `INBOX`) |

### Slack 알림

| 변수명 | 필수 | 설명 |
|--------|:----:|------|
| `SLACK_NOTIFY_ENABLED` | | Slack 알림 활성화 (`true`/`false`) |
| `SLACK_BOT_TOKEN` | | Slack Bot OAuth 토큰 |
| `SLACK_NOTIFY_CHANNEL` | | 알림 전송 채널 ID |

### 공통 (ClickUp 연동)

| 변수명 | 필수 | 설명 |
|--------|:----:|------|
| `CLICKUP_API_TOKEN` | | ClickUp API 토큰 |
| `CLICKUP_LIST_ID` | | 태스크 생성할 리스트 ID |
| `JIRA_BASE_URL` | | Jira 이슈 링크 생성용 (예: `https://example.atlassian.net`) |
| `HISTORY_MAX_SIZE` | | 히스토리 최대 개수 (기본: `100`) |

### AI Worker (config.aiworker.ini)

| 변수명 | 필수 | 설명 |
|--------|:----:|------|
| `AI_01_LIST_ID` | ✅ | Worker 1 ClickUp 리스트 ID |
| `AI_01_SRC_PATH` | ✅ | Worker 1 프로젝트 경로 |
| `AI_02_LIST_ID` | | Worker 2 ClickUp 리스트 ID |
| `AI_02_SRC_PATH` | | Worker 2 프로젝트 경로 |
| `AI_03_LIST_ID` | | Worker 3 ClickUp 리스트 ID |
| `AI_03_SRC_PATH` | | Worker 3 프로젝트 경로 |
| `AI_04_LIST_ID` | | Worker 4 ClickUp 리스트 ID |
| `AI_04_SRC_PATH` | | Worker 4 프로젝트 경로 |
| `WEBHOOK_PORT` | | Webhook 서버 포트 (기본: `8080`) |
| `HOOK_SERVER_PORT` | | Hook 서버 포트 (기본: `8081`) |
| `AI_STATUS_WORKING` | | 작업중 상태명 (기본: `작업중`) |
| `AI_STATUS_COMPLETED` | | 완료 상태명 (기본: `개발완료`) |
| `AI_COMPLETED_LIST_ID` | | 완료된 태스크 이동 리스트 ID |
| **`AI_MODEL_TYPE`** | | **AI 모델 선택 (`claude`/`opencode`/`ampcode`, 기본: `claude`)** |
| **`TERMINAL_TYPE`** | | **터미널 타입 (`terminal`/`warp`/`cmux`, 기본: `terminal`)** |

---

## 📧 Gmail OAuth 설정

### 1. Google Cloud Console 설정

1. [Google Cloud Console](https://console.cloud.google.com) 접속
2. 프로젝트 생성 또는 선택
3. **APIs & Services** → **Library** → "Gmail API" 활성화
4. **Credentials** → **Create Credentials** → **OAuth client ID**
5. 애플리케이션 유형: **웹 애플리케이션**
6. 승인된 리디렉션 URI 추가:

   ```
   https://developers.google.com/oauthplayground
   ```

### 2. Refresh Token 획득

1. [OAuth 2.0 Playground](https://developers.google.com/oauthplayground/) 접속
2. ⚙️ 설정 → **"Use your own OAuth credentials"** 체크
3. Client ID/Secret 입력
4. 스코프 입력: `https://mail.google.com/`
5. **Authorize APIs** → Google 로그인 → 권한 승인
6. **Exchange authorization code for tokens** 클릭
7. `refresh_token` 값 복사 → `config.email.ini`에 입력

---

## 🔗 참고 문서

- [Slack API - conversations.history](https://api.slack.com/methods/conversations.history)
- [Gmail API - IMAP](https://developers.google.com/gmail/imap)
- [ClickUp API](https://developer.clickup.com/)
- [Claude Code](https://code.claude.ai/)
- [OpenCode](https://opencode.ai/)
- [Ampcode](https://ampcode.com/)
- [oh-my-opencode](https://github.com/code-yeongyu/oh-my-opencode)

---

## 📄 라이선스

이 프로젝트는 개인 사용 목적으로 작성되었습니다.

### 의존성 라이센스

| 패키지 | 라이센스 |
|--------|----------|
| [go-imap](https://github.com/emersion/go-imap) | MIT |
| [go-sasl](https://github.com/emersion/go-sasl) | MIT |
| [slack-go/slack](https://github.com/slack-go/slack) | BSD-2-Clause |
| [go-sqlite3](https://github.com/mattn/go-sqlite3) | MIT |
| [oauth2](https://pkg.go.dev/golang.org/x/oauth2) | BSD-3-Clause |
| [gorilla/websocket](https://github.com/gorilla/websocket) | BSD-2-Clause |
| [cloud.google.com/go](https://github.com/googleapis/google-cloud-go) | Apache-2.0 |
| [lumberjack](https://github.com/natefinch/lumberjack) | MIT |
