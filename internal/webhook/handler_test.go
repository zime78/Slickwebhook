package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zime/slickwebhook/internal/clickup"
)

// MockProcessor는 테스트용 프로세서입니다.
type MockProcessor struct {
	EnqueuedTasks []EnqueuedTask
}

type EnqueuedTask struct {
	TaskID string
	ListID string
}

func (m *MockProcessor) EnqueueTask(taskID, listID string) {
	m.EnqueuedTasks = append(m.EnqueuedTasks, EnqueuedTask{TaskID: taskID, ListID: listID})
}

func (m *MockProcessor) IsAIList(listID string) bool {
	return listID == "ai-list-1" || listID == "ai-list-2"
}

// TestHandler_HandleWebhook은 웹훅 핸들링을 테스트합니다.
func TestHandler_HandleWebhook(t *testing.T) {
	mockProcessor := &MockProcessor{}
	handler := NewHandler(mockProcessor, "test-secret")

	// 웹훅 이벤트 생성
	event := clickup.WebhookEvent{
		Event:     clickup.EventTaskCreated,
		TaskID:    "task123",
		WebhookID: "webhook456",
		HistoryItems: []clickup.HistoryItem{
			{
				Field: "parent_id",
				After: "ai-list-1",
			},
		},
	}

	payload, _ := json.Marshal(event)
	signature := computeSignature(payload, "test-secret")

	req := httptest.NewRequest("POST", "/webhook/clickup", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusOK)
	}

	if len(mockProcessor.EnqueuedTasks) != 1 {
		t.Errorf("큐에 추가된 태스크 개수 불일치: got %d, want 1", len(mockProcessor.EnqueuedTasks))
	}
}

// TestHandler_HandleWebhook_InvalidSignature는 잘못된 서명을 테스트합니다.
func TestHandler_HandleWebhook_InvalidSignature(t *testing.T) {
	mockProcessor := &MockProcessor{}
	handler := NewHandler(mockProcessor, "test-secret")

	event := clickup.WebhookEvent{
		Event:  clickup.EventTaskCreated,
		TaskID: "task123",
	}
	payload, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhook/clickup", bytes.NewReader(payload))
	req.Header.Set("X-Signature", "invalid-signature")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestHandler_HandleWebhook_NonAIList는 AI 리스트가 아닌 경우를 테스트합니다.
func TestHandler_HandleWebhook_NonAIList(t *testing.T) {
	mockProcessor := &MockProcessor{}
	handler := NewHandler(mockProcessor, "test-secret")

	event := clickup.WebhookEvent{
		Event:  clickup.EventTaskCreated,
		TaskID: "task123",
		HistoryItems: []clickup.HistoryItem{
			{
				Field: "parent_id",
				After: "non-ai-list",
			},
		},
	}
	payload, _ := json.Marshal(event)
	signature := computeSignature(payload, "test-secret")

	req := httptest.NewRequest("POST", "/webhook/clickup", bytes.NewReader(payload))
	req.Header.Set("X-Signature", signature)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusOK)
	}

	// AI 리스트가 아니므로 큐에 추가되지 않아야 함
	if len(mockProcessor.EnqueuedTasks) != 0 {
		t.Errorf("AI 리스트가 아닌 경우 큐에 추가되면 안됨: got %d", len(mockProcessor.EnqueuedTasks))
	}
}

// TestVerifySignature는 서명 검증을 테스트합니다.
func TestVerifySignature(t *testing.T) {
	handler := NewHandler(nil, "secret-key")

	payload := []byte(`{"event": "taskCreated"}`)
	validSig := computeSignature(payload, "secret-key")

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{"유효한 서명", validSig, true},
		{"잘못된 서명", "invalid", false},
		{"빈 서명", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.VerifySignature(payload, tt.signature)
			if got != tt.want {
				t.Errorf("VerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 테스트용 서명 계산 함수
func computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
