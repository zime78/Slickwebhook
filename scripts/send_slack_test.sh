#!/bin/bash
# Slack 테스트 메시지 전송 스크립트 (ClickUp Agent 연동용)
# 사용법: 
#   ./scripts/send_slack_test.sh          # 기본 메시지
#   ./scripts/send_slack_test.sh 1        # Jira 이슈 알림
#   ./scripts/send_slack_test.sh 2        # 버그 리포트
#   ./scripts/send_slack_test.sh 3        # 상태 업데이트
#   ./scripts/send_slack_test.sh "메시지" # 커스텀 메시지

# 스크립트 디렉토리 기준으로 .env 로드
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# 환경변수 확인
if [ -z "$SLACK_BOT_TOKEN" ]; then
    echo "❌ 에러: SLACK_BOT_TOKEN 환경변수가 설정되지 않았습니다"
    exit 1
fi

if [ -z "$SLACK_CHANNEL_ID" ]; then
    echo "❌ 에러: SLACK_CHANNEL_ID 환경변수가 설정되지 않았습니다"
    exit 1
fi

# 타임스탬프
TIMESTAMP=$(date "+%Y-%m-%d %H:%M")

# 샘플 메시지 선택
case "$1" in
    1)
        # Jira ITSM 이슈 알림
        MESSAGE="🎫 *[ITSM-2950] FA asia 앱 언어 변경 이슈*

> FA asia 앱 사용시 중국으로 선택후 일본어 표시되는 현상 발생됨
> 지역: 대만
> 매장명: FRIENDS SCREEN 台灣旗艦店

📎 Jira: https://kakaovx.atlassian.net/browse/ITSM-2950
👤 담당자: @이준석zime
⚠️ 우선순위: 보통

확인 부탁드립니다."
        ;;
    2)
        # 버그 리포트
        MESSAGE="🐛 *[버그] Q-글로벌 예약시스템 오류 발생*

• 현상: 예약 완료 후 확인 화면에서 에러 발생
• 환경: Android 14, 앱 버전 2.5.1
• 재현 빈도: 간헐적 (약 30%)

📎 관련 Jira: https://kakaovx.atlassian.net/browse/ITSM-574
📸 스크린샷 첨부됨

@이준석zime 확인 부탁드립니다!"
        ;;
    3)
        # 상태 업데이트
        MESSAGE="✅ *[완료] ITSM-577 기능 테스트 완료*

• 예약 시스템 기능 테스트 완료
• 문제점 수정 확인됨
• QA 검증 통과

📎 https://kakaovx.atlassian.net/browse/ITSM-577
다음 단계: 스테이징 배포 예정"
        ;;
    4)
        # 요약 요청
        MESSAGE="@Slack/Jira Issue Flow Assistant 현재 오픈된 이슈 요약해줘"
        ;;
    5)
        # 간단한 이슈 제보
        MESSAGE="🚨 *[긴급] 로그인 실패 다수 발생*

고객센터에서 로그인 실패 문의가 급증하고 있습니다.
- 발생 시간: $TIMESTAMP 부터
- 영향 범위: 전체 사용자
- 에러 메시지: \"인증 서버 응답 없음\"

@이준석zime 긴급 확인 요청드립니다!"
        ;;
    "")
        # 기본 테스트 메시지
        MESSAGE="🧪 *모니터링 테스트* - $TIMESTAMP

• 채널: C07AFHKESVC
• 발신: SlickWebhook 테스트
• 상태: 정상 동작 확인 중"
        ;;
    *)
        # 커스텀 메시지
        MESSAGE="$1"
        ;;
esac

echo "📤 Slack 메시지 전송 중..."
echo "   채널: $SLACK_CHANNEL_ID"
echo "----------------------------------------"
echo "$MESSAGE"
echo "----------------------------------------"
echo ""

# Slack API 호출
RESPONSE=$(curl -s -X POST "https://slack.com/api/chat.postMessage" \
    -H "Authorization: Bearer $SLACK_BOT_TOKEN" \
    -H "Content-Type: application/json; charset=utf-8" \
    -d "{
        \"channel\": \"$SLACK_CHANNEL_ID\",
        \"text\": $(echo "$MESSAGE" | jq -Rs .)
    }")

# 결과 확인
OK=$(echo "$RESPONSE" | grep -o '"ok":true')
if [ -n "$OK" ]; then
    echo "✅ 전송 성공!"
    TS=$(echo "$RESPONSE" | grep -o '"ts":"[^"]*"' | cut -d'"' -f4)
    echo "   타임스탬프: $TS"
else
    echo "❌ 전송 실패!"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
fi
