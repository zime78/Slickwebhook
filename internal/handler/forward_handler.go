package handler

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/history"
)

// ForwardHandler는 이벤트를 ClickUp로 전송하고 히스토리를 관리하는 핸들러입니다.
type ForwardHandler struct {
	clickupClient clickup.Client
	historyStore  history.Store
	logger        *log.Logger
	enabled       bool
	filterBotOnly bool     // true면 봇 메시지만 전송
	allowedBotIDs []string // 허용된 봇 ID 목록 (비어있으면 모든 봇)
}

// ForwardHandlerConfig는 ForwardHandler 설정입니다.
type ForwardHandlerConfig struct {
	ClickUpClient clickup.Client
	HistoryStore  history.Store
	Logger        *log.Logger
	Enabled       bool
	FilterBotOnly bool     // true면 봇 메시지만 전송
	AllowedBotIDs []string // 허용된 봇 ID 목록 (비어있으면 모든 봇)
}

// NewForwardHandler는 새로운 ForwardHandler를 생성합니다.
func NewForwardHandler(config ForwardHandlerConfig) *ForwardHandler {
	return &ForwardHandler{
		clickupClient: config.ClickUpClient,
		historyStore:  config.HistoryStore,
		logger:        config.Logger,
		enabled:       config.Enabled,
		filterBotOnly: config.FilterBotOnly,
		allowedBotIDs: config.AllowedBotIDs,
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

	// 상태 변경 이메일은 필터링 (Jira 상태 변경 알림 제외)
	if h.isStatusChangeEmail(msg) {
		h.logger.Printf("[FORWARD] ⏭️ 상태 변경 이메일 스킵: %s\n", msg.Subject)
		return
	}

	h.logger.Printf("[FORWARD] 📤 ClickUp으로 전송 중... (BotID: %s)\n", msg.BotID)

	// ClickUp 태스크 생성 (30초 타임아웃)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := h.clickupClient.CreateTask(ctx, msg)

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

// isStatusChangeEmail은 Jira 상태 변경 이메일인지 확인합니다.
// 이메일 본문에 "상태 변경:" 패턴이 있으면 상태 변경 이메일로 판단합니다.
func (h *ForwardHandler) isStatusChangeEmail(msg *domain.Message) bool {
	// 이메일 소스가 아니면 상태 변경 필터링 불필요
	if msg.Source != "email" {
		return false
	}
	// 본문에서 상태 변경 패턴 확인
	if strings.Contains(msg.Text, "상태 변경:") {
		return true
	}
	return false
}
