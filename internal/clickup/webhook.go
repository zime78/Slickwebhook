package clickup

// Webhook 이벤트 타입 상수
const (
	EventTaskCreated       = "taskCreated"
	EventTaskUpdated       = "taskUpdated"
	EventTaskStatusUpdated = "taskStatusUpdated"
	EventTaskMoved         = "taskMoved"
	EventTaskDeleted       = "taskDeleted"
)

// WebhookEvent는 ClickUp 웹훅 이벤트입니다.
type WebhookEvent struct {
	Event        string        `json:"event"`
	TaskID       string        `json:"task_id"`
	WebhookID    string        `json:"webhook_id"`
	HistoryItems []HistoryItem `json:"history_items"`
}

// HistoryItem은 변경 이력 항목입니다.
type HistoryItem struct {
	Date   int64       `json:"date"` // Unix 밀리초
	Field  string      `json:"field"`
	User   WebhookUser `json:"user"`
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

// WebhookUser는 웹훅 이벤트를 발생시킨 사용자입니다.
type WebhookUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// WebhookRegistration은 웹훅 등록 요청입니다.
type WebhookRegistration struct {
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
	FolderID *string  `json:"folder_id"`
	ListID   *string  `json:"list_id"`
	SpaceID  *string  `json:"space_id"`
}

// WebhookRegistrationResponse는 웹훅 등록 응답입니다.
type WebhookRegistrationResponse struct {
	ID       string `json:"id"`
	Webhook  Webhook `json:"webhook"`
}

// Webhook은 등록된 웹훅 정보입니다.
type Webhook struct {
	ID       string   `json:"id"`
	TeamID   string   `json:"team_id"`
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
	Health   WebhookHealth `json:"health"`
	Secret   string   `json:"secret"`
}

// WebhookHealth는 웹훅 상태 정보입니다.
type WebhookHealth struct {
	Status    string `json:"status"`
	FailCount int    `json:"fail_count"`
}

// GetListIDFromEvent는 웹훅 이벤트에서 리스트 ID를 추출합니다.
// HistoryItems의 parent_id 필드에서 리스트 ID를 찾습니다.
func (e *WebhookEvent) GetListIDFromEvent() string {
	for _, item := range e.HistoryItems {
		if item.Field == "parent_id" {
			if listID, ok := item.After.(string); ok {
				return listID
			}
		}
	}
	return ""
}
