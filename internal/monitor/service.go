package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/handler"
	"github.com/zime/slickwebhook/internal/slack"
)

// Config는 모니터 서비스 설정입니다.
type Config struct {
	// ChannelID는 모니터링할 Slack 채널 ID입니다
	ChannelID string
	// PollInterval은 폴링 간격입니다 (기본값: 10초)
	PollInterval time.Duration
}

// DefaultPollInterval은 기본 폴링 간격입니다.
const DefaultPollInterval = 10 * time.Second

// Service는 Slack 채널 모니터링 서비스입니다.
type Service struct {
	config        Config
	client        slack.Client
	handler       handler.EventHandler
	logger        *log.Logger
	lastTimestamp string
	mu            sync.Mutex
	stopChan      chan struct{}
	running       bool
}

// NewService는 새로운 모니터 서비스를 생성합니다.
func NewService(config Config, client slack.Client, eventHandler handler.EventHandler, logger *log.Logger) *Service {
	if config.PollInterval == 0 {
		config.PollInterval = DefaultPollInterval
	}

	return &Service{
		config:   config,
		client:   client,
		handler:  eventHandler,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start는 모니터링을 시작합니다.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	s.logger.Printf("[INFO] 🚀 Slack 채널 모니터 시작 (채널: %s, 간격: %v)\n", s.config.ChannelID, s.config.PollInterval)

	// 초기 타임스탬프를 현재 시간으로 설정 (과거 메시지 무시)
	s.lastTimestamp = getCurrentTimestamp()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("[INFO] 🛑 컨텍스트 취소로 모니터링 종료")
			return ctx.Err()
		case <-s.stopChan:
			s.logger.Println("[INFO] 🛑 Stop 호출로 모니터링 종료")
			return nil
		case <-ticker.C:
			s.checkForNewMessages(ctx)
		}
	}
}

// Stop은 모니터링을 중지합니다.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
}

// IsRunning은 서비스가 실행 중인지 확인합니다.
func (s *Service) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// checkForNewMessages는 새 메시지를 확인합니다.
func (s *Service) checkForNewMessages(ctx context.Context) {
	s.mu.Lock()
	oldest := s.lastTimestamp
	s.mu.Unlock()

	messages, err := s.client.GetChannelHistory(ctx, s.config.ChannelID, oldest)
	if err != nil {
		event := domain.NewErrorEvent(err)
		s.handler.Handle(event)
		return
	}

	if len(messages) == 0 {
		s.logger.Println("[INFO] ✅ 체크 완료 - 새 메시지 없음")
		return
	}

	s.logger.Printf("[INFO] 📬 %d개의 새 메시지 발견\n", len(messages))

	// 메시지를 오래된 순서로 처리 (역순)
	var lastTs string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		event := domain.NewMessageEvent(msg)
		s.handler.Handle(event)

		// 가장 최신 타임스탬프 추적
		if msg.Timestamp > lastTs {
			lastTs = msg.Timestamp
		}
	}

	// 마지막 타임스탬프 업데이트 (루프 외부에서 한번만 Lock)
	if lastTs != "" {
		s.mu.Lock()
		if lastTs > s.lastTimestamp {
			s.lastTimestamp = lastTs
		}
		s.mu.Unlock()
	}
}

// getCurrentTimestamp는 현재 시간을 Slack 타임스탬프 형식으로 반환합니다.
func getCurrentTimestamp() string {
	return formatSlackTimestamp(time.Now())
}

// formatSlackTimestamp는 time.Time을 Slack 타임스탬프 형식으로 변환합니다.
// Slack 타임스탬프는 "1234567890.123456" 형식 (Unix 초.마이크로초)
func formatSlackTimestamp(t time.Time) string {
	return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
}
