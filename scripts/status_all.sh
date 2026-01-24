#!/bin/bash
# 전체 서비스 상태 확인 스크립트
# 사용법: ./scripts/status_all.sh

echo ""
echo "============================================"
echo "  SlickWebhook 서비스 상태"
echo "============================================"
echo ""

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_service() {
    local name=$1
    local process=$2
    local plist=$3

    echo "📌 $name:"

    # launchd 서비스 확인
    if launchctl list 2>/dev/null | grep -q "$plist"; then
        echo -e "   launchd: ${GREEN}실행 중${NC}"
    else
        echo -e "   launchd: ${YELLOW}미등록${NC}"
    fi

    # 프로세스 확인
    local pid=$(pgrep -f "$process" 2>/dev/null)
    if [ -n "$pid" ]; then
        echo -e "   프로세스: ${GREEN}실행 중${NC} (PID: $pid)"
    else
        echo -e "   프로세스: ${RED}중지됨${NC}"
    fi
    echo ""
}

# Slack Monitor
check_service "Slack Monitor" "slack-monitor" "com.slickwebhook.monitor"

# Email Monitor
check_service "Email Monitor" "email-monitor" "com.slickwebhook.email"

# AI Worker
check_service "AI Worker" "ai-worker" "com.slickwebhook.aiworker"

# 포트 상태 확인
echo "📡 포트 상태:"
for port in 8080 8081; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        local process=$(lsof -Pi :$port -sTCP:LISTEN | tail -1 | awk '{print $1}')
        echo -e "   포트 $port: ${GREEN}사용 중${NC} ($process)"
    else
        echo -e "   포트 $port: ${YELLOW}사용 가능${NC}"
    fi
done

echo ""
echo "📋 로그 파일 위치:"
echo "   ~/.slickwebhook/monitor.log      (Slack Monitor)"
echo "   ~/.slickwebhook/email.log        (Email Monitor)"
echo "   ~/.slickwebhook/aiworker.log     (AI Worker)"
echo ""
