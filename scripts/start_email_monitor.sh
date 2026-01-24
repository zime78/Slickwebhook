#!/bin/bash
# Email Monitor 시작 스크립트 (개발용)
# 사용법: ./scripts/start_email_monitor.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo ""
echo "============================================"
echo "  Email Monitor 시작 (개발 모드)"
echo "============================================"
echo ""

# 설정 파일 확인
CONFIG_FILE="$PROJECT_DIR/config.email.ini"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 설정 파일이 없습니다: $CONFIG_FILE"
    echo "   config.email.ini.example을 복사하여 설정하세요."
    exit 1
fi

echo "📋 설정 파일: $CONFIG_FILE"
echo ""

echo "📧 Email Monitor 시작..."
echo "종료하려면 Ctrl+C를 누르세요."
echo ""

cd "$PROJECT_DIR"
go run ./cmd/email-monitor
