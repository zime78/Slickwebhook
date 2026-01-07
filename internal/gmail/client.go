package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"github.com/zime/slickwebhook/internal/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config는 Gmail 클라이언트 설정입니다.
type Config struct {
	// ClientID는 Google OAuth Client ID입니다
	ClientID string
	// ClientSecret은 Google OAuth Client Secret입니다
	ClientSecret string
	// RefreshToken은 OAuth Refresh Token입니다
	RefreshToken string
	// UserEmail은 모니터링할 Gmail 주소입니다
	UserEmail string
	// FilterFrom은 필터링할 발신자 목록입니다 (콤마 구분)
	FilterFrom []string
	// FilterLabel은 모니터링할 라벨입니다 (기본: INBOX)
	FilterLabel string
}

// Client는 Gmail IMAP API와 상호작용하는 인터페이스입니다.
type Client interface {
	// GetNewMessages는 지정된 시간 이후의 새 이메일을 조회합니다.
	GetNewMessages(ctx context.Context, since time.Time) ([]*domain.Message, error)
	// Close는 IMAP 연결을 종료합니다.
	Close() error
}

// GmailClient는 Gmail IMAP 클라이언트를 래핑합니다.
type GmailClient struct {
	config      Config
	oauthConfig *oauth2.Config
	token       *oauth2.Token
	logger      *log.Logger
}

// NewGmailClient는 새로운 GmailClient를 생성합니다.
func NewGmailClient(cfg Config, logger *log.Logger) *GmailClient {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://mail.google.com/"},
	}

	// FilterLabel 기본값
	if cfg.FilterLabel == "" {
		cfg.FilterLabel = "INBOX"
	}

	return &GmailClient{
		config:      cfg,
		oauthConfig: oauthConfig,
		token: &oauth2.Token{
			RefreshToken: cfg.RefreshToken,
		},
		logger: logger,
	}
}

// xoauth2Client는 XOAUTH2 SASL 클라이언트입니다.
type xoauth2Client struct {
	username    string
	accessToken string
}

// Start는 XOAUTH2 인증을 시작합니다.
func (c *xoauth2Client) Start() (mech string, ir []byte, err error) {
	// XOAUTH2 형식: base64("user=" + username + "\x01auth=Bearer " + token + "\x01\x01")
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.accessToken)
	return "XOAUTH2", []byte(authString), nil
}

// Next는 서버 응답을 처리합니다.
func (c *xoauth2Client) Next(challenge []byte) (response []byte, err error) {
	// XOAUTH2는 단일 라운드이므로 추가 응답 불필요
	return nil, nil
}

// newXoauth2Client는 새로운 XOAUTH2 클라이언트를 생성합니다.
func newXoauth2Client(username, accessToken string) sasl.Client {
	return &xoauth2Client{
		username:    username,
		accessToken: accessToken,
	}
}

// GetNewMessages는 지정된 시간 이후의 새 이메일을 조회합니다.
func (c *GmailClient) GetNewMessages(ctx context.Context, since time.Time) ([]*domain.Message, error) {
	// Access Token 갱신
	tokenSource := c.oauthConfig.TokenSource(ctx, c.token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}
	c.token = newToken

	// IMAP 연결
	imapClient, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer imapClient.Logout()

	// XOAUTH2 인증
	saslClient := newXoauth2Client(c.config.UserEmail, newToken.AccessToken)
	if err := imapClient.Authenticate(saslClient); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// 메일박스 선택
	mbox, err := imapClient.Select(c.config.FilterLabel, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	// 날짜 기반 검색
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	// 발신자 필터가 있으면 첫 번째 발신자로 검색 (IMAP 제한)
	// 여러 발신자는 결과에서 필터링
	uids, err := imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	if len(uids) == 0 {
		return nil, nil
	}

	// 메시지 가져오기
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}

	msgChan := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- imapClient.Fetch(seqset, items, msgChan)
	}()

	var messages []*domain.Message
	for msg := range msgChan {
		if msg == nil || msg.Envelope == nil {
			continue
		}

		// 발신자 필터링
		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		// FilterFrom이 설정되어 있으면 필터링
		if len(c.config.FilterFrom) > 0 && !c.matchesFromFilter(from) {
			continue
		}

		// 본문 추출
		body := ""
		for _, literal := range msg.Body {
			if literal != nil {
				bodyBytes, err := io.ReadAll(literal)
				if err == nil {
					body = string(bodyBytes)
					// 본문이 너무 길면 자르기
					if len(body) > 5000 {
						body = body[:5000] + "..."
					}
				}
			}
		}

		messages = append(messages, &domain.Message{
			Source:    "email",
			Timestamp: fmt.Sprintf("%d", msg.Uid),
			MessageID: msg.Envelope.MessageId,
			Subject:   msg.Envelope.Subject,
			From:      from,
			Text:      body,
			CreatedAt: msg.Envelope.Date,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return messages, nil
}

// matchesFromFilter는 발신자가 필터 목록에 포함되는지 확인합니다.
func (c *GmailClient) matchesFromFilter(from string) bool {
	from = strings.ToLower(from)
	for _, filter := range c.config.FilterFrom {
		if strings.Contains(from, strings.ToLower(filter)) {
			return true
		}
	}
	return false
}

// Close는 클라이언트를 정리합니다.
func (c *GmailClient) Close() error {
	// 현재 구현에서는 각 요청마다 연결을 생성하므로 특별히 정리할 것이 없음
	return nil
}

// unused import 방지용 (base64는 참조용으로 남김)
var _ = base64.StdEncoding
