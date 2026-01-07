package handler

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/zime/slickwebhook/internal/domain"
)

// SlackNotifier는 Slack 메시지 전송 인터페이스입니다.
// 테스트 시 모킹이 가능하도록 인터페이스로 정의합니다.
type SlackNotifier interface {
	PostMessage(ctx context.Context, channelID string, blocks []slack.Block, text string) error
}

// SlackNotifyHandler는 이벤트를 Slack으로 알림 전송하는 핸들러입니다.
type SlackNotifyHandler struct {
	client      SlackNotifier
	channelID   string
	logger      *log.Logger
	enabled     bool
	jiraBaseURL string // Jira 이슈 링크용 (예: https://example.atlassian.net)
}

// SlackNotifyHandlerConfig는 SlackNotifyHandler 설정입니다.
type SlackNotifyHandlerConfig struct {
	Client      SlackNotifier
	ChannelID   string
	Logger      *log.Logger
	Enabled     bool
	JiraBaseURL string // Jira 이슈 링크용 (예: https://example.atlassian.net)
}

// NewSlackNotifyHandler는 새로운 SlackNotifyHandler를 생성합니다.
func NewSlackNotifyHandler(config SlackNotifyHandlerConfig) *SlackNotifyHandler {
	return &SlackNotifyHandler{
		client:      config.Client,
		channelID:   config.ChannelID,
		logger:      config.Logger,
		enabled:     config.Enabled,
		jiraBaseURL: config.JiraBaseURL,
	}
}

// Handle은 이벤트를 Slack으로 알림 전송합니다.
func (h *SlackNotifyHandler) Handle(event *domain.Event) {
	if !h.enabled {
		return
	}

	if event.Type != domain.EventTypeNewMessage {
		return
	}

	msg := event.Message
	if msg == nil {
		return
	}

	// 이메일 소스만 처리
	if msg.Source != "email" {
		return
	}

	// 상태 변경 이메일은 필터링 (Jira 상태 변경 알림 제외)
	if h.isStatusChangeEmail(msg) {
		h.logger.Printf("[SLACK_NOTIFY] ⏭️ 상태 변경 이메일 스킵: %s\n", msg.Subject)
		return
	}

	h.logger.Printf("[SLACK_NOTIFY] 📤 Slack 알림 전송 중...\n")

	blocks := h.buildEmailBlocks(msg)
	fallbackText := fmt.Sprintf("새 이메일: %s", msg.Subject)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.client.PostMessage(ctx, h.channelID, blocks, fallbackText); err != nil {
		h.logger.Printf("[SLACK_NOTIFY] ❌ 전송 실패: %v\n", err)
		return
	}

	h.logger.Printf("[SLACK_NOTIFY] ✅ 전송 성공!\n")
}

// buildEmailBlocks는 이메일용 Slack Block을 생성합니다.
func (h *SlackNotifyHandler) buildEmailBlocks(msg *domain.Message) []slack.Block {
	blocks := make([]slack.Block, 0, 6)

	// 1. Header Block
	headerText := slack.NewTextBlockObject(slack.PlainTextType, "📧 새 이메일 알림", true, false)
	blocks = append(blocks, slack.NewHeaderBlock(headerText))

	// 2. 메타 정보 Section Block
	metaText := fmt.Sprintf(
		"*발신자:* %s\n*제목:* %s\n*시간:* %s",
		escapeSlackText(msg.From),
		escapeSlackText(msg.Subject),
		msg.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	// Jira 링크가 있으면 추가
	jiraLinks := h.extractJiraLinks(msg.Subject, msg.Text)
	if jiraLinks != "" {
		metaText += fmt.Sprintf("\n*🔗 Jira 이슈:* %s", jiraLinks)
	}

	metaBlock := slack.NewTextBlockObject(slack.MarkdownType, metaText, false, false)
	blocks = append(blocks, slack.NewSectionBlock(metaBlock, nil, nil))

	// 3. 본문 미리보기 Section Block (최대 300자)
	preview := truncateTextForSlack(msg.Text, 300)
	if preview != "" {
		// 줄바꿈을 적절히 처리하고 인용 형식으로 표시
		preview = strings.ReplaceAll(preview, "\n", "\n> ")
		bodyText := fmt.Sprintf("> %s", escapeSlackText(preview))
		bodyBlock := slack.NewTextBlockObject(slack.MarkdownType, bodyText, false, false)
		blocks = append(blocks, slack.NewSectionBlock(bodyBlock, nil, nil))
	}

	// 4. Divider
	blocks = append(blocks, slack.NewDividerBlock())

	// 5. Context Block (푸터)
	contextText := slack.NewTextBlockObject(slack.PlainTextType, "Email Monitor 자동 알림", true, false)
	blocks = append(blocks, slack.NewContextBlock("", contextText))

	return blocks
}

// extractJiraLinks는 텍스트에서 Jira 이슈 키를 추출하고 링크를 생성합니다.
func (h *SlackNotifyHandler) extractJiraLinks(subject, body string) string {
	if h.jiraBaseURL == "" {
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

	// Slack 링크 생성 (<URL|텍스트> 형식)
	baseURL := strings.TrimSuffix(h.jiraBaseURL, "/")
	var links []string
	for _, key := range uniqueKeys {
		links = append(links, fmt.Sprintf("<%s/browse/%s|%s>", baseURL, key, key))
	}

	return strings.Join(links, ", ")
}

// escapeSlackText는 Slack 특수문자를 이스케이프합니다.
func escapeSlackText(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

// truncateTextForSlack는 텍스트를 지정된 길이로 자릅니다.
func truncateTextForSlack(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// isStatusChangeEmail은 Jira 상태 변경 이메일인지 확인합니다.
// 이메일 본문에 "상태 변경:" 패턴이 있으면 상태 변경 이메일로 판단합니다.
func (h *SlackNotifyHandler) isStatusChangeEmail(msg *domain.Message) bool {
	// 본문에서 상태 변경 패턴 확인
	if strings.Contains(msg.Text, "상태 변경:") {
		return true
	}
	return false
}
