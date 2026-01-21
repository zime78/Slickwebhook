package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client는 Jira API 클라이언트입니다.
type Client interface {
	GetIssueSummary(ctx context.Context, issueKey string) (string, error)
}

// HTTPClient는 실제 Jira API를 호출하는 클라이언트입니다.
type HTTPClient struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
	cache      *issueCache
}

// issueCache는 이슈 정보를 캐싱합니다.
type issueCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	summary   string
	expiresAt time.Time
}

// ClientConfig는 Jira 클라이언트 설정입니다.
type ClientConfig struct {
	BaseURL  string // 예: https://kakaovx.atlassian.net
	Email    string // 예: zime.lee@kakaovx.com
	APIToken string // Jira API 토큰
	CacheTTL time.Duration
}

// NewClient는 새로운 Jira 클라이언트를 생성합니다.
func NewClient(config ClientConfig) *HTTPClient {
	ttl := config.CacheTTL
	if ttl == 0 {
		ttl = 10 * time.Minute // 기본 캐시 TTL
	}

	return &HTTPClient{
		baseURL:  config.BaseURL,
		email:    config.Email,
		apiToken: config.APIToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &issueCache{
			entries: make(map[string]cacheEntry),
			ttl:     ttl,
		},
	}
}

// issueResponse는 Jira API 응답 구조체입니다.
type issueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// GetIssueSummary는 이슈 키로 요약(제목)을 가져옵니다.
func (c *HTTPClient) GetIssueSummary(ctx context.Context, issueKey string) (string, error) {
	// 캐시 확인
	if summary, ok := c.cache.get(issueKey); ok {
		return summary, nil
	}

	// API 호출
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary", c.baseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("request 생성 실패: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API 오류 (status=%d): %s", resp.StatusCode, string(body))
	}

	var issue issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return "", fmt.Errorf("응답 파싱 실패: %w", err)
	}

	// 캐시에 저장
	c.cache.set(issueKey, issue.Fields.Summary)

	return issue.Fields.Summary, nil
}

// get은 캐시에서 이슈 요약을 가져옵니다.
func (c *issueCache) get(issueKey string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[issueKey]
	if !ok {
		return "", false
	}

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.summary, true
}

// set은 캐시에 이슈 요약을 저장합니다.
func (c *issueCache) set(issueKey, summary string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[issueKey] = cacheEntry{
		summary:   summary,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Attachment는 Jira 첨부파일 정보입니다.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"` // 다운로드 URL
	Size     int    `json:"size"`
}

// IssueDetail은 이슈 상세 정보입니다.
type IssueDetail struct {
	Key         string
	Summary     string
	Description string       // 텍스트로 변환된 본문
	Attachments []Attachment // 첨부파일 목록
}

// issueDetailResponse는 Jira API 상세 응답 구조체입니다.
type issueDetailResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string          `json:"summary"`
		Description json.RawMessage `json:"description"` // ADF 형식
		Attachment  []Attachment    `json:"attachment"`
	} `json:"fields"`
}

// GetIssueDetail은 이슈의 상세 정보(본문, 첨부파일)를 가져옵니다.
func (c *HTTPClient) GetIssueDetail(ctx context.Context, issueKey string) (*IssueDetail, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary,description,attachment", c.baseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request 생성 실패: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류 (status=%d): %s", resp.StatusCode, string(body))
	}

	var issue issueDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	// ADF(Atlassian Document Format)를 텍스트로 변환
	descText := parseADFToText(issue.Fields.Description)

	return &IssueDetail{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Description: descText,
		Attachments: issue.Fields.Attachment,
	}, nil
}

// DownloadAttachment는 첨부파일을 다운로드합니다.
func (c *HTTPClient) DownloadAttachment(ctx context.Context, contentURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request 생성 실패: %w", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("다운로드 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("다운로드 오류 (status=%d)", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// parseADFToText는 Atlassian Document Format을 텍스트로 변환합니다.
func parseADFToText(adfData json.RawMessage) string {
	if len(adfData) == 0 {
		return ""
	}

	var adf map[string]interface{}
	if err := json.Unmarshal(adfData, &adf); err != nil {
		return ""
	}

	var result string
	extractText(adf, &result)
	return result
}

// extractText는 ADF 노드에서 텍스트를 재귀적으로 추출합니다.
func extractText(node map[string]interface{}, result *string) {
	// 텍스트 노드 처리
	if nodeType, ok := node["type"].(string); ok {
		if nodeType == "text" {
			if text, ok := node["text"].(string); ok {
				*result += text
			}
		}
		// 단락 끝에 줄바꿈 추가
		if nodeType == "paragraph" || nodeType == "heading" {
			defer func() { *result += "\n" }()
		}
		// 리스트 아이템
		if nodeType == "listItem" {
			defer func() { *result += "\n" }()
		}
	}

	// 자식 노드 처리
	if content, ok := node["content"].([]interface{}); ok {
		for _, child := range content {
			if childMap, ok := child.(map[string]interface{}); ok {
				extractText(childMap, result)
			}
		}
	}
}
