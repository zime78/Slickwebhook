#!/bin/bash

# ============================
# ClickUp Agent 자동화 테스트 스크립트
# ============================

# 👉 환경 변수 설정
CLICKUP_API_TOKEN="${CLICKUP_API_TOKEN:-}"   # ClickUp API 토큰 (환경변수 필수)
LIST_ID="901413896178"                          # Location Overview 리스트 ID
AGENT_NAME="Slack/Jira Issue Flow Assistant"    # 현재 설정된 Agent 이름

# 👉 테스트용 타임스탬프 생성 (중복 태스크 방지)
NOW=$(date '+%Y-%m-%d %H:%M:%S')

echo ""
echo "=============================================="
echo "  ClickUp Agent 자동화 테스트"
echo "=============================================="
echo ""
echo "📋 설정 정보:"
echo "   - List ID: $LIST_ID"
echo "   - Agent: $AGENT_NAME"
echo "   - 시간: $NOW"
echo ""

# 👉 JSON Payload 정의
read -r -d '' PAYLOAD <<-EOM
{
  "name": "[Shell Test Trigger] Slack/Jira Agent Automation - $NOW",
  "description": "이 태스크는 Shell 스크립트 테스트용 자동화 트리거입니다.\\n\\nSlack 링크: https://slack.com/app_redirect?channel=D07BRDPJCGH\\nJira 링크: https://kakaovx.atlassian.net/browse/ITSM-9999\\n\\n이 태스크가 생성되면 [$AGENT_NAME] 가 자동으로 실행되어 Jira/Slack 감지를 수행하고 [@Zime](#288777246)을 멘션한 코멘트를 추가해야 합니다.",
  "assignees": [288777246],
  "priority": 3,
  "tags": ["test", "agent-trigger"]
}
EOM

echo "📡 태스크 생성 요청 전송 중..."
echo ""

# 👉 API 호출
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "https://api.clickup.com/api/v2/list/$LIST_ID/task" \
  -H "Authorization: $CLICKUP_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

# 마지막 줄에서 HTTP 상태코드 추출
HTTP_STATUS=$(echo "$RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$RESPONSE" | sed '$d')

echo "📊 결과:"
echo "----------------------------------------------"
echo "HTTP 상태코드: $HTTP_STATUS"
echo ""

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✅ 태스크 생성 성공!"
    echo ""
    
    # jq가 있으면 JSON 포맷팅
    if command -v jq &> /dev/null; then
        TASK_ID=$(echo "$RESPONSE_BODY" | jq -r '.id')
        TASK_NAME=$(echo "$RESPONSE_BODY" | jq -r '.name')
        TASK_URL=$(echo "$RESPONSE_BODY" | jq -r '.url')
        
        echo "   Task ID: $TASK_ID"
        echo "   Task 이름: $TASK_NAME"
        echo "   Task URL: $TASK_URL"
    else
        echo "$RESPONSE_BODY"
    fi
else
    echo "❌ 태스크 생성 실패"
    echo "$RESPONSE_BODY"
fi

echo "----------------------------------------------"
echo ""

# 👉 요청 이후 결과 안내 메시지 출력
echo "➡️ Location Overview: https://app.clickup.com/9014928476/v/li/901413896178"
echo ""
echo "💡 잠시 후 (약 10~30초), 해당 태스크의 Activity 로그에서"
echo "   Agent 자동 실행 또는 코멘트 추가를 확인할 수 있습니다."
echo ""
