#!/bin/bash
# macOS 백그라운드 서비스 설치 스크립트
# 사용법: ./scripts/install_macos.sh

set -e

BINARY_NAME="slack-monitor"
PLIST_NAME="com.slickwebhook.monitor.plist"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
USER_HOME="$HOME"
USERNAME=$(whoami)

echo ""
echo "============================================"
echo "  SlickWebhook macOS 서비스 설치"
echo "============================================"
echo ""

# 1. 바이너리 빌드
echo "🔨 바이너리 빌드 중..."
cd "$PROJECT_DIR"
go build -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/monitor

# 2. 바이너리 복사
echo "📦 바이너리 설치 중..."
mkdir -p ~/bin
cp "$BINARY_NAME" ~/bin/"$BINARY_NAME"
chmod +x ~/bin/"$BINARY_NAME"

# 3. 설정 디렉토리 생성
echo "📁 설정 디렉토리 생성..."
mkdir -p ~/.slickwebhook

# 4. plist 파일 복사 및 사용자명 치환
echo "⚙️ launchd 설정 중..."
sed "s|REPLACE_WITH_USERNAME|$USERNAME|g" "$SCRIPT_DIR/$PLIST_NAME" > ~/Library/LaunchAgents/"$PLIST_NAME"

# 5. .env 파일 확인
if [ ! -f ~/.slickwebhook/.env ]; then
    echo ""
    echo "⚠️  환경변수 파일이 없습니다!"
    echo "   ~/.slickwebhook/.env 파일을 생성하거나"
    echo "   ~/Library/LaunchAgents/$PLIST_NAME 파일에서"
    echo "   환경변수를 직접 수정해주세요."
    echo ""
fi

# 6. 기존 서비스 언로드 (있다면)
launchctl unload ~/Library/LaunchAgents/"$PLIST_NAME" 2>/dev/null || true

# 7. 서비스 로드
echo "🚀 서비스 시작 중..."
launchctl load ~/Library/LaunchAgents/"$PLIST_NAME"

echo ""
echo "✅ 설치 완료!"
echo ""
echo "📋 유용한 명령어:"
echo "   상태 확인: launchctl list | grep slickwebhook"
echo "   로그 확인: tail -f ~/.slickwebhook/monitor.log"
echo "   서비스 중지: launchctl unload ~/Library/LaunchAgents/$PLIST_NAME"
echo "   서비스 시작: launchctl load ~/Library/LaunchAgents/$PLIST_NAME"
echo ""
