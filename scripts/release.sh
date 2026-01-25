#!/bin/bash
# 릴리즈 생성 스크립트
# 사용법: ./scripts/release.sh v1.0.0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo ""
echo "============================================"
echo "  SlickWebhook 릴리즈 생성"
echo "============================================"
echo ""

# 버전 인자 확인
if [ -z "$1" ]; then
    CURRENT_VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")
    echo "❌ 버전을 지정해주세요."
    echo ""
    echo "사용법: ./scripts/release.sh <version>"
    echo "예시:   ./scripts/release.sh v1.0.0"
    echo ""
    echo "현재 버전: $CURRENT_VERSION"
    exit 1
fi

VERSION="$1"

# v 접두사 확인
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

echo "📋 릴리즈 버전: $VERSION"
echo ""

# 작업 디렉토리 변경사항 확인
if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  커밋되지 않은 변경사항이 있습니다:"
    git status --short
    echo ""
    read -p "계속 진행하시겠습니까? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ 취소됨"
        exit 1
    fi
fi

# 테스트 실행
echo "🧪 테스트 실행 중..."
go test ./...
echo "   ✅ 테스트 통과"
echo ""

# VERSION 파일 업데이트
VERSION_NUM="${VERSION#v}"
echo "$VERSION_NUM" > VERSION
git add VERSION

# 변경사항 커밋 (있는 경우)
if [ -n "$(git diff --cached --name-only)" ]; then
    git commit -m "chore: release $VERSION"
fi

# 태그 생성
echo "🏷️  태그 생성 중: $VERSION"
git tag -a "$VERSION" -m "Release $VERSION"

# 푸시
echo "🚀 푸시 중..."
git push origin main
git push origin "$VERSION"

echo ""
echo "✅ 릴리즈 생성 완료!"
echo ""
echo "📌 GitHub Actions가 자동으로 빌드를 시작합니다."
echo "   확인: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/actions"
echo ""
