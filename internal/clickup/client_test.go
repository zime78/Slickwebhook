package clickup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
)

// TestNewClickUpClient는 클라이언트 생성을 테스트합니다.
func TestNewClickUpClient(t *testing.T) {
	config := Config{
		APIToken: "test-token",
		ListID:   "123456",
	}

	client := NewClickUpClient(config)

	if client == nil {
		t.Fatal("클라이언트가 nil입니다")
	}
	if client.config.AssigneeID != 288777246 {
		t.Errorf("기본 AssigneeID가 설정되지 않음: %d", client.config.AssigneeID)
	}
}

// TestClickUpClient_CreateTask는 태스크 생성을 테스트합니다.
func TestClickUpClient_CreateTask(t *testing.T) {
	// Mock 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 검증
		if r.Method != "POST" {
			t.Errorf("잘못된 메서드: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "test-token" {
			t.Error("Authorization 헤더가 없음")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type 헤더가 올바르지 않음")
		}

		// 응답 반환
		resp := TaskResponse{
			ID:   "task123",
			Name: "[Slack 이벤트] 테스트 메시지",
			URL:  "https://app.clickup.com/t/task123",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 클라이언트 생성 (테스트 서버 URL 사용)
	config := Config{
		APIToken: "test-token",
		ListID:   "123456",
	}
	client := NewClickUpClient(config)
	client.baseURL = server.URL

	// 테스트 메시지
	msg := &domain.Message{
		Timestamp: "1704153600.000001",
		UserID:    "U123ABC",
		Text:      "테스트 메시지입니다",
		ChannelID: "C0A5ZTLNWA3",
		CreatedAt: time.Now(),
	}

	// 태스크 생성
	resp, err := client.CreateTask(context.Background(), msg)

	if err != nil {
		t.Fatalf("태스크 생성 실패: %v", err)
	}
	if resp.ID != "task123" {
		t.Errorf("잘못된 태스크 ID: %s", resp.ID)
	}
}

// TestClickUpClient_CreateTask_Error는 API 에러 처리를 테스트합니다.
func TestClickUpClient_CreateTask_Error(t *testing.T) {
	// Mock 서버 - 에러 응답
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"err": "Invalid token"}`))
	}))
	defer server.Close()

	config := Config{
		APIToken: "invalid-token",
		ListID:   "123456",
	}
	client := NewClickUpClient(config)
	client.baseURL = server.URL

	msg := &domain.Message{
		Timestamp: "1704153600.000001",
		UserID:    "U123ABC",
		Text:      "테스트",
		ChannelID: "C123",
		CreatedAt: time.Now(),
	}

	_, err := client.CreateTask(context.Background(), msg)

	if err == nil {
		t.Error("에러가 발생해야 합니다")
	}
}

// TestTruncateText는 텍스트 자르기를 테스트합니다.
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{"짧은 텍스트", "안녕", 10, "안녕"},
		{"긴 텍스트", "안녕하세요, 반갑습니다", 5, "안녕하세요..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.text, tt.maxLen)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
