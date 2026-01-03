package handler

import (
	"fmt"
	"log"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
)

// EventHandler는 이벤트를 처리하는 인터페이스입니다.
type EventHandler interface {
	// Handle은 이벤트를 처리합니다.
	Handle(event *domain.Event)
}

// LogHandler는 이벤트를 로그로 출력하는 핸들러입니다.
type LogHandler struct {
	logger *log.Logger
}

// NewLogHandler는 새로운 LogHandler를 생성합니다.
func NewLogHandler(logger *log.Logger) *LogHandler {
	return &LogHandler{
		logger: logger,
	}
}

// Handle은 이벤트를 로그로 출력합니다.
func (h *LogHandler) Handle(event *domain.Event) {
	switch event.Type {
	case domain.EventTypeNewMessage:
		h.handleNewMessage(event)
	case domain.EventTypeError:
		h.handleError(event)
	default:
		h.logger.Printf("[WARN] ⚠️ 알 수 없는 이벤트 타입: %s\n", event.Type)
	}
}

// handleNewMessage는 새 메시지 이벤트를 처리합니다.
func (h *LogHandler) handleNewMessage(event *domain.Event) {
	msg := event.Message
	if msg == nil {
		h.logger.Println("[WARN] ⚠️ 메시지가 nil입니다")
		return
	}

	h.logger.Printf("[EVENT] 📨 새 메시지 감지\n")
	h.logger.Printf("  - 시간: %s\n", msg.CreatedAt.Format(time.RFC3339))
	h.logger.Printf("  - 유저: %s\n", msg.UserID)
	h.logger.Printf("  - 채널: %s\n", msg.ChannelID)
	h.logger.Printf("  - 내용: %s\n", truncateText(msg.Text, 100))
}

// handleError는 에러 이벤트를 처리합니다.
func (h *LogHandler) handleError(event *domain.Event) {
	if event.Error == nil {
		h.logger.Println("[WARN] ⚠️ 에러가 nil입니다")
		return
	}
	h.logger.Printf("[ERROR] ❌ 에러 발생: %v\n", event.Error)
}

// truncateText는 텍스트를 지정된 길이로 자릅니다.
func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return fmt.Sprintf("%s...", string(runes[:maxLen]))
}
