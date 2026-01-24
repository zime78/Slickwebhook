#!/bin/bash
# AI Worker macOS 백그라운드 서비스 설치 스크립트
# 사용법: ./scripts/install_aiworker_macos.sh

set -e

BINARY_NAME="ai-worker"
PLIST_NAME="com.slickwebhook.aiworker.plist"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
USER_HOME="$HOME"
USERNAME=$(whoami)

echo ""
echo "============================================"
echo "  AI Worker macOS 서비스 설치"
echo "============================================"
echo ""

# 1. 바이너리 빌드
echo "🤖 AI Worker 빌드 중..."
cd "$PROJECT_DIR"
go build -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/ai-worker

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

# 6. Claude Code Hook 설정 확인
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
if [ ! -f "$CLAUDE_SETTINGS" ]; then
    echo ""
    echo "⚠️  Claude Code 설정 파일이 없습니다!"
    echo "   AI Worker 시작 시 자동으로 Hook 설정이 추가됩니다."
    echo ""
fi

# 7. 기존 서비스 언로드 (있다면)
launchctl unload ~/Library/LaunchAgents/"$PLIST_NAME" 2>/dev/null || true

# 8. 서비스 로드
echo "🚀 서비스 시작 중..."
launchctl load ~/Library/LaunchAgents/"$PLIST_NAME"

echo ""
echo "✅ AI Worker 설치 완료!"
echo ""
echo "📋 유용한 명령어:"
echo "   상태 확인: launchctl list | grep aiworker"
echo "   로그 확인: tail -f ~/.slickwebhook/aiworker.log"
echo "   서비스 중지: launchctl unload ~/Library/LaunchAgents/$PLIST_NAME"
echo "   서비스 시작: launchctl load ~/Library/LaunchAgents/$PLIST_NAME"
echo ""
echo "📡 서버 포트:"
echo "   Webhook 서버: http://localhost:8080"
echo "   Hook 서버: http://localhost:8081"
echo ""
echo "⚠️  중요: ngrok 등으로 Webhook 서버를 외부에 노출해야 합니다!"
echo "   ngrok http 8080"
echo ""
