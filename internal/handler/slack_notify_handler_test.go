package handler

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/zime/slickwebhook/internal/domain"
)

// MockSlackNotifier는 테스트용 Slack 클라이언트입니다.
type MockSlackNotifier struct {
	shouldFail    bool
	callCount     int
	lastChannelID string
	lastBlocks    []slack.Block
	lastText      string
}

func (m *MockSlackNotifier) PostMessage(ctx context.Context, channelID string, blocks []slack.Block, text string) error {
	m.callCount++
	m.lastChannelID = channelID
	m.lastBlocks = blocks
	m.lastText = text
	if m.shouldFail {
		return errors.New("mock slack error")
	}
	return nil
}

// TestSlackNotifyHandler_Handle는 정상 전송을 테스트합니다.
func TestSlackNotifyHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	mockClient := &MockSlackNotifier{}

	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Client:    mockClient,
		ChannelID: "C123456",
		Logger:    logger,
		Enabled:   true,
	})

	msg := &domain.Message{
		Source:    "email",
		Subject:   "테스트 제목",
		From:      "sender@example.com",
		Text:      "테스트 본문입니다.",
		CreatedAt: time.Now(),
	}
	event := domain.NewMessageEvent(msg)

	handler.Handle(event)

	// Slack 클라이언트가 호출되었는지 확인
	if mockClient.callCount != 1 {
		t.Errorf("Slack 클라이언트가 호출되지 않음: %d", mockClient.callCount)
	}

	// 채널 ID 확인
	if mockClient.lastChannelID != "C123456" {
		t.Errorf("채널 ID가 올바르지 않음: %s", mockClient.lastChannelID)
	}

	// Block이 생성되었는지 확인
	if len(mockClient.lastBlocks) == 0 {
		t.Error("Block이 생성되지 않음")
	}

	// 로그 확인
	output := buf.String()
	if !strings.Contains(output, "전송 성공") {
		t.Errorf("성공 로그가 없음: %s", output)
	}
}

// TestSlackNotifyHandler_Disabled는 비활성화 시 처리를 테스트합니다.
func TestSlackNotifyHandler_Disabled(t *testing.T) {
	mockClient := &MockSlackNotifier{}

	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Client:    mockClient,
		ChannelID: "C123456",
		Logger:    log.New(&bytes.Buffer{}, "", 0),
		Enabled:   false, // 비활성화
	})

	msg := &domain.Message{
		Source: "email",
		Text:   "test",
	}
	event := domain.NewMessageEvent(msg)
	handler.Handle(event)

	// 비활성화 시 클라이언트가 호출되지 않아야 함
	if mockClient.callCount != 0 {
		t.Error("비활성화 시 클라이언트가 호출됨")
	}
}

// TestSlackNotifyHandler_NonEmailSource는 이메일 외 소스 스킵을 테스트합니다.
func TestSlackNotifyHandler_NonEmailSource(t *testing.T) {
	mockClient := &MockSlackNotifier{}

	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Client:    mockClient,
		ChannelID: "C123456",
		Logger:    log.New(&bytes.Buffer{}, "", 0),
		Enabled:   true,
	})

	// slack 소스 메시지 (이메일이 아님)
	msg := &domain.Message{
		Source: "slack",
		Text:   "test",
	}
	event := domain.NewMessageEvent(msg)
	handler.Handle(event)

	// 이메일이 아닌 소스는 처리하지 않아야 함
	if mockClient.callCount != 0 {
		t.Error("이메일 외 소스도 처리됨")
	}
}

// TestSlackNotifyHandler_Error는 에러 처리를 테스트합니다.
func TestSlackNotifyHandler_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	mockClient := &MockSlackNotifier{shouldFail: true}

	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Client:    mockClient,
		ChannelID: "C123456",
		Logger:    logger,
		Enabled:   true,
	})

	msg := &domain.Message{
		Source:    "email",
		Subject:   "테스트",
		CreatedAt: time.Now(),
	}
	event := domain.NewMessageEvent(msg)
	handler.Handle(event)

	// 에러 로그 확인
	output := buf.String()
	if !strings.Contains(output, "전송 실패") {
		t.Errorf("실패 로그가 없음: %s", output)
	}
}

// TestBuildEmailBlocks는 Block 생성을 테스트합니다.
func TestBuildEmailBlocks(t *testing.T) {
	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Logger:  log.New(&bytes.Buffer{}, "", 0),
		Enabled: true,
	})

	msg := &domain.Message{
		Source:    "email",
		Subject:   "테스트 제목",
		From:      "sender@example.com",
		Text:      "테스트 본문입니다.",
		CreatedAt: time.Date(2025, 1, 7, 14, 30, 0, 0, time.UTC),
	}

	blocks := handler.buildEmailBlocks(msg)

	// Block 수 확인 (Header + Meta + Body + Divider + Context = 5)
	if len(blocks) != 5 {
		t.Errorf("Block 수가 올바르지 않음: %d", len(blocks))
	}
}

// TestEscapeSlackText는 특수문자 이스케이프를 테스트합니다.
func TestEscapeSlackText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{"<test>&value</test>", "&lt;test&gt;&amp;value&lt;/test&gt;"},
	}

	for _, tt := range tests {
		result := escapeSlackText(tt.input)
		if result != tt.expected {
			t.Errorf("escapeSlackText(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestTruncateTextForSlack는 텍스트 자르기를 테스트합니다.
func TestTruncateTextForSlack(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"  trimmed  ", 20, "trimmed"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateTextForSlack(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateTextForSlack(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// TestExtractJiraLinks는 Jira 링크 추출을 테스트합니다.
func TestExtractJiraLinks(t *testing.T) {
	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Logger:      log.New(&bytes.Buffer{}, "", 0),
		Enabled:     true,
		JiraBaseURL: "https://example.atlassian.net",
	})

	tests := []struct {
		name     string
		subject  string
		body     string
		expected string
	}{
		{
			name:     "제목에서 단일 이슈 추출",
			subject:  "[Jira] (ITSM-4417) Golf VX App 수정요청",
			body:     "",
			expected: "<https://example.atlassian.net/browse/ITSM-4417|ITSM-4417>",
		},
		{
			name:     "본문에서 이슈 추출",
			subject:  "테스트 이메일",
			body:     "PROJ-123 이슈를 확인해주세요",
			expected: "<https://example.atlassian.net/browse/PROJ-123|PROJ-123>",
		},
		{
			name:     "중복 이슈 제거",
			subject:  "ITSM-100 관련",
			body:     "ITSM-100 이슈와 ITSM-100 확인",
			expected: "<https://example.atlassian.net/browse/ITSM-100|ITSM-100>",
		},
		{
			name:     "여러 이슈 추출",
			subject:  "ITSM-100, PROJ-200",
			body:     "",
			expected: "<https://example.atlassian.net/browse/ITSM-100|ITSM-100>, <https://example.atlassian.net/browse/PROJ-200|PROJ-200>",
		},
		{
			name:     "이슈 없음",
			subject:  "일반 이메일 제목",
			body:     "본문 내용입니다",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractJiraLinks(tt.subject, tt.body)
			if result != tt.expected {
				t.Errorf("extractJiraLinks() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExtractJiraLinks_NoBaseURL는 JiraBaseURL이 없을 때를 테스트합니다.
func TestExtractJiraLinks_NoBaseURL(t *testing.T) {
	handler := NewSlackNotifyHandler(SlackNotifyHandlerConfig{
		Logger:      log.New(&bytes.Buffer{}, "", 0),
		Enabled:     true,
		JiraBaseURL: "", // 빈 URL
	})

	result := handler.extractJiraLinks("[Jira] ITSM-100", "본문")
	if result != "" {
		t.Errorf("JiraBaseURL이 없을 때는 빈 문자열 반환해야 함, got %q", result)
	}
}
