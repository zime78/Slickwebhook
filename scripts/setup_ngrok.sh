#!/bin/bash
# ngrok 설정 및 실행 스크립트 (AI Worker Webhook용)
# 사용법: ./scripts/setup_ngrok.sh

echo ""
echo "============================================"
echo "  ngrok 설정 (AI Worker Webhook)"
echo "============================================"
echo ""

# ngrok 설치 확인
if ! command -v ngrok &> /dev/null; then
    echo "❌ ngrok이 설치되어 있지 않습니다."
    echo ""
    echo "설치 방법:"
    echo "   brew install ngrok/ngrok/ngrok"
    echo "   또는 https://ngrok.com/download"
    echo ""
    exit 1
fi

# ngrok 인증 확인
if ! ngrok config check &> /dev/null; then
    echo "⚠️  ngrok 인증이 필요합니다."
    echo ""
    echo "1. https://dashboard.ngrok.com/get-started/your-authtoken 에서 토큰 확인"
    echo "2. ngrok config add-authtoken YOUR_TOKEN"
    echo ""
    exit 1
fi

echo "🌐 ngrok 터널 시작 (포트 8080)..."
echo ""
echo "터널이 시작되면 Forwarding URL을 ClickUp Webhook에 등록하세요."
echo "예: https://xxxx-xxx-xxx-xxx-xxx.ngrok-free.app/webhook/clickup"
echo ""
echo "종료하려면 Ctrl+C를 누르세요."
echo ""

ngrok http 8080
