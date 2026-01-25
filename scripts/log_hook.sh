#!/bin/bash
# Claude Code Hook 로깅 스크립트
# 모든 Hook 페이로드를 파일로 기록

LOG_FILE="${HOME}/.claude/hook_debug.log"
HOOK_NAME="${1:-unknown}"

# stdin에서 JSON 읽기
PAYLOAD=$(cat)

# 타임스탬프와 함께 로그 기록
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$HOOK_NAME] $PAYLOAD" >> "$LOG_FILE"

# 원래 Hook 처리 (Stop, SessionEnd)
if [[ "$HOOK_NAME" == "Stop" ]]; then
    echo "$PAYLOAD" | curl -s -X POST http://localhost:8081/hook/stop -H 'Content-Type: application/json' -d @-
elif [[ "$HOOK_NAME" == "SessionEnd" ]]; then
    echo "$PAYLOAD" | curl -s -X POST http://localhost:8081/hook/session-end -H 'Content-Type: application/json' -d @-
fi
