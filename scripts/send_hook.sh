#!/bin/bash
#
# Slack → ClickUp Agent Hook 전송 스크립트
# ClickUp AI Agent에 Slack 스타일 메시지를 전송합니다.
#

# ===== 설정 =====
# API 토큰을 여기에 직접 입력하거나, 환경변수로 설정하세요
CLICKUP_API_TOKEN="${CLICKUP_API_TOKEN:-}"

# Agent Hook 엔드포인트
ENDPOINT="https://api.clickup.com/api/v2/agent/hook"

# 스크립트 디렉토리 기준 상대 경로
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PAYLOAD_FILE="${SCRIPT_DIR}/../payload.json"

# ===== 함수 =====
print_header() {
    echo ""
    echo "=============================================="
    echo "  Slack → ClickUp Agent Hook 전송"
    echo "=============================================="
    echo ""
}

check_prerequisites() {
    # API 토큰 확인
    if [ -z "$CLICKUP_API_TOKEN" ]; then
        echo "❌ 오류: CLICKUP_API_TOKEN 환경변수를 설정해주세요."
        echo ""
        echo "   사용법:"
        echo "   export CLICKUP_API_TOKEN='your_token_here'"
        echo "   ./send_hook.sh"
        echo ""
        exit 1
    fi

    # curl 설치 확인
    if ! command -v curl &> /dev/null; then
        echo "❌ 오류: curl이 설치되어 있지 않습니다."
        exit 1
    fi

    # jq 설치 확인 (선택)
    if ! command -v jq &> /dev/null; then
        echo "⚠️  경고: jq가 설치되어 있지 않아 JSON 포맷팅이 제한됩니다."
        echo "   설치: brew install jq"
        echo ""
    fi

    # 페이로드 파일 확인
    if [ ! -f "$PAYLOAD_FILE" ]; then
        echo "⚠️  경고: 페이로드 파일을 찾을 수 없습니다: $PAYLOAD_FILE"
        echo "   기본 페이로드를 사용합니다."
        echo ""
    fi
}

send_hook() {
    echo "📡 Agent Hook 전송 중..."
    echo "   엔드포인트: $ENDPOINT"
    echo ""

    # 페이로드 파일이 있으면 사용, 없으면 기본 데이터 사용
    if [ -f "$PAYLOAD_FILE" ]; then
        PAYLOAD_DATA=$(cat "$PAYLOAD_FILE")
        echo "📋 페이로드 파일 사용: $PAYLOAD_FILE"
    else
        # 기본 페이로드
        PAYLOAD_DATA='{
            "channel": "C0A5ZTLNWA3",
            "username": "Slack/Jira Bot",
            "text": "[테스트] Slack Hook 테스트 메시지",
            "attachments": [],
            "metadata": {
                "agent_target": "Slack/Jira Issue Flow Assistant",
                "clickup_folder": "https://app.clickup.com/9014928476/v/o/f/90147454316"
            }
        }'
        echo "📋 기본 페이로드 사용"
    fi
    echo ""

    # HTTP 상태코드와 응답 본문을 함께 가져옴
    HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Authorization: $CLICKUP_API_TOKEN" \
        -d "$PAYLOAD_DATA")

    # 마지막 줄에서 HTTP 상태코드 추출
    HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n 1)
    RESPONSE=$(echo "$HTTP_RESPONSE" | sed '$d')

    echo "📊 전송 결과:"
    echo "----------------------------------------------"
    echo "HTTP 상태코드: $HTTP_STATUS"
    echo ""

    # 응답 처리
    if [ "$HTTP_STATUS" = "200" ]; then
        echo "✅ Agent Hook 전송 성공!"
        echo ""
        if command -v jq &> /dev/null && echo "$RESPONSE" | jq . &> /dev/null; then
            echo "$RESPONSE" | jq .
        else
            echo "$RESPONSE"
        fi
    elif [ "$HTTP_STATUS" = "401" ]; then
        echo "❌ 인증 실패: API 토큰을 확인해주세요."
        echo "$RESPONSE"
    elif [ "$HTTP_STATUS" = "400" ]; then
        echo "❌ 잘못된 요청: 페이로드를 확인해주세요."
        echo "$RESPONSE"
    elif [ "$HTTP_STATUS" = "404" ]; then
        echo "❌ 찾을 수 없음: Agent 또는 엔드포인트를 확인해주세요."
        echo "$RESPONSE"
    else
        echo "⚠️  응답 수신"
        if command -v jq &> /dev/null && echo "$RESPONSE" | jq . &> /dev/null; then
            echo "$RESPONSE" | jq .
        else
            echo "원본 응답:"
            echo "$RESPONSE"
        fi
    fi

    echo "----------------------------------------------"
    echo ""
}

# ===== 메인 =====
print_header
check_prerequisites
send_hook

echo "✅ 스크립트 완료"
