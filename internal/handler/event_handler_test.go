package handler

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
)

// TestLogHandler_HandleNewMessage는 새 메시지 이벤트 처리를 테스트합니다.
func TestLogHandler_HandleNewMessage(t *testing.T) {
	// Given: 로그 캡처용 버퍼와 핸들러
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	handler := NewLogHandler(logger)

	msg := &domain.Message{
		Timestamp: "1704153600.000001",
		UserID:    "U123ABC456",
		Text:      "테스트 메시지입니다",
		ChannelID: "C0A5ZTLNWA3",
		CreatedAt: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	event := domain.NewMessageEvent(msg)

	// When: 이벤트 처리
	handler.Handle(event)

	// Then: 로그에 필요한 정보가 포함됨
	output := buf.String()

	expectedParts := []string{
		"[EVENT]",
		"새 메시지 감지",
		"U123ABC456",
		"C0A5ZTLNWA3",
		"테스트 메시지입니다",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("로그에 '%s'가 포함되지 않음. 실제 출력: %s", part, output)
		}
	}
}

// TestLogHandler_HandleError는 에러 이벤트 처리를 테스트합니다.
func TestLogHandler_HandleError(t *testing.T) {
	// Given: 로그 캡처용 버퍼와 핸들러
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	handler := NewLogHandler(logger)

	testErr := errors.New("API 호출 실패")
	event := domain.NewErrorEvent(testErr)

	// When: 이벤트 처리
	handler.Handle(event)

	// Then: 로그에 에러 정보가 포함됨
	output := buf.String()

	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("로그에 '[ERROR]'가 포함되지 않음: %s", output)
	}
	if !strings.Contains(output, "API 호출 실패") {
		t.Errorf("로그에 에러 메시지가 포함되지 않음: %s", output)
	}
}

// TestTruncateText는 텍스트 자르기 함수를 테스트합니다.
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "짧은 텍스트는 그대로",
			text:   "안녕",
			maxLen: 10,
			want:   "안녕",
		},
		{
			name:   "긴 텍스트는 자름",
			text:   "안녕하세요, 긴 메시지입니다.",
			maxLen: 5,
			want:   "안녕하세요...",
		},
		{
			name:   "정확히 maxLen인 경우",
			text:   "12345",
			maxLen: 5,
			want:   "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.text, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewLogHandler는 핸들러 생성을 테스트합니다.
func TestNewLogHandler(t *testing.T) {
	logger := log.New(&bytes.Buffer{}, "", 0)
	handler := NewLogHandler(logger)

	if handler == nil {
		t.Error("핸들러가 nil입니다")
	}
	if handler.logger != logger {
		t.Error("로거가 올바르게 설정되지 않음")
	}
}
