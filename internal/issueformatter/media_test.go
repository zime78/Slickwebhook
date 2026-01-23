package issueformatter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestIsImageFile은 이미지 파일 확인을 테스트합니다.
func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"test.png", true},
		{"test.jpg", true},
		{"test.jpeg", true},
		{"test.gif", true},
		{"test.webp", true},
		{"test.mp4", false},
		{"test.txt", false},
		{"test.PNG", true}, // 대소문자 무시
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsImageFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// TestIsVideoFile은 동영상 파일 확인을 테스트합니다.
func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"test.mp4", true},
		{"test.mov", true},
		{"test.avi", true},
		{"test.webm", true},
		{"test.mkv", true},
		{"test.png", false},
		{"test.txt", false},
		{"test.MP4", true}, // 대소문자 무시
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsVideoFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsVideoFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// TestDownloadImage는 이미지 다운로드를 테스트합니다.
func TestDownloadImage(t *testing.T) {
	// Mock 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		// 간단한 PNG 헤더 (1x1 픽셀)
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	}))
	defer server.Close()

	// 임시 디렉토리
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_image.png")

	// 테스트 실행
	processor := NewMediaProcessor()
	err := processor.DownloadImage(context.Background(), server.URL, outputPath)

	if err != nil {
		t.Fatalf("DownloadImage 실패: %v", err)
	}

	// 파일 존재 확인
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("다운로드된 파일이 존재하지 않음")
	}
}

// TestDownloadImage_HTTPError는 HTTP 에러 처리를 테스트합니다.
func TestDownloadImage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_image.png")

	processor := NewMediaProcessor()
	err := processor.DownloadImage(context.Background(), server.URL, outputPath)

	if err == nil {
		t.Error("HTTP 에러가 발생해야 합니다")
	}
}

// TestNewMediaProcessor는 MediaProcessor 생성을 테스트합니다.
func TestNewMediaProcessor(t *testing.T) {
	processor := NewMediaProcessor()
	if processor == nil {
		t.Error("MediaProcessor가 nil입니다")
	}
	if processor.httpClient == nil {
		t.Error("httpClient가 nil입니다")
	}
}
