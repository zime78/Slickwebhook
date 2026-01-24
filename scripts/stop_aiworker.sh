#!/bin/bash
# AI Worker 중지 스크립트
# 사용법: ./scripts/stop_aiworker.sh

PLIST_NAME="com.slickwebhook.aiworker.plist"

echo ""
echo "============================================"
echo "  AI Worker 중지"
echo "============================================"
echo ""

# launchd 서비스 중지
if launchctl list | grep -q "com.slickwebhook.aiworker"; then
    echo "🛑 launchd 서비스 중지 중..."
    launchctl unload ~/Library/LaunchAgents/"$PLIST_NAME" 2>/dev/null || true
    echo "   ✅ 서비스 중지됨"
else
    echo "ℹ️  launchd 서비스가 실행 중이 아닙니다."
fi

# 프로세스 직접 종료 (개발 모드로 실행 중인 경우)
AI_WORKER_PID=$(pgrep -f "ai-worker" 2>/dev/null)
if [ -n "$AI_WORKER_PID" ]; then
    echo "🛑 AI Worker 프로세스 종료 중... (PID: $AI_WORKER_PID)"
    kill "$AI_WORKER_PID" 2>/dev/null || true
    sleep 1

    # 강제 종료 확인
    if pgrep -f "ai-worker" >/dev/null 2>&1; then
        echo "   ⚠️  강제 종료 중..."
        kill -9 "$AI_WORKER_PID" 2>/dev/null || true
    fi
    echo "   ✅ 프로세스 종료됨"
else
    echo "ℹ️  AI Worker 프로세스가 실행 중이 아닙니다."
fi

# 포트 확인
echo ""
echo "📡 포트 상태 확인:"
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "   ⚠️  포트 8080이 여전히 사용 중입니다!"
    lsof -Pi :8080 -sTCP:LISTEN
else
    echo "   ✅ 포트 8080 사용 가능"
fi

if lsof -Pi :8081 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "   ⚠️  포트 8081이 여전히 사용 중입니다!"
    lsof -Pi :8081 -sTCP:LISTEN
else
    echo "   ✅ 포트 8081 사용 가능"
fi

echo ""
echo "✅ AI Worker 중지 완료!"
echo ""
