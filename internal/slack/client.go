package slack

import (
	"context"
	"time"

	"github.com/slack-go/slack"
	"github.com/zime/slickwebhook/internal/domain"
)

// Client는 Slack API와 상호작용하는 인터페이스입니다.
// 테스트 시 모킹이 가능하도록 인터페이스로 정의합니다.
type Client interface {
	// GetChannelHistory는 채널의 메시지 히스토리를 조회합니다.
	// oldest 이후의 메시지만 반환합니다.
	GetChannelHistory(ctx context.Context, channelID string, oldest string) ([]*domain.Message, error)

	// PostMessage는 채널에 메시지를 전송합니다.
	// blocks를 사용하여 Block Kit 형식의 메시지를 보낼 수 있습니다.
	PostMessage(ctx context.Context, channelID string, blocks []slack.Block, text string) error

	// UploadFile은 채널에 파일을 업로드합니다.
	UploadFile(ctx context.Context, channelID string, filename string, data []byte, comment string) error
}

// SlackClient는 실제 Slack API 클라이언트를 래핑합니다.
type SlackClient struct {
	api *slack.Client
}

// NewSlackClient는 새로운 SlackClient를 생성합니다.
func NewSlackClient(token string) *SlackClient {
	return &SlackClient{
		api: slack.New(token),
	}
}

// GetChannelHistory는 채널의 메시지 히스토리를 조회합니다.
func (c *SlackClient) GetChannelHistory(ctx context.Context, channelID string, oldest string) ([]*domain.Message, error) {
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Oldest:    oldest,
		Limit:     100,
	}

	resp, err := c.api.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, err
	}

	messages := make([]*domain.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// 시스템 메시지는 제외하되, 봇 메시지와 일반 메시지는 포함
		// channel_join, channel_leave 등 시스템 이벤트만 필터링
		if msg.SubType == "channel_join" || msg.SubType == "channel_leave" ||
			msg.SubType == "channel_topic" || msg.SubType == "channel_purpose" {
			continue
		}

		// 타임스탬프를 시간으로 변환
		createdAt := parseSlackTimestamp(msg.Timestamp)

		messages = append(messages, &domain.Message{
			Source:    "slack",
			Timestamp: msg.Timestamp,
			UserID:    msg.User,
			BotID:     msg.BotID,
			Text:      msg.Text,
			ChannelID: channelID,
			CreatedAt: createdAt,
		})
	}

	return messages, nil
}

// parseSlackTimestamp는 Slack 타임스탬프를 time.Time으로 변환합니다.
// Slack 타임스탬프는 "1234567890.123456" 형식입니다.
func parseSlackTimestamp(ts string) time.Time {
	// 타임스탬프에서 초 부분만 추출 (소수점 앞부분)
	var seconds int64
	for i, c := range ts {
		if c == '.' {
			break
		}
		seconds = seconds*10 + int64(c-'0')
		if i > 10 { // 오버플로우 방지
			break
		}
	}
	return time.Unix(seconds, 0)
}

// PostMessage는 채널에 Block Kit 형식의 메시지를 전송합니다.
// text는 Block을 지원하지 않는 클라이언트를 위한 폴백 텍스트입니다.
func (c *SlackClient) PostMessage(ctx context.Context, channelID string, blocks []slack.Block, text string) error {
	options := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}

	if len(blocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(blocks...))
	}

	_, _, err := c.api.PostMessageContext(ctx, channelID, options...)
	return err
}

// UploadFile은 채널에 파일을 업로드합니다.
func (c *SlackClient) UploadFile(ctx context.Context, channelID string, filename string, data []byte, comment string) error {
	_, err := c.api.UploadFileContext(ctx, slack.FileUploadParameters{
		Channels:       []string{channelID},
		Filename:       filename,
		Content:        string(data),
		InitialComment: comment,
	})
	return err
}
