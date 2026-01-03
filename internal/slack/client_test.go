package slack

import (
	"testing"
	"time"
)

// TestParseSlackTimestamp는 Slack 타임스탬프 파싱을 테스트합니다.
func TestParseSlackTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "일반적인 타임스탬프",
			timestamp: "1704153600.123456",
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   2,
		},
		{
			name:      "소수점 없는 타임스탬프",
			timestamp: "1704153600",
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSlackTimestamp(tt.timestamp)
			if got.Year() != tt.wantYear {
				t.Errorf("Year = %d, want %d", got.Year(), tt.wantYear)
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("Month = %v, want %v", got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("Day = %d, want %d", got.Day(), tt.wantDay)
			}
		})
	}
}

// TestNewSlackClient는 클라이언트 생성을 테스트합니다.
func TestNewSlackClient(t *testing.T) {
	// Given: 테스트용 토큰
	token := "xoxb-test-token"

	// When: 클라이언트 생성
	client := NewSlackClient(token)

	// Then: 클라이언트가 nil이 아님
	if client == nil {
		t.Error("클라이언트가 nil입니다")
	}
	if client.api == nil {
		t.Error("내부 API 클라이언트가 nil입니다")
	}
}
