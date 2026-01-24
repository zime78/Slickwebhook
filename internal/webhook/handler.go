package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/zime/slickwebhook/internal/clickup"
)

// Processor는 웹훅 이벤트를 처리하는 인터페이스입니다.
type Processor interface {
	EnqueueTask(taskID, listID string)
	IsAIList(listID string) bool
}

// Handler는 ClickUp 웹훅을 처리합니다.
type Handler struct {
	processor Processor
	secret    string
	logger    *log.Logger
}

// NewHandler는 새 Handler를 생성합니다.
func NewHandler(processor Processor, secret string) *Handler {
	return &Handler{
		processor: processor,
		secret:    secret,
	}
}

// SetLogger는 로거를 설정합니다.
func (h *Handler) SetLogger(logger *log.Logger) {
	h.logger = logger
}

// HandleWebhook은 ClickUp 웹훅을 처리합니다.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// POST 메서드만 허용
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 페이로드 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logError("페이로드 읽기 실패: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 서명 검증
	signature := r.Header.Get("X-Signature")
	if !h.VerifySignature(body, signature) {
		h.logError("서명 검증 실패")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// 이벤트 파싱
	var event clickup.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logError("이벤트 파싱 실패: %v", err)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	// 이벤트 처리
	h.processEvent(&event)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// processEvent는 웹훅 이벤트를 처리합니다.
func (h *Handler) processEvent(event *clickup.WebhookEvent) {
	h.logInfo("웹훅 이벤트 수신: %s, 태스크: %s", event.Event, event.TaskID)

	// 리스트 ID 추출
	listID := event.GetListIDFromEvent()
	if listID == "" {
		h.logInfo("리스트 ID를 찾을 수 없음, 이벤트 무시")
		return
	}

	// AI 리스트인지 확인
	if !h.processor.IsAIList(listID) {
		h.logInfo("AI 리스트가 아님: %s, 이벤트 무시", listID)
		return
	}

	// 처리할 이벤트 타입 확인
	switch event.Event {
	case clickup.EventTaskCreated, clickup.EventTaskUpdated, clickup.EventTaskStatusUpdated:
		h.logInfo("태스크를 큐에 추가: %s (리스트: %s)", event.TaskID, listID)
		h.processor.EnqueueTask(event.TaskID, listID)
	default:
		h.logInfo("처리하지 않는 이벤트 타입: %s", event.Event)
	}
}

// VerifySignature는 웹훅 서명을 검증합니다.
func (h *Handler) VerifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h *Handler) logInfo(format string, args ...interface{}) {
	if h.logger != nil {
		h.logger.Printf("[Webhook] "+format, args...)
	}
}

func (h *Handler) logError(format string, args ...interface{}) {
	if h.logger != nil {
		h.logger.Printf("[Webhook ERROR] "+format, args...)
	}
}
