#!/bin/bash
#
# Slack → ClickUp 자동 진단 스크립트
# 터미널에서 직접 실행하여 진단 결과를 확인할 수 있습니다.
#

# ===== 설정 =====
# API 토큰을 여기에 직접 입력하거나, 환경변수로 설정하세요
CLICKUP_API_TOKEN="${CLICKUP_API_TOKEN:-}"
ENDPOINT="https://api.clickup.com/api/v2/agent/diagnostic"

# 스크립트 디렉토리 기준 상대 경로
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/../diagnostic_config.json"

# ===== 함수 =====
print_header() {
    echo ""
    echo "=============================================="
    echo "  Slack → ClickUp 자동 진단"
    echo "=============================================="
    echo ""
}

check_prerequisites() {
    if [ -z "$CLICKUP_API_TOKEN" ]; then
        echo "❌ 오류: CLICKUP_API_TOKEN 환경변수를 설정해주세요."
        echo ""
        echo "   사용법:"
        echo "   export CLICKUP_API_TOKEN='your_token_here'"
        echo "   ./run_diagnostic.sh"
        echo ""
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        echo "❌ 오류: curl이 설치되어 있지 않습니다."
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        echo "⚠️  경고: jq가 설치되어 있지 않아 JSON 포맷팅이 제한됩니다."
        echo "   설치: brew install jq"
        echo ""
    fi

    if [ ! -f "$CONFIG_FILE" ]; then
        echo "❌ 오류: 설정 파일을 찾을 수 없습니다: $CONFIG_FILE"
        exit 1
    fi
}

run_diagnostic() {
    echo "📡 진단 요청 전송 중..."
    echo "   엔드포인트: $ENDPOINT"
    echo ""

    # HTTP 상태코드와 응답 본문을 함께 가져옴
    HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $CLICKUP_API_TOKEN" \
        -d @"$CONFIG_FILE")

    # 마지막 줄에서 HTTP 상태코드 추출
    HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n 1)
    RESPONSE=$(echo "$HTTP_RESPONSE" | sed '$d')

    echo "📊 진단 결과:"
    echo "----------------------------------------------"
    echo "HTTP 상태코드: $HTTP_STATUS"
    echo ""

    # 응답이 비어있는지 확인
    if [ -z "$RESPONSE" ]; then
        echo "⚠️  응답이 비어있습니다."
    elif command -v jq &> /dev/null && echo "$RESPONSE" | jq . &> /dev/null; then
        # 유효한 JSON인 경우 포맷팅
        echo "$RESPONSE" | jq .
    else
        # JSON이 아닌 경우 원본 출력
        echo "원본 응답:"
        echo "$RESPONSE"
    fi

    echo "----------------------------------------------"
    echo ""
}

# ===== 메인 =====
print_header
check_prerequisites
run_diagnostic

echo "✅ 진단 완료"
