package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
)

// Client는 ClickUp API와 상호작용하는 인터페이스입니다.
type Client interface {
	CreateTask(ctx context.Context, msg *domain.Message) (*TaskResponse, error)
	UploadAttachment(ctx context.Context, taskID string, filename string, data []byte) error
	GetTask(ctx context.Context, taskID string) (*Task, error)
	GetTasks(ctx context.Context, listID string, opts *GetTasksOptions) ([]*Task, error)
	UpdateTask(ctx context.Context, taskID string, req UpdateTaskRequest) error
	MoveTaskToList(ctx context.Context, taskID string, listID string) error
}

// GetTasksOptions는 태스크 목록 조회 옵션입니다.
type GetTasksOptions struct {
	OrderBy       string   // 정렬 기준: "created", "updated", "due_date"
	Reverse       bool     // true면 오름차순 (오래된 순)
	Statuses      []string // 필터링할 상태 목록
	IncludeClosed bool     // 완료된 태스크 포함 여부
}

// GetTasksResponse는 태스크 목록 조회 응답입니다.
type GetTasksResponse struct {
	Tasks []*Task `json:"tasks"`
}

// TaskResponse는 ClickUp 태스크 생성 응답입니다.
type TaskResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Task는 ClickUp 태스크 조회 응답입니다.
type Task struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      TaskStatus   `json:"status"`
	URL         string       `json:"url"`
	DateCreated string       `json:"date_created"`
	DateUpdated string       `json:"date_updated"`
	Attachments []Attachment `json:"attachments"`
}

// TaskStatus는 태스크 상태 정보입니다.
type TaskStatus struct {
	Status string `json:"status"`
	Color  string `json:"color"`
}

// Attachment는 태스크 첨부파일 정보입니다.
type Attachment struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Extension       string `json:"extension"`
	URL             string `json:"url"`
	Size            int64  `json:"size"`
	MimeType        string `json:"mimetype"`
	ThumbnailSmall  string `json:"thumbnail_small"`
	ThumbnailMedium string `json:"thumbnail_medium"`
	ThumbnailLarge  string `json:"thumbnail_large"`
}

// Config는 ClickUp 클라이언트 설정입니다.
type Config struct {
	APIToken    string
	ListID      string
	AssigneeID  int
	JiraBaseURL string // Jira 이슈 링크용 (예: https://example.atlassian.net)
	TeamID      string // Workspace ID (리스트 이동용)
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

	// Jira 이슈 키 추출 및 링크 생성
	jiraLinks := c.extractJiraLinks(msg.Subject, msg.Text)

	// 설명 생성 (간소화된 포맷)
	var descBuilder strings.Builder
	descBuilder.WriteString(fmt.Sprintf("**수신 시간:** %s",
		msg.CreatedAt.Format("2006-01-02 15:04:05"),
	))

	// Jira 링크가 있으면 추가
	if jiraLinks != "" {
		descBuilder.WriteString(fmt.Sprintf("\n**🔗 Jira 이슈:** %s", jiraLinks))
	}

	descBuilder.WriteString(fmt.Sprintf("\n\n---\n\n%s", msg.Text))

	description = descBuilder.String()
	tags = []string{"auto-generated"}
	return
}

// extractJiraLinks는 텍스트에서 Jira 이슈 키를 추출하고 링크를 생성합니다.
func (c *ClickUpClient) extractJiraLinks(subject, body string) string {
	if c.config.JiraBaseURL == "" {
		return ""
	}

	// Jira 이슈 키 패턴 (예: ITSM-1234, PROJ-123)
	issuePattern := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

	// 제목과 본문에서 이슈 키 추출
	combinedText := subject + " " + body
	matches := issuePattern.FindAllString(combinedText, -1)

	if len(matches) == 0 {
		return ""
	}

	// 중복 제거
	seen := make(map[string]bool)
	var uniqueKeys []string
	for _, key := range matches {
		if !seen[key] {
			seen[key] = true
			uniqueKeys = append(uniqueKeys, key)
		}
	}

	// 링크 생성 (마크다운 형식)
	baseURL := strings.TrimSuffix(c.config.JiraBaseURL, "/")
	var links []string
	for _, key := range uniqueKeys {
		links = append(links, fmt.Sprintf("[%s](%s/browse/%s)", key, baseURL, key))
	}

	return strings.Join(links, ", ")
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

// GetTask는 태스크 ID로 태스크 정보를 조회합니다.
func (c *ClickUpClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	url := fmt.Sprintf("%s/task/%s", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", c.config.APIToken)

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

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return &task, nil
}

// UploadAttachment는 태스크에 첨부파일을 업로드합니다.
func (c *ClickUpClient) UploadAttachment(ctx context.Context, taskID string, filename string, data []byte) error {
	url := fmt.Sprintf("%s/task/%s/attachment", c.baseURL, taskID)

	// multipart form 생성
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil {
		return fmt.Errorf("form file 생성 실패: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("파일 데이터 쓰기 실패: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("multipart writer 닫기 실패: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", c.config.APIToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("업로드 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("업로드 오류 (status=%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetTasks는 리스트의 태스크 목록을 조회합니다.
// API: GET /api/v2/list/{list_id}/task
func (c *ClickUpClient) GetTasks(ctx context.Context, listID string, opts *GetTasksOptions) ([]*Task, error) {
	reqURL := fmt.Sprintf("%s/list/%s/task", c.baseURL, listID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}

	// 쿼리 파라미터 구성
	q := req.URL.Query()
	if opts != nil {
		if opts.OrderBy != "" {
			q.Set("order_by", opts.OrderBy)
		}
		if opts.Reverse {
			q.Set("reverse", "true")
		}
		for _, status := range opts.Statuses {
			q.Add("statuses[]", status)
		}
		if opts.IncludeClosed {
			q.Set("include_closed", "true")
		}
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", c.config.APIToken)

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

	var tasksResp GetTasksResponse
	if err := json.Unmarshal(body, &tasksResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return tasksResp.Tasks, nil
}

// UpdateTaskRequest는 태스크 업데이트 요청 페이로드입니다.
// ClickUp API는 날짜를 Unix milliseconds (타임스탬프 * 1000)로 받습니다.
type UpdateTaskRequest struct {
	Status    string `json:"status,omitempty"`
	StartDate int64  `json:"start_date,omitempty"` // Unix milliseconds
	DueDate   int64  `json:"due_date,omitempty"`   // Unix milliseconds
}

// UpdateTask는 태스크의 상태와 날짜 정보를 변경합니다.
// API: PUT /api/v2/task/{task_id}
func (c *ClickUpClient) UpdateTask(ctx context.Context, taskID string, req UpdateTaskRequest) error {
	reqURL := fmt.Sprintf("%s/task/%s", c.baseURL, taskID)

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("페이로드 직렬화 실패: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", reqURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %w", err)
	}

	httpReq.Header.Set("Authorization", c.config.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 에러 (상태코드: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// MoveTaskToList는 태스크를 다른 리스트로 이동합니다.
// API: PUT /api/v3/workspaces/{workspace_id}/tasks/{task_id}/home_list/{list_id}
func (c *ClickUpClient) MoveTaskToList(ctx context.Context, taskID string, listID string) error {
	if c.config.TeamID == "" {
		return fmt.Errorf("리스트 이동 실패: TeamID가 설정되지 않음")
	}

	// v3 API 엔드포인트 사용. 테스트 환경 지원을 위해 baseURL 호스트 활용
	reqURL := fmt.Sprintf("%s/workspaces/%s/tasks/%s/home_list/%s", strings.ReplaceAll(c.baseURL, "/api/v2", "/api/v3"), c.config.TeamID, taskID, listID)

	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL, nil)
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 에러 (상태코드: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
