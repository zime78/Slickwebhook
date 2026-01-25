#!/bin/bash
# AI Worker 업데이트 스크립트
# 사용법: 
#   ./scripts/update_aiworker.sh           # Git pull 방식 (개발 환경)
#   ./scripts/update_aiworker.sh --release  # GitHub Releases에서 다운로드

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PLIST_NAME="com.slickwebhook.aiworker.plist"
PLIST_PATH=~/Library/LaunchAgents/"$PLIST_NAME"

# GitHub 저장소 정보
GITHUB_REPO="zime78/SlickWebhook"
BINARY_NAME="ai-worker"

echo ""
echo "============================================"
echo "  AI Worker 업데이트"
echo "============================================"
echo ""

cd "$PROJECT_DIR"

# 현재 실행 상태 확인
IS_LAUNCHD=false
IS_BACKGROUND=false

if launchctl list 2>/dev/null | grep -q "com.slickwebhook.aiworker"; then
    IS_LAUNCHD=true
    echo "📌 현재 상태: LaunchAgent로 실행 중"
elif [ -f "$PROJECT_DIR/logs/aiworker.pid" ]; then
    PID=$(cat "$PROJECT_DIR/logs/aiworker.pid")
    if ps -p "$PID" > /dev/null 2>&1; then
        IS_BACKGROUND=true
        echo "📌 현재 상태: 백그라운드로 실행 중 (PID: $PID)"
    fi
else
    echo "📌 현재 상태: 실행 중이 아님"
fi

# 1) 서비스 중지
echo ""
echo "🛑 서비스 중지 중..."
if [ "$IS_LAUNCHD" = true ]; then
    launchctl unload "$PLIST_PATH" 2>/dev/null
    echo "   ✅ LaunchAgent 중지됨"
elif [ "$IS_BACKGROUND" = true ]; then
    "$SCRIPT_DIR/stop_aiworker.sh" > /dev/null 2>&1
    echo "   ✅ 백그라운드 프로세스 중지됨"
fi

# 릴리즈 모드 확인
if [ "$1" = "--release" ] || [ "$1" = "-r" ]; then
    # GitHub Releases에서 다운로드
    echo ""
    echo "📥 GitHub Releases에서 최신 버전 다운로드 중..."
    
    # 아키텍처 감지
    ARCH=$(uname -m)
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    if [ "$ARCH" = "arm64" ]; then
        ARCH_NAME="arm64"
    else
        ARCH_NAME="amd64"
    fi
    
    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH_NAME}"
    
    # 최신 릴리즈 정보 가져오기
    LATEST_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    RELEASE_INFO=$(curl -s "$LATEST_URL")
    
    if echo "$RELEASE_INFO" | grep -q "Not Found"; then
        echo "❌ 릴리즈를 찾을 수 없습니다."
        echo "   저장소: https://github.com/${GITHUB_REPO}/releases"
        exit 1
    fi
    
    DOWNLOAD_URL=$(echo "$RELEASE_INFO" | grep -o "\"browser_download_url\": \"[^\"]*${ASSET_NAME}\"" | cut -d'"' -f4)
    VERSION=$(echo "$RELEASE_INFO" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$DOWNLOAD_URL" ]; then
        echo "❌ 해당 플랫폼용 바이너리를 찾을 수 없습니다: $ASSET_NAME"
        echo "   사용 가능한 에셋:"
        echo "$RELEASE_INFO" | grep -o '"name": "[^"]*"' | head -10
        exit 1
    fi
    
    echo "   버전: $VERSION"
    echo "   파일: $ASSET_NAME"
    
    # 다운로드
    mkdir -p "$PROJECT_DIR/bin"
    curl -L -o "$PROJECT_DIR/bin/ai-worker" "$DOWNLOAD_URL"
    chmod +x "$PROJECT_DIR/bin/ai-worker"
    
    echo "   ✅ 다운로드 완료"
    
else
    # Git pull 방식 (기존)
    echo ""
    echo "📥 코드 업데이트 중..."
    git pull
    if [ $? -ne 0 ]; then
        echo "   ⚠️  git pull 실패 (계속 진행)"
    fi
    
    echo ""
    echo "🔨 빌드 중..."
    go build -o ./bin/ai-worker ./cmd/ai-worker
    if [ $? -ne 0 ]; then
        echo "❌ 빌드 실패"
        exit 1
    fi
    echo "   ✅ 빌드 완료"
fi

# 서비스 재시작
echo ""
echo "🚀 서비스 재시작 중..."
if [ "$IS_LAUNCHD" = true ]; then
    launchctl load "$PLIST_PATH"
    echo "   ✅ LaunchAgent 시작됨"
elif [ "$IS_BACKGROUND" = true ]; then
    "$SCRIPT_DIR/start_aiworker.sh" --bg > /dev/null 2>&1
    echo "   ✅ 백그라운드로 시작됨"
else
    echo "   ℹ️  이전 실행 상태 없음 - 수동으로 시작하세요"
    echo "      ./scripts/start_aiworker.sh --bg"
fi

echo ""
echo "✅ 업데이트 완료!"
echo ""
