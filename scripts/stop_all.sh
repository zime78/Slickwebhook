#!/bin/bash
# 전체 서비스 중지 스크립트
# 사용법: ./scripts/stop_all.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ""
echo "============================================"
echo "  SlickWebhook 전체 서비스 중지"
echo "============================================"
echo ""

# Slack Monitor 중지
echo "🛑 Slack Monitor 중지..."
"$SCRIPT_DIR/stop_slack_monitor.sh" 2>/dev/null || true

# Email Monitor 중지
echo "🛑 Email Monitor 중지..."
"$SCRIPT_DIR/stop_email_monitor.sh" 2>/dev/null || true

# AI Worker 중지
echo "🛑 AI Worker 중지..."
"$SCRIPT_DIR/stop_aiworker.sh" 2>/dev/null || true

echo ""
echo "============================================"
echo "✅ 전체 서비스 중지 완료!"
echo "============================================"
echo ""

# 최종 상태 확인
"$SCRIPT_DIR/status_all.sh"
