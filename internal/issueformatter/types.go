package issueformatter

import (
	"context"

	"github.com/zime/slickwebhook/internal/clickup"
)

// Config는 포맷터 설정입니다.
type Config struct {
	OutputDir     string // 미디어 저장 디렉토리
	MaxFrames     int    // 동영상 최대 프레임 수
	FrameInterval int    // 프레임 추출 간격 (초)
}

// DefaultConfig는 기본 설정을 반환합니다.
func DefaultConfig() Config {
	return Config{
		OutputDir:     "/tmp/issueformatter",
		MaxFrames:     5,
		FrameInterval: 5,
	}
}

// AIPrompt는 AI에게 전달할 결과물입니다.
type AIPrompt struct {
	Text       string   // 마크다운 텍스트
	ImagePaths []string // 로컬 이미지 경로
}

// Formatter는 ClickUp 이슈를 AI 전달용 텍스트로 변환하는 인터페이스입니다.
type Formatter interface {
	Format(ctx context.Context, task *clickup.Task) (*AIPrompt, error)
}

// MediaProcessor는 미디어 파일을 처리하는 인터페이스입니다.
type MediaProcessor interface {
	DownloadImage(ctx context.Context, url, outputPath string) error
	ExtractVideoFrames(ctx context.Context, url, outputDir string, interval int) ([]string, error)
}
