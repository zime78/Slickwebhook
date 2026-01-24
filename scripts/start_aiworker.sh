#!/bin/bash
# AI Worker 시작 스크립트 (개발용)
# 사용법: ./scripts/start_aiworker.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo ""
echo "============================================"
echo "  AI Worker 시작 (개발 모드)"
echo "============================================"
echo ""

# 설정 파일 확인
CONFIG_FILE="$PROJECT_DIR/config.email.ini"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 설정 파일이 없습니다: $CONFIG_FILE"
    exit 1
fi

echo "📋 설정 파일: $CONFIG_FILE"
echo ""

# 포트 사용 확인
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  포트 8080이 이미 사용 중입니다!"
    lsof -Pi :8080 -sTCP:LISTEN
    echo ""
fi

if lsof -Pi :8081 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  포트 8081이 이미 사용 중입니다!"
    lsof -Pi :8081 -sTCP:LISTEN
    echo ""
fi

echo "🤖 AI Worker 시작..."
echo "   Webhook 서버: http://localhost:8080"
echo "   Hook 서버: http://localhost:8081"
echo ""
echo "종료하려면 Ctrl+C를 누르세요."
echo ""

cd "$PROJECT_DIR"
go run ./cmd/ai-worker
