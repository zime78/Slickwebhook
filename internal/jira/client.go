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
