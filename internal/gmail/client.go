package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
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
	ClientID             string
	ClientSecret         string
	RefreshToken         string
	UserEmail            string
	FilterFrom           []string
	FilterExclude        []string // 제외 필터 (발신자)
	FilterExcludeSubject []string // 제외 필터 (제목)
	FilterLabel          string
}

// Client는 Gmail IMAP API와 상호작용하는 인터페이스입니다.
type Client interface {
	GetNewMessages(ctx context.Context, since time.Time) ([]*domain.Message, error)
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

func (c *xoauth2Client) Start() (mech string, ir []byte, err error) {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.accessToken)
	return "XOAUTH2", []byte(authString), nil
}

func (c *xoauth2Client) Next(challenge []byte) (response []byte, err error) {
	return nil, nil
}

func newXoauth2Client(username, accessToken string) sasl.Client {
	return &xoauth2Client{
		username:    username,
		accessToken: accessToken,
	}
}

// ErrRefreshTokenExpired는 Refresh Token이 만료/취소되었을 때 반환되는 에러입니다.
var ErrRefreshTokenExpired = fmt.Errorf("refresh token expired or revoked")

// GetNewMessages는 지정된 시간 이후의 새 이메일을 조회합니다.
func (c *GmailClient) GetNewMessages(ctx context.Context, since time.Time) ([]*domain.Message, error) {
	tokenSource := c.oauthConfig.TokenSource(ctx, c.token)
	newToken, err := tokenSource.Token()
	if err != nil {
		// invalid_grant 에러 감지 (Refresh Token 만료/취소)
		errStr := err.Error()
		if strings.Contains(errStr, "invalid_grant") ||
			strings.Contains(errStr, "Token has been expired or revoked") {
			c.logger.Printf("\n" +
				"╔══════════════════════════════════════════════════════════════════╗\n" +
				"║  🚨 GMAIL_REFRESH_TOKEN 만료 또는 취소됨                          ║\n" +
				"╠══════════════════════════════════════════════════════════════════╣\n" +
				"║  원인:                                                            ║\n" +
				"║    - Google 계정 비밀번호 변경                                    ║\n" +
				"║    - Google 계정 권한 페이지에서 앱 연결 해제                     ║\n" +
				"║    - 6개월 이상 토큰 미사용                                       ║\n" +
				"║    - 새로운 Refresh Token 발급으로 인한 기존 토큰 무효화          ║\n" +
				"╠══════════════════════════════════════════════════════════════════╣\n" +
				"║  복구 방법:                                                       ║\n" +
				"║    1. OAuth 인증을 다시 진행하여 새 Refresh Token 발급            ║\n" +
				"║    2. config.email.ini 또는 환경변수 GMAIL_REFRESH_TOKEN 업데이트 ║\n" +
				"║    3. email-monitor 재시작                                        ║\n" +
				"╚══════════════════════════════════════════════════════════════════╝\n")
			return nil, fmt.Errorf("%w: %v", ErrRefreshTokenExpired, err)
		}
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}
	c.token = newToken

	imapClient, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer imapClient.Logout()

	saslClient := newXoauth2Client(c.config.UserEmail, newToken.AccessToken)
	if err := imapClient.Authenticate(saslClient); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	mbox, err := imapClient.Select(c.config.FilterLabel, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	uids, err := imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	if len(uids) == 0 {
		return nil, nil
	}

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

		// 발신자 정보
		fromAddr := ""
		fromName := ""
		if len(msg.Envelope.From) > 0 {
			fromAddr = msg.Envelope.From[0].Address()
			fromName = msg.Envelope.From[0].PersonalName
			if fromName == "" {
				fromName = fromAddr
			}
		}

		// FilterFrom이 설정되어 있으면 필터링
		if len(c.config.FilterFrom) > 0 && !c.matchesFromFilter(fromAddr) {
			continue
		}

		// FilterExclude가 설정되어 있으면 제외 필터링
		if len(c.config.FilterExclude) > 0 && c.matchesExcludeFilter(fromAddr) {
			continue
		}

		// FilterExcludeSubject가 설정되어 있으면 제목 기반 제외 필터링
		if len(c.config.FilterExcludeSubject) > 0 && c.matchesExcludeSubjectFilter(msg.Envelope.Subject) {
			continue
		}

		// 본문 추출 (MIME 파싱)
		body := ""
		for _, literal := range msg.Body {
			if literal != nil {
				bodyBytes, err := io.ReadAll(literal)
				if err == nil {
					body = extractTextFromMIME(bodyBytes)
				}
			}
		}

		// 본문 정리 및 제한
		body = cleanEmailBody(body)
		if len(body) > 2000 {
			body = body[:2000] + "..."
		}

		messages = append(messages, &domain.Message{
			Source:    "email",
			Timestamp: fmt.Sprintf("%d", msg.Uid),
			MessageID: msg.Envelope.MessageId,
			Subject:   msg.Envelope.Subject,
			From:      fromName,
			Text:      body,
			CreatedAt: msg.Envelope.Date,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return messages, nil
}

// extractTextFromMIME는 MIME 메시지에서 텍스트 본문을 추출합니다.
func extractTextFromMIME(raw []byte) string {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		// 파싱 실패 시 원본에서 헤더 제거 시도
		return removeHeaders(string(raw))
	}

	contentType := msg.Header.Get("Content-Type")
	encoding := msg.Header.Get("Content-Transfer-Encoding")

	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return ""
	}

	// multipart 처리
	if strings.Contains(contentType, "multipart") {
		return extractFromMultipart(contentType, body)
	}

	// 단일 파트
	decoded := decodeBody(body, encoding)

	// HTML인 경우 텍스트 추출
	if strings.Contains(contentType, "text/html") {
		return stripHTML(decoded)
	}

	return decoded
}

// extractFromMultipart는 multipart 메시지에서 텍스트를 추출합니다.
func extractFromMultipart(contentType string, body []byte) string {
	// boundary 추출
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return cleanRawContent(string(body))
	}

	boundary := params["boundary"]
	if boundary == "" {
		return cleanRawContent(string(body))
	}

	// mime/multipart 표준 라이브러리 사용
	reader := multipart.NewReader(bytes.NewReader(body), boundary)

	var textPart, htmlPart string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 파싱 실패 시 수동 파싱 시도
			return extractFromMultipartManual(boundary, body)
		}

		partContentType := part.Header.Get("Content-Type")
		partEncoding := part.Header.Get("Content-Transfer-Encoding")

		partBody, err := io.ReadAll(part)
		if err != nil {
			continue
		}

		// 중첩된 multipart 처리
		if strings.Contains(partContentType, "multipart") {
			nested := extractFromMultipart(partContentType, partBody)
			if nested != "" {
				return nested
			}
			continue
		}

		decoded := decodeBody(partBody, partEncoding)

		if strings.Contains(strings.ToLower(partContentType), "text/plain") {
			textPart = decoded
		} else if strings.Contains(strings.ToLower(partContentType), "text/html") {
			htmlPart = decoded
		}
	}

	// text/plain 우선, 없으면 html에서 추출
	if textPart != "" {
		return textPart
	}
	if htmlPart != "" {
		return stripHTML(htmlPart)
	}

	return ""
}

// extractFromMultipartManual은 수동으로 multipart를 파싱합니다 (fallback).
func extractFromMultipartManual(boundary string, body []byte) string {
	parts := bytes.Split(body, []byte("--"+boundary))

	var textPart, htmlPart string

	for _, part := range parts {
		partStr := string(part)
		lowerPart := strings.ToLower(partStr)

		// 종료 마커 무시
		if strings.HasPrefix(strings.TrimSpace(partStr), "--") {
			continue
		}

		// Content-Transfer-Encoding 확인
		encoding := ""
		if strings.Contains(lowerPart, "content-transfer-encoding: base64") {
			encoding = "base64"
		} else if strings.Contains(lowerPart, "content-transfer-encoding: quoted-printable") {
			encoding = "quoted-printable"
		}

		// 파트 헤더 파싱
		if strings.Contains(lowerPart, "content-type: text/plain") {
			textPart = extractPartBody(partStr, encoding)
		} else if strings.Contains(lowerPart, "content-type: text/html") {
			htmlPart = extractPartBody(partStr, encoding)
		}
	}

	// text/plain 우선, 없으면 html에서 추출
	if textPart != "" {
		return textPart
	}
	if htmlPart != "" {
		return stripHTML(htmlPart)
	}

	return ""
}

// cleanRawContent는 MIME 경계 마커를 제거합니다.
func cleanRawContent(content string) string {
	// MIME 경계 마커 제거 패턴
	lines := strings.Split(content, "\n")
	var cleaned []string
	skipUntilEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// MIME 경계 라인 및 헤더 스킵
		if strings.HasPrefix(trimmed, "--") && (strings.Contains(trimmed, "enmime") || len(trimmed) > 30) {
			skipUntilEmpty = true
			continue
		}

		// Content-* 헤더 스킵
		if strings.HasPrefix(strings.ToLower(trimmed), "content-") {
			skipUntilEmpty = true
			continue
		}

		// 빈 줄 후 내용 시작
		if skipUntilEmpty && trimmed == "" {
			skipUntilEmpty = false
			continue
		}

		if !skipUntilEmpty && trimmed != "" {
			// Base64로 보이는 줄 스킵
			if isBase64Line(trimmed) {
				continue
			}
			cleaned = append(cleaned, line)
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// isBase64Line은 해당 줄이 Base64 인코딩된 데이터인지 확인합니다.
// 보수적 조건: 60자 이상 + (표준 MIME Base64 라인 길이 76자 또는 패딩 '=' 포함)
func isBase64Line(line string) bool {
	lineLen := len(line)

	// 60자 미만은 Base64로 간주하지 않음
	if lineLen < 60 {
		return false
	}

	// 표준 MIME Base64 라인은 76자 또는 끝에 = 패딩이 있음
	hasStandardLength := lineLen == 76
	hasPadding := strings.HasSuffix(line, "=")

	if !hasStandardLength && !hasPadding {
		return false
	}

	// Base64 문자셋 검증: A-Z, a-z, 0-9, +, /, =
	for _, r := range line {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=') {
			return false
		}
	}
	return true
}

// extractPartBody는 MIME 파트에서 본문만 추출합니다.
func extractPartBody(part string, encoding string) string {
	// 빈 줄로 헤더와 본문 분리
	idx := strings.Index(part, "\r\n\r\n")
	if idx == -1 {
		idx = strings.Index(part, "\n\n")
	}
	if idx == -1 {
		return ""
	}

	bodyPart := part[idx:]
	bodyPart = strings.TrimSpace(bodyPart)

	return decodeBody([]byte(bodyPart), encoding)
}

// decodeBody는 인코딩된 본문을 디코딩합니다.
func decodeBody(body []byte, encoding string) string {
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err == nil {
			return string(decoded)
		}
	case "base64":
		// base64 줄바꿈 제거
		cleaned := strings.ReplaceAll(string(body), "\r\n", "")
		cleaned = strings.ReplaceAll(cleaned, "\n", "")
		cleaned = strings.TrimSpace(cleaned)

		// base64 디코딩
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err == nil {
			return string(decoded)
		}
		// URL-safe base64 시도
		decoded, err = base64.URLEncoding.DecodeString(cleaned)
		if err == nil {
			return string(decoded)
		}
		// Raw base64 시도 (패딩 없음)
		decoded, err = base64.RawStdEncoding.DecodeString(cleaned)
		if err == nil {
			return string(decoded)
		}
	}
	return string(body)
}

// stripHTML는 HTML 태그를 제거합니다.
func stripHTML(html string) string {
	// 스크립트/스타일 제거
	reScript := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	html = reScript.ReplaceAllString(html, "")

	reStyle := regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// 줄바꿈 태그를 실제 줄바꿈으로
	reBr := regexp.MustCompile(`<br\s*/?>`)
	html = reBr.ReplaceAllString(html, "\n")

	reP := regexp.MustCompile(`</p>`)
	html = reP.ReplaceAllString(html, "\n\n")

	// 모든 태그 제거
	reTag := regexp.MustCompile(`<[^>]*>`)
	html = reTag.ReplaceAllString(html, "")

	// HTML 엔티티 디코딩
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")

	return html
}

// removeHeaders는 이메일 헤더를 제거합니다.
func removeHeaders(content string) string {
	idx := strings.Index(content, "\r\n\r\n")
	if idx == -1 {
		idx = strings.Index(content, "\n\n")
	}
	if idx != -1 {
		return strings.TrimSpace(content[idx:])
	}
	return content
}

// cleanEmailBody는 이메일 본문을 정리합니다.
func cleanEmailBody(body string) string {
	// 연속 공백 줄이기
	reSpaces := regexp.MustCompile(`[ \t]+`)
	body = reSpaces.ReplaceAllString(body, " ")

	// 연속 줄바꿈 줄이기
	reNewlines := regexp.MustCompile(`\n{3,}`)
	body = reNewlines.ReplaceAllString(body, "\n\n")

	return strings.TrimSpace(body)
}

func (c *GmailClient) matchesFromFilter(from string) bool {
	from = strings.ToLower(from)
	for _, filter := range c.config.FilterFrom {
		if strings.Contains(from, strings.ToLower(filter)) {
			return true
		}
	}
	return false
}

// matchesExcludeFilter는 발신자가 제외 필터에 매칭되는지 확인합니다.
func (c *GmailClient) matchesExcludeFilter(from string) bool {
	from = strings.ToLower(from)
	for _, filter := range c.config.FilterExclude {
		if strings.Contains(from, strings.ToLower(filter)) {
			return true
		}
	}
	return false
}

// matchesExcludeSubjectFilter는 제목이 제외 필터에 매칭되는지 확인합니다.
func (c *GmailClient) matchesExcludeSubjectFilter(subject string) bool {
	subject = strings.ToLower(subject)
	for _, filter := range c.config.FilterExcludeSubject {
		if strings.Contains(subject, strings.ToLower(filter)) {
			return true
		}
	}
	return false
}

func (c *GmailClient) Close() error {
	return nil
}
