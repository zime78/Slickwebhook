#!/bin/bash
# Email Monitor macOS 백그라운드 서비스 설치 스크립트
# 사용법: ./scripts/install_email_macos.sh

set -e

BINARY_NAME="email-monitor"
PLIST_NAME="com.slickwebhook.email.plist"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
USER_HOME="$HOME"
USERNAME=$(whoami)

echo ""
echo "============================================"
echo "  Email Monitor macOS 서비스 설치"
echo "============================================"
echo ""

# 1. 바이너리 빌드
echo "📧 Email Monitor 빌드 중..."
cd "$PROJECT_DIR"
go build -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/email-monitor

# 2. 바이너리 복사
echo "📦 바이너리 설치 중..."
mkdir -p ~/bin
cp "$BINARY_NAME" ~/bin/"$BINARY_NAME"
chmod +x ~/bin/"$BINARY_NAME"

# /usr/local/bin에도 심볼릭 링크 생성 (선택적)
if [ -w /usr/local/bin ]; then
    ln -sf ~/bin/"$BINARY_NAME" /usr/local/bin/"$BINARY_NAME"
    echo "   ✅ /usr/local/bin에 링크 생성됨"
fi

# 3. 설정 디렉토리 생성
echo "📁 설정 디렉토리 생성..."
mkdir -p ~/.slickwebhook

# 4. 설정 파일 복사
if [ -f "$PROJECT_DIR/config.email.ini" ]; then
    cp "$PROJECT_DIR/config.email.ini" ~/.slickwebhook/config.email.ini
    echo "   ✅ config.email.ini 복사됨"
fi

# 5. plist 파일 복사 및 사용자명 치환
echo "⚙️ launchd 설정 중..."
sed "s|REPLACE_WITH_USERNAME|$USERNAME|g" "$SCRIPT_DIR/$PLIST_NAME" > ~/Library/LaunchAgents/"$PLIST_NAME"

# 6. 기존 서비스 언로드 (있다면)
launchctl unload ~/Library/LaunchAgents/"$PLIST_NAME" 2>/dev/null || true

# 7. 서비스 로드
echo "🚀 서비스 시작 중..."
launchctl load ~/Library/LaunchAgents/"$PLIST_NAME"

echo ""
echo "✅ Email Monitor 설치 완료!"
echo ""
echo "📋 유용한 명령어:"
echo "   상태 확인: launchctl list | grep email"
echo "   로그 확인: tail -f ~/.slickwebhook/email.log"
echo "   서비스 중지: launchctl unload ~/Library/LaunchAgents/$PLIST_NAME"
echo "   서비스 시작: launchctl load ~/Library/LaunchAgents/$PLIST_NAME"
echo ""
