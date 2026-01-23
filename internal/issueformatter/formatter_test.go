package issueformatter

import (
	"context"
	"strings"
	"testing"

	"github.com/zime/slickwebhook/internal/clickup"
)

// MockMediaProcessor는 테스트용 미디어 프로세서입니다.
type MockMediaProcessor struct {
	downloadCalled bool
	extractCalled  bool
}

func (m *MockMediaProcessor) DownloadImage(ctx context.Context, url, outputPath string) error {
	m.downloadCalled = true
	return nil
}

func (m *MockMediaProcessor) ExtractVideoFrames(ctx context.Context, url, outputDir string, interval int) ([]string, error) {
	m.extractCalled = true
	return []string{"/tmp/frame_001.jpg", "/tmp/frame_002.jpg"}, nil
}

// TestExtractTitle은 제목 추출을 테스트합니다.
func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "ITSM-5168 [Q-글로벌][AOS] 매장 찾기 > 검색 결과가 없는 경우 토스트 팝업 미노출",
			want: "매장 찾기 > 검색 결과가 없는 경우 토스트 팝업 미노출",
		},
		{
			name: "[Q-글로벌][AOS] 매장 찾기 버그",
			want: "매장 찾기 버그",
		},
		{
			name: "단순한 버그 제목",
			want: "단순한 버그 제목",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.name)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseDescription은 설명 파싱을 테스트합니다.
func TestParseDescription(t *testing.T) {
	description := `[재현 스텝]
1. Q app 실행 > 로그인
2. 메뉴바 > [매장찾기] 선택

[오류내용]
- 지도형에서 토스트 미노출

[수정요청]
1. 토스트 팝업 노출하도록 수정`

	sections := parseDescription(description)

	if _, ok := sections["재현 스텝"]; !ok {
		t.Error("재현 스텝 섹션이 없습니다")
	}

	if _, ok := sections["오류내용"]; !ok {
		t.Error("오류내용 섹션이 없습니다")
	}

	if _, ok := sections["수정요청"]; !ok {
		t.Error("수정요청 섹션이 없습니다")
	}
}

// TestNewIssueFormatter는 포맷터 생성을 테스트합니다.
func TestNewIssueFormatter(t *testing.T) {
	config := DefaultConfig()
	formatter := NewIssueFormatter(config)

	if formatter == nil {
		t.Error("Formatter가 nil입니다")
	}
	if formatter.processor == nil {
		t.Error("Processor가 nil입니다")
	}
}

// TestFormat은 전체 포맷팅을 테스트합니다.
func TestFormat(t *testing.T) {
	config := Config{
		OutputDir:     t.TempDir(),
		MaxFrames:     3,
		FrameInterval: 5,
	}

	mockProcessor := &MockMediaProcessor{}
	formatter := NewIssueFormatterWithProcessor(config, mockProcessor)

	task := &clickup.Task{
		ID:   "test123",
		Name: "ITSM-5168 [Q-글로벌][AOS] 매장 찾기 버그",
		Description: `[재현 스텝]
1. 앱 실행

[오류내용]
지도형에서 토스트 미노출

[수정요청]
1. 토스트 팝업 노출`,
		Attachments: []clickup.Attachment{
			{Title: "screenshot.png", URL: "https://example.com/image.png"},
		},
	}

	prompt, err := formatter.Format(context.Background(), task)

	if err != nil {
		t.Fatalf("Format 실패: %v", err)
	}

	if prompt == nil {
		t.Fatal("prompt가 nil입니다")
	}

	// 마크다운 텍스트 검증
	if !strings.Contains(prompt.Text, "# 버그:") {
		t.Error("제목이 포함되어 있지 않습니다")
	}

	if !strings.Contains(prompt.Text, "## 현재 동작") {
		t.Error("현재 동작 섹션이 없습니다")
	}

	if !strings.Contains(prompt.Text, "## 기대 동작") {
		t.Error("기대 동작 섹션이 없습니다")
	}

	// 미디어 처리 확인
	if !mockProcessor.downloadCalled {
		t.Error("이미지 다운로드가 호출되지 않았습니다")
	}
}

// TestFormat_NilTask는 nil 태스크 처리를 테스트합니다.
func TestFormat_NilTask(t *testing.T) {
	config := DefaultConfig()
	formatter := NewIssueFormatter(config)

	_, err := formatter.Format(context.Background(), nil)

	if err == nil {
		t.Error("nil 태스크에 대해 에러가 발생해야 합니다")
	}
}

// TestExtractFixPoint는 수정 포인트 추출을 테스트합니다.
func TestExtractFixPoint(t *testing.T) {
	description := `[수정요청]
1. 토스트 팝업 노출하도록 수정
2. 추가 작업`

	fixPoint := extractFixPoint(description)

	if !strings.Contains(fixPoint, "토스트 팝업") {
		t.Errorf("수정 포인트가 올바르지 않습니다: %s", fixPoint)
	}
}

// TestDefaultConfig는 기본 설정을 테스트합니다.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.OutputDir == "" {
		t.Error("OutputDir가 비어있습니다")
	}
	if config.MaxFrames <= 0 {
		t.Error("MaxFrames가 0 이하입니다")
	}
	if config.FrameInterval <= 0 {
		t.Error("FrameInterval가 0 이하입니다")
	}
}
