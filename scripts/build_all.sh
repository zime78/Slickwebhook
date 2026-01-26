#!/bin/bash
# 전체 빌드 스크립트
# 사용법: ./scripts/build_all.sh [platform]
# 예: ./scripts/build_all.sh          # 현재 플랫폼
#     ./scripts/build_all.sh darwin   # macOS
#     ./scripts/build_all.sh linux    # Linux
#     ./scripts/build_all.sh all      # 모든 플랫폼

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PLATFORM=${1:-current}

echo ""
echo "============================================"
echo "  SlickWebhook 전체 빌드"
echo "============================================"
echo ""

cd "$PROJECT_DIR"

case "$PLATFORM" in
    current)
        echo "🔨 현재 플랫폼 빌드..."
        echo ""
        make build-slack && echo "   ✅ Slack Monitor"
        make build-email && echo "   ✅ Email Monitor"
        make build-ai-worker && echo "   ✅ AI Worker"
        ;;
    darwin|macos)
        echo "🍎 macOS 빌드..."
        echo ""
        make build-slack-darwin && echo "   ✅ Slack Monitor (macOS)"
        make build-email-darwin && echo "   ✅ Email Monitor (macOS)"
        make build-ai-worker-darwin && echo "   ✅ AI Worker (macOS)"
        ;;
    linux)
        echo "🐧 Linux 빌드..."
        echo ""
        make build-slack-linux && echo "   ✅ Slack Monitor (Linux)"
        make build-email-linux && echo "   ✅ Email Monitor (Linux)"
        make build-ai-worker-linux && echo "   ✅ AI Worker (Linux)"
        ;;
    windows)
        echo "🪟 Windows 빌드..."
        echo ""
        make build-slack-windows && echo "   ✅ Slack Monitor (Windows)"
        make build-email-windows && echo "   ✅ Email Monitor (Windows)"
        make build-ai-worker-windows && echo "   ✅ AI Worker (Windows)"
        ;;
    all)
        echo "🌍 모든 플랫폼 빌드..."
        echo ""
        make build-all
        ;;
    *)
        echo "사용법: $0 [platform]"
        echo ""
        echo "플랫폼:"
        echo "  current  - 현재 플랫폼 (기본값)"
        echo "  darwin   - macOS (Apple Silicon + Intel)"
        echo "  linux    - Linux (x86 + ARM)"
        echo "  windows  - Windows (x86)"
        echo "  all      - 모든 플랫폼"
        echo ""
        exit 1
        ;;
esac

echo ""
echo "============================================"
echo "✅ 빌드 완료!"
echo "============================================"
echo ""

# 빌드 결과 확인
if [ "$PLATFORM" == "all" ]; then
    echo "📦 빌드 결과:"
    ls -la "$PROJECT_DIR/build/" 2>/dev/null || echo "build 디렉토리가 없습니다."
else
    echo "📦 빌드 결과:"
    ls -la "$PROJECT_DIR"/*-monitor "$PROJECT_DIR"/ai-worker 2>/dev/null || true
fi
echo ""
