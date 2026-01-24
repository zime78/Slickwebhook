#!/bin/bash
# AI Worker Webhook 테스트 스크립트
# 사용법: ./scripts/test_aiworker_webhook.sh [task_id]

WEBHOOK_PORT=${WEBHOOK_PORT:-8080}
TASK_ID=${1:-"test_task_123"}
LIST_ID=${AI_01_LIST_ID:-"901414115524"}

echo ""
echo "============================================"
echo "  AI Worker Webhook 테스트"
echo "============================================"
echo ""

# 서버 상태 확인
echo "📡 서버 상태 확인..."
if ! curl -s "http://localhost:$WEBHOOK_PORT/health" > /dev/null 2>&1; then
    echo "❌ AI Worker가 실행 중이 아닙니다. (포트: $WEBHOOK_PORT)"
    echo "   ./scripts/start_aiworker.sh 로 먼저 시작하세요."
    exit 1
fi
echo "   ✅ AI Worker 실행 중"
echo ""

# 1. taskCreated 이벤트 테스트
echo "📤 taskCreated 이벤트 전송..."
RESPONSE=$(curl -s -X POST "http://localhost:$WEBHOOK_PORT/webhook/clickup" \
    -H "Content-Type: application/json" \
    -d "{
        \"event\": \"taskCreated\",
        \"task_id\": \"$TASK_ID\",
        \"webhook_id\": \"test_webhook\",
        \"history_items\": [{
            \"field\": \"status\",
            \"after\": {
                \"status\": \"등록\"
            }
        }]
    }")
echo "   응답: $RESPONSE"
echo ""

# 2. taskStatusUpdated 이벤트 테스트
echo "📤 taskStatusUpdated 이벤트 전송..."
RESPONSE=$(curl -s -X POST "http://localhost:$WEBHOOK_PORT/webhook/clickup" \
    -H "Content-Type: application/json" \
    -d "{
        \"event\": \"taskStatusUpdated\",
        \"task_id\": \"$TASK_ID\",
        \"webhook_id\": \"test_webhook\",
        \"history_items\": [{
            \"field\": \"status\",
            \"after\": {
                \"status\": \"등록\"
            }
        }]
    }")
echo "   응답: $RESPONSE"
echo ""

# 3. Health check
echo "📡 Health Check..."
HEALTH=$(curl -s "http://localhost:$WEBHOOK_PORT/health")
echo "   응답: $HEALTH"
echo ""

echo "============================================"
echo "✅ 테스트 완료!"
echo "============================================"
echo ""
echo "💡 실제 ClickUp Webhook 등록:"
echo "   curl -X POST \"https://api.clickup.com/api/v2/team/{TEAM_ID}/webhook\" \\"
echo "     -H \"Authorization: {API_TOKEN}\" \\"
echo "     -H \"Content-Type: application/json\" \\"
echo "     -d '{\"endpoint\": \"https://your-ngrok-url/webhook/clickup\", \"events\": [\"taskCreated\", \"taskStatusUpdated\"]}'"
echo ""
