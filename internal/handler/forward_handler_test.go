package handler

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/history"
)

// MockClickUpClient는 테스트용 ClickUp 클라이언트입니다.
type MockClickUpClient struct {
	shouldFail bool
	callCount  int
	lastMsg    *domain.Message
}

func (m *MockClickUpClient) CreateTask(ctx context.Context, msg *domain.Message) (*clickup.TaskResponse, error) {
	m.callCount++
	m.lastMsg = msg
	if m.shouldFail {
		return nil, context.DeadlineExceeded
	}
	return &clickup.TaskResponse{
		ID:   "mock-task-123",
		Name: "[Slack 이벤트] " + msg.Text,
		URL:  "https://app.clickup.com/t/mock-task-123",
	}, nil
}

func (m *MockClickUpClient) UploadAttachment(ctx context.Context, taskID string, filename string, data []byte) error {
	return nil // Mock: 항상 성공
}

func (m *MockClickUpClient) GetTask(ctx context.Context, taskID string) (*clickup.Task, error) {
	return nil, nil // Mock: 항상 nil 반환
}

func (m *MockClickUpClient) GetTasks(ctx context.Context, listID string, opts *clickup.GetTasksOptions) ([]*clickup.Task, error) {
	return nil, nil // Mock: 항상 nil 반환
}

func (m *MockClickUpClient) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	return nil // Mock: 항상 성공
}

func (m *MockClickUpClient) MoveTaskToList(ctx context.Context, taskID string, listID string) error {
	return nil // Mock: 항상 성공
}

// TestForwardHandler_Handle는 ClickUp 전송을 테스트합니다.
func TestForwardHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	mockClient := &MockClickUpClient{}
	store := history.NewMemoryStore(100)

	handler := NewForwardHandler(ForwardHandlerConfig{
		ClickUpClient: mockClient,
		HistoryStore:  store,
		Logger:        logger,
		Enabled:       true,
	})

	msg := &domain.Message{
		Timestamp: "1704153600.000001",
		UserID:    "U123",
		Text:      "테스트 메시지",
		ChannelID: "C123",
		CreatedAt: time.Now(),
	}
	event := domain.NewMessageEvent(msg)

	handler.Handle(event)

	// ClickUp 클라이언트가 호출되었는지 확인
	if mockClient.callCount != 1 {
		t.Errorf("ClickUp 클라이언트가 호출되지 않음: %d", mockClient.callCount)
	}

	// 히스토리에 저장되었는지 확인
	if store.Count() != 1 {
		t.Errorf("히스토리에 저장되지 않음: %d", store.Count())
	}

	// 성공 기록 확인
	records := store.GetAll()
	if !records[0].Success {
		t.Error("성공으로 기록되어야 함")
	}

	// 로그 확인
	output := buf.String()
	if !strings.Contains(output, "전송 성공") {
		t.Errorf("성공 로그가 없음: %s", output)
	}
}

// TestForwardHandler_Disabled는 비활성화 시 처리를 테스트합니다.
func TestForwardHandler_Disabled(t *testing.T) {
	mockClient := &MockClickUpClient{}
	store := history.NewMemoryStore(100)

	handler := NewForwardHandler(ForwardHandlerConfig{
		ClickUpClient: mockClient,
		HistoryStore:  store,
		Logger:        log.New(&bytes.Buffer{}, "", 0),
		Enabled:       false, // 비활성화
	})

	event := domain.NewMessageEvent(&domain.Message{Text: "test"})
	handler.Handle(event)

	// 비활성화 시 클라이언트가 호출되지 않아야 함
	if mockClient.callCount != 0 {
		t.Error("비활성화 시 클라이언트가 호출됨")
	}
}

// TestForwardHandler_Error는 에러 처리를 테스트합니다.
func TestForwardHandler_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	mockClient := &MockClickUpClient{shouldFail: true}
	store := history.NewMemoryStore(100)

	handler := NewForwardHandler(ForwardHandlerConfig{
		ClickUpClient: mockClient,
		HistoryStore:  store,
		Logger:        logger,
		Enabled:       true,
	})

	event := domain.NewMessageEvent(&domain.Message{Text: "test", CreatedAt: time.Now()})
	handler.Handle(event)

	// 히스토리에 실패로 기록되어야 함
	records := store.GetAll()
	if records[0].Success {
		t.Error("실패로 기록되어야 함")
	}
	if records[0].ErrorMessage == "" {
		t.Error("에러 메시지가 있어야 함")
	}
}

// TestChainHandler는 체인 핸들러를 테스트합니다.
func TestChainHandler(t *testing.T) {
	callOrder := []string{}

	handler1 := &testHandler{name: "h1", callOrder: &callOrder}
	handler2 := &testHandler{name: "h2", callOrder: &callOrder}

	chain := NewChainHandler(handler1, handler2)
	chain.Handle(&domain.Event{Type: domain.EventTypeNewMessage})

	if len(callOrder) != 2 {
		t.Errorf("두 핸들러가 호출되어야 함: %d", len(callOrder))
	}
	if callOrder[0] != "h1" || callOrder[1] != "h2" {
		t.Errorf("순서가 올바르지 않음: %v", callOrder)
	}
}

type testHandler struct {
	name      string
	callOrder *[]string
}

func (h *testHandler) Handle(event *domain.Event) {
	*h.callOrder = append(*h.callOrder, h.name)
}
