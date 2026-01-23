package issueformatter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultMediaProcessor는 기본 미디어 처리기입니다.
type DefaultMediaProcessor struct {
	httpClient *http.Client
}

// NewMediaProcessor는 새 미디어 처리기를 생성합니다.
func NewMediaProcessor() *DefaultMediaProcessor {
	return &DefaultMediaProcessor{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// DownloadImage는 URL에서 이미지를 다운로드합니다.
func (p *DefaultMediaProcessor) DownloadImage(ctx context.Context, url, outputPath string) error {
	// 디렉토리 생성
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %w", err)
	}

	// 요청 실행
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("다운로드 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 에러: %d", resp.StatusCode)
	}

	// 파일 생성
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %w", err)
	}
	defer file.Close()

	// 데이터 복사
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}

// ExtractVideoFrames는 동영상에서 프레임을 추출합니다.
// ffmpeg가 설치되어 있어야 합니다.
func (p *DefaultMediaProcessor) ExtractVideoFrames(ctx context.Context, url, outputDir string, interval int) ([]string, error) {
	// 디렉토리 생성
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// ffmpeg 실행 가능 여부 확인
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg가 설치되어 있지 않습니다: %w", err)
	}

	// 출력 패턴
	outputPattern := filepath.Join(outputDir, "frame_%03d.jpg")

	// ffmpeg 명령 실행
	// fps=1/interval: interval초마다 1프레임 추출
	fpsFilter := fmt.Sprintf("fps=1/%d", interval)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",           // 덮어쓰기
		"-i", url,      // 입력 URL
		"-vf", fpsFilter, // 프레임 추출 간격
		"-q:v", "2",    // 품질 (낮을수록 좋음)
		outputPattern,  // 출력 패턴
	)

	// 에러 출력 캡처
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg 실행 실패: %w, output: %s", err, string(output))
	}

	// 생성된 프레임 파일 목록 조회
	frames, err := filepath.Glob(filepath.Join(outputDir, "frame_*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("프레임 파일 조회 실패: %w", err)
	}

	return frames, nil
}

// IsImageFile은 파일이 이미지인지 확인합니다.
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp"
}

// IsVideoFile은 파일이 동영상인지 확인합니다.
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".webm" || ext == ".mkv"
}
