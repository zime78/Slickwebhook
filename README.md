# SlickWebhook

Slack 채널 모니터링 및 ClickUp 자동 연동 도구입니다.

> 📌 **개발 지침**: 모든 문서와 내용은 한국어로 작성합니다.

---

## 🚀 주요 기능

| 기능 | 설명 |
|------|------|
| Slack 모니터링 | 채널 메시지를 실시간 감지 (폴링 방식) |
| ClickUp 연동 | 새 메시지 감지 시 자동 태스크 생성 |
| 히스토리 관리 | 전송 기록 저장 (최대 100개, 설정 가능) |
| 크로스 플랫폼 | macOS, Linux, Windows 지원 |
| 백그라운드 실행 | macOS launchd 서비스 지원 |

---

## ⚙️ 빠른 시작

### 1. 바이너리 다운로드 또는 빌드

```bash
# 전체 플랫폼 빌드
make build-all

# build 폴더에 생성됨:
# - slack-monitor-macos-apple-silicon
# - slack-monitor-macos-intel
# - slack-monitor-linux-x86
# - slack-monitor-linux-arm
# - slack-monitor-windows-x86.exe
# - config.ini
```

### 2. 설정 파일 생성

바이너리와 같은 폴더에 `config.ini` 파일 생성:

```bash
# Slack 설정 (필수)
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_CHANNEL_ID=[Channel ID]
POLL_INTERVAL=10s

# ClickUp 설정 (선택)
CLICKUP_API_TOKEN=pk_your_token
CLICKUP_LIST_ID=[List ID]
HISTORY_MAX_SIZE=100

# 필터 설정 (선택)
FILTER_BOT_ONLY=true          # 봇 메시지만 처리
ALLOWED_BOT_IDS=B123,B456     # 특정 봇만 허용 (콤마 구분)
```

### 3. 실행

```bash
# macOS (Apple Silicon)
./slack-monitor-macos-apple-silicon

# macOS (Intel)
./slack-monitor-macos-intel

# Linux
./slack-monitor-linux-x86      # x86
./slack-monitor-linux-arm      # ARM

# Windows (PowerShell)
.\slack-monitor-windows-x86.exe
```

### 4. 백그라운드 실행 (macOS)

```bash
# nohup 사용
nohup ./slack-monitor-macos-apple-silicon > monitor.log 2>&1 &

# 또는 launchd 서비스 설치 (프로젝트 루트에서)
make install
```

> 💡 `config.ini`와 `history.json`은 바이너리와 **같은 폴더**에 위치해야 합니다.

---

## 📦 파일 구조

```text
SlickWebhook/
├── cmd/monitor/main.go        # 메인 엔트리포인트
├── internal/
│   ├── config/                # 설정 로더
│   ├── clickup/               # ClickUp API 클라이언트
│   ├── domain/                # 도메인 모델
│   ├── handler/               # 이벤트 핸들러
│   ├── history/               # 히스토리 저장소
│   ├── monitor/               # 모니터 서비스
│   └── slack/                 # Slack API 클라이언트
├── scripts/
│   ├── send_slack_test.sh     # Slack 테스트 메시지 전송
│   ├── install_macos.sh       # macOS 서비스 설치
│   └── com.slickwebhook.monitor.plist
├── build/                     # 빌드 결과물
├── config.env.example         # 설정 템플릿
├── Makefile                   # 빌드/테스트 명령
└── go.mod
```

---

## 🛠️ Makefile 명령어

| 명령어 | 설명 |
|--------|------|
| `make build` | 현재 플랫폼 빌드 |
| `make build-all` | 전체 플랫폼 빌드 (darwin/linux/windows) |
| `make test` | 테스트 실행 |
| `make test-cover` | 커버리지 포함 테스트 |
| `make install` | macOS 백그라운드 서비스 설치 |
| `make uninstall` | macOS 서비스 제거 |
| `make status` | 서비스 상태 확인 |
| `make clean` | 빌드 결과물 정리 |

---

## 🍎 백그라운드 실행

### 방법 1: nohup (간단)

```bash
cd build
nohup ./slack-monitor-macos-apple-silicon > monitor.log 2>&1 &

# 프로세스 확인
ps aux | grep slack-monitor

# 로그 확인
tail -f monitor.log
```

### 방법 2: macOS launchd 서비스 (권장)

```bash
# 설치 (프로젝트 루트에서)
./scripts/install_macos.sh
# 또는
make install

# 로그 확인
tail -f ~/.slickwebhook/monitor.log

# 서비스 중지
make uninstall

# 상태 확인
make status
```

> 💡 launchd 서비스는 **재부팅 후에도 자동 시작**되며, 프로세스 종료 시 **자동 재시작**됩니다.

### 방법 3: screen/tmux

```bash
screen -S slack-monitor
./slack-monitor-macos-apple-silicon
# Ctrl+A, D로 detach
```

---

## 🧪 테스트

```bash
# 전체 테스트
make test

# Slack 테스트 메시지 전송
./scripts/send_slack_test.sh 1   # Jira 이슈 스타일
./scripts/send_slack_test.sh 2   # 버그 리포트 스타일
```

---

## 📋 환경변수

| 변수명 | 필수 | 설명 |
|--------|------|------|
| `SLACK_BOT_TOKEN` | ✅ | Slack Bot 토큰 (`channels:history` 권한) |
| `SLACK_CHANNEL_ID` | ✅ | 모니터링할 채널 ID |
| `POLL_INTERVAL` | | 폴링 간격 (기본: `10s`) |
| `CLICKUP_API_TOKEN` | | ClickUp API 토큰 |
| `CLICKUP_LIST_ID` | | 태스크 생성할 리스트 ID |
| `HISTORY_MAX_SIZE` | | 히스토리 최대 개수 (기본: `100`) |

---

## 🔗 참고 문서

- [Slack API - conversations.history](https://api.slack.com/methods/conversations.history)
- [ClickUp API](https://developer.clickup.com/)
- [slack-go/slack SDK](https://github.com/slack-go/slack)
