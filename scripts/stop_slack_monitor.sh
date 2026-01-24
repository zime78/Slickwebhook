#!/bin/bash
# Slack Monitor 중지 스크립트
# 사용법: ./scripts/stop_slack_monitor.sh

PLIST_NAME="com.slickwebhook.monitor.plist"

echo ""
echo "============================================"
echo "  Slack Monitor 중지"
echo "============================================"
echo ""

# launchd 서비스 중지
if launchctl list | grep -q "com.slickwebhook.monitor"; then
    echo "🛑 launchd 서비스 중지 중..."
    launchctl unload ~/Library/LaunchAgents/"$PLIST_NAME" 2>/dev/null || true
    echo "   ✅ 서비스 중지됨"
else
    echo "ℹ️  launchd 서비스가 실행 중이 아닙니다."
fi

# 프로세스 직접 종료 (개발 모드로 실행 중인 경우)
MONITOR_PID=$(pgrep -f "slack-monitor" 2>/dev/null)
if [ -n "$MONITOR_PID" ]; then
    echo "🛑 Slack Monitor 프로세스 종료 중... (PID: $MONITOR_PID)"
    kill "$MONITOR_PID" 2>/dev/null || true
    sleep 1

    if pgrep -f "slack-monitor" >/dev/null 2>&1; then
        echo "   ⚠️  강제 종료 중..."
        kill -9 "$MONITOR_PID" 2>/dev/null || true
    fi
    echo "   ✅ 프로세스 종료됨"
else
    echo "ℹ️  Slack Monitor 프로세스가 실행 중이 아닙니다."
fi

echo ""
echo "✅ Slack Monitor 중지 완료!"
echo ""
