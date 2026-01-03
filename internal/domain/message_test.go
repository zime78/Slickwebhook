package domain

import (
	"errors"
	"testing"
	"time"
)

// TestMessage_Creation은 Message 구조체 생성을 테스트합니다.
func TestMessage_Creation(t *testing.T) {
	// Given: 메시지 생성에 필요한 데이터
	timestamp := "1704153600.000001"
	userID := "U123ABC456"
	text := "안녕하세요, 테스트 메시지입니다"
	channelID := "C07AFHKESVC"
	createdAt := time.Now()

	// When: Message 구조체 생성
	msg := Message{
		Timestamp: timestamp,
		UserID:    userID,
		Text:      text,
		ChannelID: channelID,
		CreatedAt: createdAt,
	}

	// Then: 모든 필드가 올바르게 설정됨
	if msg.Timestamp != timestamp {
		t.Errorf("Timestamp가 일치하지 않음: got %s, want %s", msg.Timestamp, timestamp)
	}
	if msg.UserID != userID {
		t.Errorf("UserID가 일치하지 않음: got %s, want %s", msg.UserID, userID)
	}
	if msg.Text != text {
		t.Errorf("Text가 일치하지 않음: got %s, want %s", msg.Text, text)
	}
	if msg.ChannelID != channelID {
		t.Errorf("ChannelID가 일치하지 않음: got %s, want %s", msg.ChannelID, channelID)
	}
}

// TestNewMessageEvent는 새 메시지 이벤트 생성을 테스트합니다.
func TestNewMessageEvent(t *testing.T) {
	// Given: 테스트용 메시지
	msg := &Message{
		Timestamp: "1704153600.000001",
		UserID:    "U123ABC456",
		Text:      "테스트 메시지",
		ChannelID: "C07AFHKESVC",
		CreatedAt: time.Now(),
	}

	// When: 새 메시지 이벤트 생성
	event := NewMessageEvent(msg)

	// Then: 이벤트가 올바르게 생성됨
	if event.Type != EventTypeNewMessage {
		t.Errorf("이벤트 타입이 일치하지 않음: got %s, want %s", event.Type, EventTypeNewMessage)
	}
	if event.Message != msg {
		t.Error("메시지가 올바르게 설정되지 않음")
	}
	if event.OccurredAt.IsZero() {
		t.Error("OccurredAt이 설정되지 않음")
	}
}

// TestNewErrorEvent는 에러 이벤트 생성을 테스트합니다.
func TestNewErrorEvent(t *testing.T) {
	// Given: 테스트용 에러
	testErr := errors.New("테스트 에러: API 호출 실패")

	// When: 에러 이벤트 생성
	event := NewErrorEvent(testErr)

	// Then: 이벤트가 올바르게 생성됨
	if event.Type != EventTypeError {
		t.Errorf("이벤트 타입이 일치하지 않음: got %s, want %s", event.Type, EventTypeError)
	}
	if event.Error != testErr {
		t.Error("에러가 올바르게 설정되지 않음")
	}
	if event.OccurredAt.IsZero() {
		t.Error("OccurredAt이 설정되지 않음")
	}
}

// TestEventType_Constants는 이벤트 타입 상수를 테스트합니다.
func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      EventType
		expected string
	}{
		{"EventTypeNewMessage", EventTypeNewMessage, "new_message"},
		{"EventTypeError", EventTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.got, tt.expected)
			}
		})
	}
}
