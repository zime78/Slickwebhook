#!/bin/bash
# 전체 서비스 제거 스크립트
# 사용법: ./scripts/uninstall_all.sh

echo ""
echo "============================================"
echo "  SlickWebhook 전체 서비스 제거"
echo "============================================"
echo ""

read -p "⚠️  모든 서비스와 설정을 제거하시겠습니까? (y/N) " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "취소되었습니다."
    exit 0
fi

echo ""

# launchd 서비스 언로드
echo "🛑 launchd 서비스 중지..."
launchctl unload ~/Library/LaunchAgents/com.slickwebhook.monitor.plist 2>/dev/null || true
launchctl unload ~/Library/LaunchAgents/com.slickwebhook.email.plist 2>/dev/null || true
launchctl unload ~/Library/LaunchAgents/com.slickwebhook.aiworker.plist 2>/dev/null || true

# plist 파일 제거
echo "🗑️ launchd 설정 제거..."
rm -f ~/Library/LaunchAgents/com.slickwebhook.monitor.plist
rm -f ~/Library/LaunchAgents/com.slickwebhook.email.plist
rm -f ~/Library/LaunchAgents/com.slickwebhook.aiworker.plist

# 바이너리 제거
echo "🗑️ 바이너리 제거..."
rm -f ~/bin/slack-monitor
rm -f ~/bin/email-monitor
rm -f ~/bin/ai-worker
rm -f /usr/local/bin/slack-monitor 2>/dev/null || true
rm -f /usr/local/bin/email-monitor 2>/dev/null || true
rm -f /usr/local/bin/ai-worker 2>/dev/null || true

# 프로세스 종료
echo "🛑 실행 중인 프로세스 종료..."
pkill -f "slack-monitor" 2>/dev/null || true
pkill -f "email-monitor" 2>/dev/null || true
pkill -f "ai-worker" 2>/dev/null || true

# 설정 디렉토리 제거 여부 확인
echo ""
read -p "📁 설정 및 로그 디렉토리(~/.slickwebhook)도 제거하시겠습니까? (y/N) " remove_config
if [[ "$remove_config" == "y" || "$remove_config" == "Y" ]]; then
    rm -rf ~/.slickwebhook
    echo "   ✅ ~/.slickwebhook 제거됨"
else
    echo "   ℹ️  ~/.slickwebhook 유지됨"
fi

echo ""
echo "============================================"
echo "✅ 제거 완료!"
echo "============================================"
echo ""
