package domain

import "time"

// Message는 Slack 채널에서 수신된 메시지를 나타냅니다.
type Message struct {
	// Timestamp는 Slack 메시지의 고유 식별자입니다 (ts 필드)
	Timestamp string
	// UserID는 메시지를 보낸 사용자의 ID입니다
	UserID string
	// BotID는 메시지를 보낸 봇의 ID입니다 (봇 메시지인 경우)
	BotID string
	// Text는 메시지 본문입니다
	Text string
	// ChannelID는 메시지가 발생한 채널 ID입니다
	ChannelID string
	// CreatedAt는 메시지가 작성된 시간입니다
	CreatedAt time.Time
}

// EventType은 이벤트의 종류를 나타냅니다.
type EventType string

const (
	// EventTypeNewMessage는 새 메시지 수신 이벤트입니다
	EventTypeNewMessage EventType = "new_message"
	// EventTypeError는 에러 발생 이벤트입니다
	EventTypeError EventType = "error"
)

// Event는 모니터링 중 발생한 이벤트를 나타냅니다.
type Event struct {
	// Type은 이벤트 종류입니다
	Type EventType
	// Message는 이벤트에 연관된 메시지입니다 (EventTypeNewMessage일 때 사용)
	Message *Message
	// Error는 에러 정보입니다 (EventTypeError일 때 사용)
	Error error
	// OccurredAt는 이벤트 발생 시간입니다
	OccurredAt time.Time
}

// NewMessageEvent는 새 메시지 이벤트를 생성합니다.
func NewMessageEvent(msg *Message) *Event {
	return &Event{
		Type:       EventTypeNewMessage,
		Message:    msg,
		OccurredAt: time.Now(),
	}
}

// NewErrorEvent는 에러 이벤트를 생성합니다.
func NewErrorEvent(err error) *Event {
	return &Event{
		Type:       EventTypeError,
		Error:      err,
		OccurredAt: time.Now(),
	}
}
