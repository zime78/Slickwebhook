#!/bin/bash
# AI Worker 시작 스크립트 (개발용)
# 사용법: ./start_aiworker.sh (scripts 폴더 내에서 실행 가능)

# 스크립트가 있는 디렉토리 기준으로 프로젝트 루트 찾기
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo ""
echo "============================================"
echo "  AI Worker 시작 (개발 모드)"
echo "============================================"
echo ""

# 설정 파일 확인
CONFIG_FILE="$PROJECT_DIR/config.email.ini"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 설정 파일이 없습니다: $CONFIG_FILE"
    exit 1
fi

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

# 설정 파일에서 환경변수 로드 (go run 시 임시 디렉토리 문제 해결)
set -a
source "$CONFIG_FILE"
set +a

# 프로젝트 루트로 이동
cd "$PROJECT_DIR"

# 빌드 후 실행 (코드 변경 즉시 반영)
echo "🔨 빌드 중..."
go build -o ./bin/ai-worker ./cmd/ai-worker
if [ $? -ne 0 ]; then
    echo "❌ 빌드 실패"
    exit 1
fi

echo "✅ 빌드 완료"
./bin/ai-worker
