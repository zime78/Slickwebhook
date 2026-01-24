package clickup

import (
	"encoding/json"
	"testing"
)

// TestWebhookEvent_Parse는 웹훅 이벤트 JSON 파싱을 테스트합니다.
func TestWebhookEvent_Parse(t *testing.T) {
	jsonData := `{
		"event": "taskCreated",
		"task_id": "task123",
		"webhook_id": "webhook456",
		"history_items": [
			{
				"date": 1704153600000,
				"field": "status",
				"user": {
					"id": 12345,
					"username": "testuser",
					"email": "test@example.com"
				},
				"before": null,
				"after": "Open"
			}
		]
	}`

	var event WebhookEvent
	err := json.Unmarshal([]byte(jsonData), &event)

	if err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}
	if event.Event != EventTaskCreated {
		t.Errorf("이벤트 타입 불일치: got %s, want %s", event.Event, EventTaskCreated)
	}
	if event.TaskID != "task123" {
		t.Errorf("태스크 ID 불일치: %s", event.TaskID)
	}
	if len(event.HistoryItems) != 1 {
		t.Errorf("히스토리 항목 개수 불일치: %d", len(event.HistoryItems))
	}
	if event.HistoryItems[0].User.Username != "testuser" {
		t.Errorf("사용자 이름 불일치: %s", event.HistoryItems[0].User.Username)
	}
}

// TestWebhookEvent_GetListIDFromEvent는 리스트 ID 추출을 테스트합니다.
func TestWebhookEvent_GetListIDFromEvent(t *testing.T) {
	event := WebhookEvent{
		Event:  EventTaskCreated,
		TaskID: "task123",
		HistoryItems: []HistoryItem{
			{
				Field: "parent_id",
				After: "list789",
			},
		},
	}

	listID := event.GetListIDFromEvent()

	if listID != "list789" {
		t.Errorf("리스트 ID 불일치: got %s, want list789", listID)
	}
}

// TestWebhookEvent_GetListIDFromEvent_NotFound는 리스트 ID가 없는 경우를 테스트합니다.
func TestWebhookEvent_GetListIDFromEvent_NotFound(t *testing.T) {
	event := WebhookEvent{
		Event:  EventTaskUpdated,
		TaskID: "task123",
		HistoryItems: []HistoryItem{
			{
				Field: "status",
				After: "In Progress",
			},
		},
	}

	listID := event.GetListIDFromEvent()

	if listID != "" {
		t.Errorf("빈 문자열이어야 함: got %s", listID)
	}
}

// TestWebhookRegistration_Marshal는 웹훅 등록 요청 직렬화를 테스트합니다.
func TestWebhookRegistration_Marshal(t *testing.T) {
	reg := WebhookRegistration{
		Endpoint: "https://example.com/webhook",
		Events:   []string{EventTaskCreated, EventTaskUpdated},
	}

	data, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("직렬화 실패: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["endpoint"] != "https://example.com/webhook" {
		t.Error("endpoint 불일치")
	}
	if events, ok := parsed["events"].([]interface{}); !ok || len(events) != 2 {
		t.Error("events 불일치")
	}
}
