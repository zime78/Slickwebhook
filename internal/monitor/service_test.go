package monitor

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

// MockSlackClient는 테스트용 Slack 클라이언트입니다.
type MockSlackClient struct {
	messages   []*domain.Message
	shouldFail bool
	callCount  int
}

func (m *MockSlackClient) GetChannelHistory(ctx context.Context, channelID string, oldest string) ([]*domain.Message, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("API 호출 실패")
	}
	return m.messages, nil
}

func (m *MockSlackClient) PostMessage(ctx context.Context, channelID string, blocks []slack.Block, text string) error {
	return nil
}

// MockEventHandler는 테스트용 이벤트 핸들러입니다.
type MockEventHandler struct {
	events []*domain.Event
}

func (m *MockEventHandler) Handle(event *domain.Event) {
	m.events = append(m.events, event)
}

// TestService_NewService는 서비스 생성을 테스트합니다.
func TestService_NewService(t *testing.T) {
	config := Config{
		ChannelID:    "C123456",
		PollInterval: 5 * time.Second,
	}
	client := &MockSlackClient{}
	handler := &MockEventHandler{}
	logger := log.New(&bytes.Buffer{}, "", 0)

	service := NewService(config, client, handler, logger)

	if service == nil {
		t.Fatal("서비스가 nil입니다")
	}
	if service.config.ChannelID != "C123456" {
		t.Errorf("ChannelID가 올바르지 않음: %s", service.config.ChannelID)
	}
	if service.config.PollInterval != 5*time.Second {
		t.Errorf("PollInterval이 올바르지 않음: %v", service.config.PollInterval)
	}
}

// TestService_DefaultPollInterval은 기본 폴링 간격을 테스트합니다.
func TestService_DefaultPollInterval(t *testing.T) {
	config := Config{
		ChannelID: "C123456",
		// PollInterval을 설정하지 않음
	}
	client := &MockSlackClient{}
	handler := &MockEventHandler{}
	logger := log.New(&bytes.Buffer{}, "", 0)

	service := NewService(config, client, handler, logger)

	if service.config.PollInterval != DefaultPollInterval {
		t.Errorf("기본 PollInterval이 적용되지 않음: got %v, want %v",
			service.config.PollInterval, DefaultPollInterval)
	}
}

// TestService_StartStop은 서비스 시작/중지를 테스트합니다.
func TestService_StartStop(t *testing.T) {
	config := Config{
		ChannelID:    "C123456",
		PollInterval: 50 * time.Millisecond, // 빠른 테스트를 위해 짧게 설정
	}
	client := &MockSlackClient{}
	handler := &MockEventHandler{}
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	service := NewService(config, client, handler, logger)

	// Start를 고루틴으로 실행
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = service.Start(ctx)
	}()

	// 잠시 대기 후 Running 상태 확인
	time.Sleep(100 * time.Millisecond)

	if !service.IsRunning() {
		t.Error("서비스가 실행 중이어야 합니다")
	}

	// Stop 호출
	service.Stop()

	// 중지 확인을 위해 잠시 대기
	time.Sleep(100 * time.Millisecond)

	if service.IsRunning() {
		t.Error("서비스가 중지되어야 합니다")
	}

	// 로그에 시작/종료 메시지 확인
	output := buf.String()
	if !strings.Contains(output, "시작") {
		t.Error("시작 로그가 없습니다")
	}
}

// TestService_CheckNewMessages는 새 메시지 감지를 테스트합니다.
func TestService_CheckNewMessages(t *testing.T) {
	messages := []*domain.Message{
		{
			Timestamp: "1704153600.000001",
			UserID:    "U123",
			Text:      "테스트 메시지",
			ChannelID: "C123456",
		},
	}

	config := Config{
		ChannelID:    "C123456",
		PollInterval: 50 * time.Millisecond,
	}
	client := &MockSlackClient{messages: messages}
	handler := &MockEventHandler{}
	logger := log.New(&bytes.Buffer{}, "", 0)

	service := NewService(config, client, handler, logger)

	// checkForNewMessages 직접 호출
	ctx := context.Background()
	service.checkForNewMessages(ctx)

	// 핸들러에 이벤트가 전달되었는지 확인
	if len(handler.events) != 1 {
		t.Errorf("이벤트 수가 올바르지 않음: got %d, want 1", len(handler.events))
	}

	if handler.events[0].Type != domain.EventTypeNewMessage {
		t.Error("이벤트 타입이 올바르지 않음")
	}
}

// TestService_HandleError는 에러 처리를 테스트합니다.
func TestService_HandleError(t *testing.T) {
	config := Config{
		ChannelID:    "C123456",
		PollInterval: 50 * time.Millisecond,
	}
	client := &MockSlackClient{shouldFail: true}
	handler := &MockEventHandler{}
	logger := log.New(&bytes.Buffer{}, "", 0)

	service := NewService(config, client, handler, logger)

	// checkForNewMessages 호출 (에러 발생해야 함)
	ctx := context.Background()
	service.checkForNewMessages(ctx)

	// 에러 이벤트가 전달되었는지 확인
	if len(handler.events) != 1 {
		t.Errorf("이벤트 수가 올바르지 않음: got %d, want 1", len(handler.events))
	}

	if handler.events[0].Type != domain.EventTypeError {
		t.Error("에러 이벤트가 발생해야 합니다")
	}
}
