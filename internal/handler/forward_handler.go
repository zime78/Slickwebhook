package handler

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/history"
	"github.com/zime/slickwebhook/internal/jira"
	"github.com/zime/slickwebhook/internal/store"
)

// ForwardJiraClient는 Jira API 클라이언트 인터페이스입니다.
type ForwardJiraClient interface {
	GetIssueSummary(ctx context.Context, issueKey string) (string, error)
	GetIssueDetail(ctx context.Context, issueKey string) (*jira.IssueDetail, error)
	DownloadAttachment(ctx context.Context, contentURL string) ([]byte, error)
}

// ForwardHandler는 이벤트를 ClickUp로 전송하고 히스토리를 관리하는 핸들러입니다.
type ForwardHandler struct {
	clickupClient  clickup.Client
	historyStore   history.Store
	logger         *log.Logger
	enabled        bool
	filterBotOnly  bool                 // true면 봇 메시지만 전송
	allowedBotIDs  []string             // 허용된 봇 ID 목록 (비어있으면 모든 봇)
	jiraClient     ForwardJiraClient    // Jira API 클라이언트 (이슈 타이틀 조회용)
	jiraIssueStore store.JiraIssueStore // Jira 이슈 중복 체크 저장소
}

// ForwardHandlerConfig는 ForwardHandler 설정입니다.
type ForwardHandlerConfig struct {
	ClickUpClient  clickup.Client
	HistoryStore   history.Store
	Logger         *log.Logger
	Enabled        bool
	FilterBotOnly  bool                 // true면 봇 메시지만 전송
	AllowedBotIDs  []string             // 허용된 봇 ID 목록 (비어있으면 모든 봇)
	JiraClient     ForwardJiraClient    // Jira API 클라이언트 (이슈 타이틀 조회용)
	JiraIssueStore store.JiraIssueStore // Jira 이슈 중복 체크 저장소
}

// NewForwardHandler는 새로운 ForwardHandler를 생성합니다.
func NewForwardHandler(config ForwardHandlerConfig) *ForwardHandler {
	return &ForwardHandler{
		clickupClient:  config.ClickUpClient,
		historyStore:   config.HistoryStore,
		logger:         config.Logger,
		enabled:        config.Enabled,
		filterBotOnly:  config.FilterBotOnly,
		allowedBotIDs:  config.AllowedBotIDs,
		jiraClient:     config.JiraClient,
		jiraIssueStore: config.JiraIssueStore,
	}
}

// Handle은 이벤트를 ClickUp으로 전송합니다.
func (h *ForwardHandler) Handle(event *domain.Event) {
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

	// 봇 필터링
	if h.filterBotOnly {
		if msg.BotID == "" {
			h.logger.Println("[FORWARD] ⏭️ 사용자 메시지 스킵 (봇 메시지만 처리)")
			return
		}

		// 허용된 봇 ID 목록이 있으면 체크
		if len(h.allowedBotIDs) > 0 {
			allowed := false
			for _, id := range h.allowedBotIDs {
				if msg.BotID == id {
					allowed = true
					break
				}
			}
			if !allowed {
				h.logger.Printf("[FORWARD] ⏭️ 허용되지 않은 봇 메시지 스킵 (BotID: %s)\n", msg.BotID)
				return
			}
		}
	}

	// 노이즈 이메일 필터링 (Jira 상태 변경, 담당자 변경 알림 제외)
	if h.isFilteredEmail(msg) {
		h.logger.Printf("[FORWARD] ⏭️ 필터링된 이메일 스킵: %s\n", msg.Subject)
		return
	}

	// Jira 이슈 중복 체크 (동일 이슈는 한 번만 전송)
	issueKey := h.extractJiraIssueKey(msg.Subject)
	if issueKey != "" && h.jiraIssueStore != nil {
		processed, err := h.jiraIssueStore.IsProcessed(issueKey)
		if err != nil {
			h.logger.Printf("[FORWARD] ⚠️ Jira 이슈 중복 체크 실패: %v\n", err)
		} else if processed {
			h.logger.Printf("[FORWARD] ⏭️ Jira 이슈 중복 스킵 (이미 처리됨): %s\n", issueKey)
			return
		}
	}

	h.logger.Printf("[FORWARD] 📤 ClickUp으로 전송 중... (BotID: %s)\n", msg.BotID)

	// Jira 이메일인 경우 제목을 이슈키 + 이슈타이틀 형식으로 변환 + 본문 재구성
	processedMsg := msg
	var imageAttachments []jira.Attachment
	if msg.Source == "email" && strings.Contains(msg.Subject, "[Jira]") && issueKey != "" {
		newSubject := h.formatJiraSubjectForClickUp(msg.Subject)

		// Jira 이슈 상세 정보 조회 (본문, 첨부파일)
		if h.jiraClient != nil {
			detailCtx, detailCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer detailCancel()

			if detail, err := h.jiraClient.GetIssueDetail(detailCtx, issueKey); err == nil {
				h.logger.Printf("[FORWARD] ✅ Jira 이슈 상세 조회 성공\n")

				// 이미지 첨부파일 필터링
				imageAttachments = jira.FilterImageAttachments(detail.Attachments)
				h.logger.Printf("[FORWARD] 📷 첨부 이미지: %d개\n", len(imageAttachments))

				// 본문 재구성 ([현 결과] → [오류내용], [기대 결과] → [수정요청])
				urls := make([]string, len(imageAttachments))
				for i, img := range imageAttachments {
					urls[i] = img.Content
				}
				reformattedDesc := jira.ReformatDescription(detail.Description, urls)

				// 메시지 복사본 생성 (원본 수정 방지)
				msgCopy := *msg
				msgCopy.Subject = newSubject
				msgCopy.Text = reformattedDesc
				processedMsg = &msgCopy
				h.logger.Printf("[FORWARD] 🔄 Jira 본문 재구성 완료\n")
			} else {
				h.logger.Printf("[FORWARD] ⚠️ Jira 이슈 상세 조회 실패: %v\n", err)
				// 상세 조회 실패해도 제목 변환은 진행
				if newSubject != msg.Subject {
					msgCopy := *msg
					msgCopy.Subject = newSubject
					processedMsg = &msgCopy
				}
			}
		} else if newSubject != msg.Subject {
			msgCopy := *msg
			msgCopy.Subject = newSubject
			processedMsg = &msgCopy
		}

		if processedMsg.Subject != msg.Subject {
			h.logger.Printf("[FORWARD] 🔄 Jira 제목 변환: %s\n", processedMsg.Subject)
		}
	}

	// ClickUp 태스크 생성 (30초 타임아웃)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := h.clickupClient.CreateTask(ctx, processedMsg)

	// 히스토리 레코드 생성
	record := &history.Record{
		SlackTimestamp: msg.Timestamp,
		MessageText:    truncateText(msg.Text, 50),
	}

	if err != nil {
		record.Success = false
		record.ErrorMessage = err.Error()
		h.logger.Printf("[FORWARD] ❌ 전송 실패: %v\n", err)
	} else {
		record.Success = true
		record.ClickUpTaskID = resp.ID
		record.ClickUpTaskURL = resp.URL
		h.logger.Printf("[FORWARD] ✅ 전송 성공!\n")
		h.logger.Printf("  - Task ID: %s\n", resp.ID)
		h.logger.Printf("  - Task URL: %s\n", resp.URL)

		// Jira 첨부 이미지 ClickUp에 업로드
		if len(imageAttachments) > 0 && h.jiraClient != nil {
			h.uploadJiraAttachments(ctx, resp.ID, imageAttachments)
		}

		// Jira 이슈 처리 완료 마킹 (중복 전송 방지)
		if issueKey != "" && h.jiraIssueStore != nil {
			if markErr := h.jiraIssueStore.MarkProcessed(issueKey, processedMsg.Subject); markErr != nil {
				h.logger.Printf("[FORWARD] ⚠️ Jira 이슈 마킹 실패: %v\n", markErr)
			} else {
				h.logger.Printf("[FORWARD] 📝 Jira 이슈 처리 완료 마킹: %s\n", issueKey)
			}
		}
	}

	// 히스토리 저장
	h.historyStore.Add(record)
	h.logger.Printf("[HISTORY] 📋 히스토리 저장 (총 %d개)\n", h.historyStore.Count())
}

// ChainHandler는 여러 핸들러를 체이닝하는 핸들러입니다.
type ChainHandler struct {
	handlers []EventHandler
}

// NewChainHandler는 새로운 ChainHandler를 생성합니다.
func NewChainHandler(handlers ...EventHandler) *ChainHandler {
	return &ChainHandler{
		handlers: handlers,
	}
}

// Handle은 모든 핸들러를 순차적으로 호출합니다.
func (h *ChainHandler) Handle(event *domain.Event) {
	for _, handler := range h.handlers {
		handler.Handle(event)
	}
}

// isFilteredEmail은 필터링 대상 Jira 알림 이메일인지 확인합니다.
// 이메일 본문에 필터링 대상 패턴이 있으면 필터링합니다.
func (h *ForwardHandler) isFilteredEmail(msg *domain.Message) bool {
	// 이메일 소스가 아니면 필터링 불필요
	if msg.Source != "email" {
		return false
	}

	// 필터링 대상 패턴 목록
	filterPatterns := []string{
		"상태 변경",
		"담당자 변경",
	}

	for _, pattern := range filterPatterns {
		if strings.Contains(msg.Text, pattern) {
			return true
		}
	}
	return false
}

// formatJiraSubjectForClickUp은 Jira API를 사용하여 이슈 타이틀을 조회하고 제목을 변환합니다.
func (h *ForwardHandler) formatJiraSubjectForClickUp(subject string) string {
	// 이슈 키 추출
	issueKeyPattern := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)
	issueKey := issueKeyPattern.FindString(subject)

	if issueKey == "" {
		return subject
	}

	// Jira 클라이언트가 없으면 원래 제목 반환
	if h.jiraClient == nil {
		h.logger.Printf("[FORWARD] ⚠️ Jira 클라이언트가 설정되지 않음, 원래 제목 사용\n")
		return subject
	}

	// Jira API로 이슈 타이틀 조회
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	issueTitle, err := h.jiraClient.GetIssueSummary(ctx, issueKey)
	if err != nil {
		h.logger.Printf("[FORWARD] ⚠️ Jira 이슈 조회 실패 (%s): %v\n", issueKey, err)
		return subject
	}

	// "ITSM-5052 [Q-글로벌][iOS] 회원가입 > ..." 형식으로 반환
	h.logger.Printf("[FORWARD] ✅ Jira 이슈 타이틀 조회 성공: %s\n", issueTitle)
	return issueKey + " " + issueTitle
}

// extractJiraIssueKey는 텍스트에서 Jira 이슈 키를 추출합니다.
func (h *ForwardHandler) extractJiraIssueKey(text string) string {
	issueKeyPattern := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)
	return issueKeyPattern.FindString(text)
}

// uploadJiraAttachments는 Jira 첨부파일을 ClickUp에 업로드합니다.
func (h *ForwardHandler) uploadJiraAttachments(ctx context.Context, taskID string, attachments []jira.Attachment) {
	for _, att := range attachments {
		h.logger.Printf("[FORWARD] 📤 이미지 업로드 중: %s\n", att.Filename)

		// Jira에서 이미지 다운로드
		data, err := h.jiraClient.DownloadAttachment(ctx, att.Content)
		if err != nil {
			h.logger.Printf("[FORWARD] ⚠️ 이미지 다운로드 실패 (%s): %v\n", att.Filename, err)
			continue
		}

		// ClickUp에 업로드
		if err := h.clickupClient.UploadAttachment(ctx, taskID, att.Filename, data); err != nil {
			h.logger.Printf("[FORWARD] ⚠️ 이미지 업로드 실패 (%s): %v\n", att.Filename, err)
		} else {
			h.logger.Printf("[FORWARD] ✅ 이미지 업로드 성공: %s\n", att.Filename)
		}
	}
}
