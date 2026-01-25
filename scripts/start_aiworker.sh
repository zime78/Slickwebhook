#!/bin/bash
# AI Worker 시작 스크립트
# 사용법: 
#   ./start_aiworker.sh       # 포그라운드 실행 (개발용)
#   ./start_aiworker.sh --bg  # 백그라운드 실행 (로그 파일 저장)

# 스크립트가 있는 디렉토리 기준으로 프로젝트 루트 찾기
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 설정 파일 확인
CONFIG_FILE="$PROJECT_DIR/config.email.ini"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 설정 파일이 없습니다: $CONFIG_FILE"
    exit 1
fi

# 프로젝트 루트로 이동
cd "$PROJECT_DIR"

# 설정 파일에서 환경변수 로드
set -a
source "$CONFIG_FILE"
set +a

# 빌드
echo "🔨 빌드 중..."
go build -o ./bin/ai-worker ./cmd/ai-worker
if [ $? -ne 0 ]; then
    echo "❌ 빌드 실패"
    exit 1
fi
echo "✅ 빌드 완료"

# 백그라운드 실행 모드 확인
if [ "$1" = "--bg" ] || [ "$1" = "-b" ] || [ "$1" = "--background" ]; then
    echo ""
    echo "============================================"
    echo "  AI Worker 시작 (백그라운드 모드)"
    echo "============================================"
    echo ""
    
    # logs 디렉토리 생성
    mkdir -p "$PROJECT_DIR/logs"
    
    # 로그 파일 경로
    LOG_FILE="$PROJECT_DIR/logs/aiworker.log"
    PID_FILE="$PROJECT_DIR/logs/aiworker.pid"
    
    # 이미 실행 중인지 확인
    if [ -f "$PID_FILE" ]; then
        OLD_PID=$(cat "$PID_FILE")
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            echo "⚠️  이미 실행 중입니다 (PID: $OLD_PID)"
            echo "   종료하려면: ./scripts/stop_aiworker.sh"
            exit 1
        fi
    fi
    
    # 백그라운드로 실행 (LOG_TO_FILE 환경변수 설정)
    LOG_TO_FILE=1 nohup ./bin/ai-worker > /dev/null 2>&1 &
    NEW_PID=$!
    echo $NEW_PID > "$PID_FILE"
    
    echo "📋 설정 파일: $CONFIG_FILE"
    echo "📝 로그 파일: $LOG_FILE"
    echo "🆔 PID: $NEW_PID (저장: $PID_FILE)"
    echo ""
    echo "🤖 AI Worker가 백그라운드에서 시작되었습니다."
    echo ""
    echo "📌 명령어:"
    echo "   로그 확인: tail -f $LOG_FILE"
    echo "   종료: ./scripts/stop_aiworker.sh"
    echo "   상태: ps -p $NEW_PID"
    
else
    # 포그라운드 실행 (기존 방식)
    echo ""
    echo "============================================"
    echo "  AI Worker 시작 (개발 모드)"
    echo "============================================"
    echo ""
    
    echo "📋 설정 파일: $CONFIG_FILE"
    echo ""
    
    # 포트 사용 확인
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  포트 8080이 이미 사용 중입니다!"
        lsof -Pi :8080 -sTCP:LISTEN
        echo ""
    fi

    if lsof -Pi :8081 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  포트 8081이 이미 사용 중입니다!"
        lsof -Pi :8081 -sTCP:LISTEN
        echo ""
    fi

    echo "🤖 AI Worker 시작..."
    echo "   Webhook 서버: http://localhost:8080"
    echo "   Hook 서버: http://localhost:8081"
    echo ""
    echo "종료하려면 Ctrl+C를 누르세요."
    echo ""
    
    ./bin/ai-worker
fi
