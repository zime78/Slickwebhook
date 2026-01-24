#!/bin/bash
# 로그 확인 스크립트
# 사용법: ./scripts/logs.sh [service]
# 예: ./scripts/logs.sh slack
#     ./scripts/logs.sh email
#     ./scripts/logs.sh aiworker
#     ./scripts/logs.sh all

SERVICE=${1:-all}
LOG_DIR="$HOME/.slickwebhook"

echo ""
echo "============================================"
echo "  SlickWebhook 로그 뷰어"
echo "============================================"
echo ""

case "$SERVICE" in
    slack)
        echo "📋 Slack Monitor 로그 (Ctrl+C로 종료)"
        echo ""
        tail -f "$LOG_DIR/monitor.log" "$LOG_DIR/monitor.error.log" 2>/dev/null || echo "로그 파일이 없습니다."
        ;;
    email)
        echo "📋 Email Monitor 로그 (Ctrl+C로 종료)"
        echo ""
        tail -f "$LOG_DIR/email.log" "$LOG_DIR/email.error.log" 2>/dev/null || echo "로그 파일이 없습니다."
        ;;
    aiworker|ai)
        echo "📋 AI Worker 로그 (Ctrl+C로 종료)"
        echo ""
        tail -f "$LOG_DIR/aiworker.log" "$LOG_DIR/aiworker.error.log" 2>/dev/null || echo "로그 파일이 없습니다."
        ;;
    all)
        echo "📋 전체 로그 (Ctrl+C로 종료)"
        echo ""
        tail -f "$LOG_DIR"/*.log 2>/dev/null || echo "로그 파일이 없습니다."
        ;;
    *)
        echo "사용법: $0 [service]"
        echo ""
        echo "서비스:"
        echo "  slack     - Slack Monitor 로그"
        echo "  email     - Email Monitor 로그"
        echo "  aiworker  - AI Worker 로그"
        echo "  all       - 전체 로그 (기본값)"
        echo ""
        exit 1
        ;;
esac
