# SlickWebhook

Slack 채널 및 Email(Gmail) 모니터링과 ClickUp 자동 연동 도구입니다.

> 📌 **개발 지침**: 모든 문서와 내용은 한국어로 작성합니다.

---

## 🚀 주요 기능

| 기능 | Slack Monitor | Email Monitor |
|------|:-------------:|:-------------:|
| 메시지 감지 | ✅ 채널 폴링 | ✅ IMAP 폴링 |
| ClickUp 연동 | ✅ | ✅ |
| 히스토리 관리 | ✅ | ✅ (SQLite) |
| 발신자 필터 | ✅ 봇 ID | ✅ 이메일 주소 |
| Slack 알림 | - | ✅ (선택) |
| 크로스 플랫폼 | ✅ | ✅ |

---

## 📐 아키텍처

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

> 📖 상세 아키텍처는 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)를 참고하세요.

---

## ⚙️ 빠른 시작

### 1. 설정 파일 생성

```bash
# Slack Monitor 설정
cp _config.ini config.ini

# Email Monitor 설정
cp _config.email.ini config.email.ini
```

> ⚠️ `config.ini`와 `config.email.ini`는 `.gitignore`에 포함되어 있어 Git에 커밋되지 않습니다. 보안을 위해 반드시 템플릿 파일을 복사하여 사용하세요.

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

---

## 📦 파일 구조

```text
SlickWebhook/
├── cmd/
│   ├── slack-monitor/         # Slack Monitor 진입점
│   │   └── main.go
│   └── email-monitor/         # Email Monitor 진입점
│       └── main.go
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
│   └── gmail/                 # Gmail IMAP 클라이언트
├── docs/                      # 문서
│   ├── ARCHITECTURE.md        # 아키텍처 문서
│   └── CONTRIBUTING.md        # 기여 가이드
├── scripts/                   # 유틸리티 스크립트
│   └── com.slickwebhook.monitor.plist  # macOS launchd 설정
├── _config.ini                # Slack Monitor 설정 템플릿
├── _config.email.ini          # Email Monitor 설정 템플릿
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
| `make build-slack-all` | Slack Monitor 전 플랫폼 빌드 |
| `make build-email-all` | Email Monitor 전 플랫폼 빌드 |
| `make build-all` | 모든 플랫폼 빌드 (Slack + Email) |

### 실행 및 테스트

| 명령어 | 설명 |
|--------|------|
| `make run-slack` | Slack Monitor 실행 |
| `make run-email` | Email Monitor 실행 |
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

### Slack 알림 (Email Monitor 전용)

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
- [slack-go/slack SDK](https://github.com/slack-go/slack)
- [emersion/go-imap](https://github.com/emersion/go-imap)

---

## 📄 라이선스

이 프로젝트는 개인 사용 목적으로 작성되었습니다.
