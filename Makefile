# SlickWebhook Makefile
# Slack 채널 모니터링 서비스 빌드 및 테스트

.PHONY: all build test run clean build-all install uninstall

# Go 바이너리 이름
BINARY_NAME=slack-monitor
VERSION?=1.0.0
BUILD_DIR=build

# 기본 타겟
all: test build

# 현재 플랫폼 빌드
build:
	@echo "🔨 빌드 중..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/monitor

# 테스트 실행
test:
	@echo "🧪 테스트 실행 중..."
	go test ./... -v

# 테스트 + 커버리지
test-cover:
	@echo "🧪 테스트 + 커버리지 실행 중..."
	go test ./... -v -cover

# 실행 (환경변수 필요)
run:
	@echo "🚀 모니터링 서비스 실행..."
	go run ./cmd/monitor

# 빌드 파일 정리
clean:
	@echo "🧹 정리 중..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	go clean

# 의존성 정리
tidy:
	@echo "📦 의존성 정리 중..."
	go mod tidy

# ============================================
# 크로스 플랫폼 빌드
# ============================================

# 모든 플랫폼 빌드 (clean 후 빌드)
build-all: clean build-darwin build-linux build-windows
	@echo "✅ 모든 플랫폼 빌드 완료!"
	@cp config.ini $(BUILD_DIR)/config.ini
	@echo "📄 config.ini 복사됨"
	@ls -la $(BUILD_DIR)/

# macOS (Apple Silicon + Intel)
build-darwin:
	@echo "🍎 macOS 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-apple-silicon ./cmd/monitor
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-intel ./cmd/monitor
	@echo "  ✅ macos-apple-silicon, macos-intel"

# Linux (x86 + ARM)
build-linux:
	@echo "🐧 Linux 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-x86 ./cmd/monitor
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./cmd/monitor
	@echo "  ✅ linux-x86, linux-arm"

# Windows (x86)
build-windows:
	@echo "🪟 Windows 빌드 중..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-x86.exe ./cmd/monitor
	@echo "  ✅ windows-x86"

# ============================================
# macOS 백그라운드 서비스 (launchd)
# ============================================

# macOS 서비스 설치
install:
	@echo "📦 macOS 서비스 설치 중..."
	@mkdir -p ~/.slickwebhook
	@cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) 2>/dev/null || cp $(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@cp scripts/com.slickwebhook.monitor.plist ~/Library/LaunchAgents/
	@launchctl load ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@echo "✅ 설치 완료! 서비스가 백그라운드에서 실행됩니다."
	@echo "   로그: ~/.slickwebhook/monitor.log"

# macOS 서비스 제거
uninstall:
	@echo "🗑️ macOS 서비스 제거 중..."
	@launchctl unload ~/Library/LaunchAgents/com.slickwebhook.monitor.plist 2>/dev/null || true
	@rm -f ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@rm -f /usr/local/bin/$(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@echo "✅ 제거 완료!"

# 서비스 상태 확인
status:
	@export LANG=ko_KR.UTF-8 && launchctl list | grep slickwebhook || echo "서비스가 실행 중이 아닙니다."

# 서비스 재시작
restart:
	@launchctl unload ~/Library/LaunchAgents/com.slickwebhook.monitor.plist 2>/dev/null || true
	@launchctl load ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
	@echo "✅ 서비스 재시작 완료!"
