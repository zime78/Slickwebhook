# SlickWebhook Makefile
# Slack 채널 모니터링 및 Email 모니터링 서비스 빌드 및 테스트

.PHONY: all build test run clean build-all install uninstall build-slack build-email build-ai-worker

# Go 바이너리 이름
SLACK_BINARY=slack-monitor
EMAIL_BINARY=email-monitor
AI_WORKER_BINARY=ai-worker
VERSION?=1.0.0
BUILD_DIR=build

# 기본 타겟
all: test build-slack

# ============================================
# Slack Monitor 빌드
# ============================================

# Slack Monitor - 현재 플랫폼 빌드
build-slack:
	@echo "🔨 Slack Monitor 빌드 중..."
	go build -ldflags="-s -w" -o $(SLACK_BINARY) ./cmd/slack-monitor

# Slack Monitor 실행 (환경변수 필요)
run-slack:
	@echo "🚀 Slack 모니터링 서비스 실행..."
	go run ./cmd/slack-monitor

# ============================================
# Email Monitor 빌드
# ============================================

# Email Monitor - 현재 플랫폼 빌드
build-email:
	@echo "📧 Email Monitor 빌드 중..."
	go build -ldflags="-s -w" -o $(EMAIL_BINARY) ./cmd/email-monitor

# Email Monitor 실행 (환경변수 필요)
run-email:
	@echo "📧 Email 모니터링 서비스 실행..."
	go run ./cmd/email-monitor

# ============================================
# AI Worker 빌드
# ============================================

# AI Worker - 현재 플랫폼 빌드
build-ai-worker:
	@echo "🤖 AI Worker 빌드 중..."
	go build -ldflags="-s -w" -o $(AI_WORKER_BINARY) ./cmd/ai-worker

# AI Worker 실행 (환경변수 필요)
run-ai-worker:
	@echo "🤖 AI Worker 서비스 실행..."
	go run ./cmd/ai-worker

# ============================================
# 테스트
# ============================================

# 테스트 실행
test:
	@echo "🧪 테스트 실행 중..."
	go test ./... -v

# 테스트 + 커버리지
test-cover:
	@echo "🧪 테스트 + 커버리지 실행 중..."
	go test ./... -v -cover

# ============================================
# 빌드 정리 및 의존성
# ============================================

# 빌드 파일 정리
clean:
	@echo "🧹 정리 중..."
	rm -f $(SLACK_BINARY) $(EMAIL_BINARY) $(AI_WORKER_BINARY)
	rm -rf $(BUILD_DIR)
	go clean

# 의존성 정리
tidy:
	@echo "📦 의존성 정리 중..."
	go mod tidy

# ============================================
# 크로스 플랫폼 빌드 - Slack Monitor
# ============================================

# Slack Monitor 모든 플랫폼 빌드
build-slack-all: build-slack-darwin build-slack-linux build-slack-windows
	@echo "✅ Slack Monitor 모든 플랫폼 빌드 완료!"

# macOS (Apple Silicon + Intel)
build-slack-darwin:
	@echo "🍎 Slack Monitor macOS 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(SLACK_BINARY)-macos-apple-silicon ./cmd/slack-monitor
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(SLACK_BINARY)-macos-intel ./cmd/slack-monitor
	@echo "  ✅ macos-apple-silicon, macos-intel"

# Linux (x86 + ARM)
build-slack-linux:
	@echo "🐧 Slack Monitor Linux 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(SLACK_BINARY)-linux-x86 ./cmd/slack-monitor
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(SLACK_BINARY)-linux-arm ./cmd/slack-monitor
	@echo "  ✅ linux-x86, linux-arm"

# Windows (x86)
build-slack-windows:
	@echo "🪟 Slack Monitor Windows 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(SLACK_BINARY)-windows-x86.exe ./cmd/slack-monitor
	@echo "  ✅ windows-x86"

# ============================================
# 크로스 플랫폼 빌드 - Email Monitor
# ============================================

# Email Monitor 모든 플랫폼 빌드
build-email-all: build-email-darwin build-email-linux build-email-windows
	@echo "✅ Email Monitor 모든 플랫폼 빌드 완료!"

# macOS (Apple Silicon + Intel)
build-email-darwin:
	@echo "📧 Email Monitor macOS 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(EMAIL_BINARY)-macos-apple-silicon ./cmd/email-monitor
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(EMAIL_BINARY)-macos-intel ./cmd/email-monitor
	@echo "  ✅ macos-apple-silicon, macos-intel"

# Linux (x86 + ARM)
build-email-linux:
	@echo "📧 Email Monitor Linux 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(EMAIL_BINARY)-linux-x86 ./cmd/email-monitor
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(EMAIL_BINARY)-linux-arm ./cmd/email-monitor
	@echo "  ✅ linux-x86, linux-arm"

# Windows (x86)
build-email-windows:
	@echo "📧 Email Monitor Windows 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(EMAIL_BINARY)-windows-x86.exe ./cmd/email-monitor
	@echo "  ✅ windows-x86"

# ============================================
# 전체 빌드 (Slack + Email)
# ============================================

# 모든 플랫폼 빌드 (clean 후 빌드)
build-all: clean build-slack-all build-email-all
	@echo "✅ 모든 플랫폼 빌드 완료!"
	@cp config.ini $(BUILD_DIR)/config.ini 2>/dev/null || true
	@cp config.email.ini $(BUILD_DIR)/config.email.ini 2>/dev/null || true
	@echo "📄 설정 파일 복사됨"
	@ls -la $(BUILD_DIR)/

# ============================================
# macOS 백그라운드 서비스 (launchd) - Slack Monitor
# ============================================

# macOS 서비스 설치
install:
	@echo "📦 macOS 서비스 설치 중..."
	@mkdir -p ~/.slickwebhook
	@cp $(SLACK_BINARY) /usr/local/bin/$(SLACK_BINARY) 2>/dev/null || cp $(SLACK_BINARY) ~/bin/$(SLACK_BINARY)
	@cp scripts/com.slickwebhook.monitor.plist ~/Library/LaunchAgents/
	@launchctl load ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@echo "✅ 설치 완료! 서비스가 백그라운드에서 실행됩니다."
	@echo "   로그: ~/.slickwebhook/monitor.log"

# macOS 서비스 제거
uninstall:
	@echo "🗑️ macOS 서비스 제거 중..."
	@launchctl unload ~/Library/LaunchAgents/com.slickwebhook.monitor.plist 2>/dev/null || true
	@rm -f ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@rm -f /usr/local/bin/$(SLACK_BINARY) ~/bin/$(SLACK_BINARY)
	@echo "✅ 제거 완료!"

# 서비스 상태 확인
status:
	@export LANG=ko_KR.UTF-8 && launchctl list | grep slickwebhook || echo "서비스가 실행 중이 아닙니다."

# 서비스 재시작
restart:
	@launchctl unload ~/Library/LaunchAgents/com.slickwebhook.monitor.plist 2>/dev/null || true
	@launchctl load ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@echo "✅ 서비스 재시작 완료!"
