package clickup

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"time"

"github.com/zime/slickwebhook/internal/domain"
)

// Client는 ClickUp API와 상호작용하는 인터페이스입니다.
type Client interface {
	CreateTask(ctx context.Context, msg *domain.Message) (*TaskResponse, error)
}

// TaskResponse는 ClickUp 태스크 생성 응답입니다.
type TaskResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config는 ClickUp 클라이언트 설정입니다.
type Config struct {
	APIToken   string
	ListID     string
	AssigneeID int
}

// ClickUpClient는 실제 ClickUp API 클라이언트입니다.
type ClickUpClient struct {
	config     Config
	httpClient *http.Client
	baseURL    string
}

// NewClickUpClient는 새로운 ClickUpClient를 생성합니다.
func NewClickUpClient(config Config) *ClickUpClient {
	if config.AssigneeID == 0 {
		config.AssigneeID = 288777246
	}

	return &ClickUpClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.clickup.com/api/v2",
	}
}

// taskPayload는 ClickUp 태스크 생성 요청 페이로드입니다.
type taskPayload struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Assignees   []int    `json:"assignees"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
}

// CreateTask는 메시지를 기반으로 ClickUp 태스크를 생성합니다.
func (c *ClickUpClient) CreateTask(ctx context.Context, msg *domain.Message) (*TaskResponse, error) {
	var name, description string
	var tags []string

	// 소스에 따라 다른 포맷 적용
	if msg.Source == "email" {
		name, description, tags = c.formatEmailTask(msg)
	} else {
		name, description, tags = c.formatSlackTask(msg)
	}

	payload := taskPayload{
		Name:        name,
		Description: description,
		Assignees:   []int{c.config.AssigneeID},
		Priority:    3,
		Tags:        tags,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("페이로드 직렬화 실패: %w", err)
	}

	url := fmt.Sprintf("%s/list/%s/task", c.baseURL, c.config.ListID)

	// 재시도 로직 (최대 3회)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, url, payloadBytes)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("3회 재시도 후 실패: %w", lastErr)
}

// formatEmailTask는 이메일용 태스크 포맷을 생성합니다.
func (c *ClickUpClient) formatEmailTask(msg *domain.Message) (name, description string, tags []string) {
	// 제목 사용 (없으면 본문 앞 50자)
	if msg.Subject != "" {
		name = msg.Subject
	} else {
		name = fmt.Sprintf("[Email] %s", truncateText(msg.Text, 50))
	}

	// 설명 생성
	description = fmt.Sprintf(`📧 이메일 자동 수집

**발신자:** %s
**제목:** %s
**수신 시간:** %s

---

%s

---
*이 태스크는 Email Monitor에 의해 자동 생성되었습니다.*`,
msg.From,
msg.Subject,
msg.CreatedAt.Format("2006-01-02 15:04:05"),
msg.Text,
)

	tags = []string{"auto-generated", "email"}
	return
}

// formatSlackTask는 Slack용 태스크 포맷을 생성합니다.
func (c *ClickUpClient) formatSlackTask(msg *domain.Message) (name, description string, tags []string) {
	name = fmt.Sprintf("[Slack 이벤트] %s", truncateText(msg.Text, 50))

	description = fmt.Sprintf(`📨 Slack 채널 메시지 자동 수집

**원문 메시지:**
> %s

**메시지 정보:**
- 채널 ID: %s
- 유저 ID: %s
- 수신 시간: %s
- 타임스탬프: %s

---
*이 태스크는 SlickWebhook 모니터에 의해 자동 생성되었습니다.*`,
msg.Text,
msg.ChannelID,
msg.UserID,
msg.CreatedAt.Format(time.RFC3339),
msg.Timestamp,
)

	tags = []string{"auto-generated"}
	return
}

func (c *ClickUpClient) doRequest(ctx context.Context, url string, payload []byte) (*TaskResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 에러 (상태코드: %d): %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return &taskResp, nil
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}
