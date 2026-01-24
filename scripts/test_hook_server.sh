#!/bin/bash
# Hook Server 테스트 스크립트 (Claude Code Stop Hook 시뮬레이션)
# 사용법: ./scripts/test_hook_server.sh [work_dir]

HOOK_PORT=${HOOK_SERVER_PORT:-8081}
WORK_DIR=${1:-"/Users/zime/screen_get/SynologyDrive/screen_get_new/q_na_aos"}

echo ""
echo "============================================"
echo "  Hook Server 테스트 (Claude Code Stop)"
echo "============================================"
echo ""

# 서버 상태 확인
echo "📡 서버 상태 확인..."
if ! curl -s "http://localhost:$HOOK_PORT/health" > /dev/null 2>&1; then
    echo "❌ Hook Server가 실행 중이 아닙니다. (포트: $HOOK_PORT)"
    echo "   AI Worker를 먼저 시작하세요: ./scripts/start_aiworker.sh"
    exit 1
fi
echo "   ✅ Hook Server 실행 중"
echo ""

# Stop Hook 시뮬레이션
echo "📤 Stop Hook 이벤트 전송..."
echo "   작업 디렉토리: $WORK_DIR"
echo ""

RESPONSE=$(curl -s -X POST "http://localhost:$HOOK_PORT/hook/stop" \
    -H "Content-Type: application/json" \
    -d "{
        \"cwd\": \"$WORK_DIR\",
        \"session_id\": \"test_session_$(date +%s)\",
        \"transcript_path\": \"/tmp/test_transcript.json\",
        \"exit_code\": 0
    }")

echo "   응답: $RESPONSE"
echo ""

# Health check
echo "📡 Health Check..."
HEALTH=$(curl -s "http://localhost:$HOOK_PORT/health")
echo "   응답: $HEALTH"
echo ""

echo "============================================"
echo "✅ 테스트 완료!"
echo "============================================"
echo ""
echo "💡 이 테스트는 Claude Code가 종료될 때 자동으로 발생하는"
echo "   Stop Hook을 시뮬레이션합니다."
echo ""
