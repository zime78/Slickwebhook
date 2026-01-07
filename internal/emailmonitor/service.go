package emailmonitor

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/gmail"
	"github.com/zime/slickwebhook/internal/handler"
)

// Config는 Email 모니터 서비스 설정입니다.
type Config struct {
	// PollInterval은 폴링 간격입니다 (기본값: 30초)
	PollInterval time.Duration
}

// DefaultPollInterval은 기본 폴링 간격입니다.
const DefaultPollInterval = 30 * time.Second

// Service는 Email 모니터링 서비스입니다.
type Service struct {
	config   Config
	client   gmail.Client
	handler  handler.EventHandler
	logger   *log.Logger
	lastTime time.Time
	mu       sync.Mutex
	stopChan chan struct{}
	running  bool
}

// NewService는 새로운 Email 모니터 서비스를 생성합니다.
func NewService(config Config, client gmail.Client, eventHandler handler.EventHandler, logger *log.Logger) *Service {
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

	s.logger.Printf("[INFO] 📧 Email 모니터 시작 (간격: %v)\n", s.config.PollInterval)

	// 초기 시간을 현재로 설정 (과거 이메일 무시)
	s.lastTime = time.Now()

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
			s.checkForNewEmails(ctx)
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

// checkForNewEmails는 새 이메일을 확인합니다.
func (s *Service) checkForNewEmails(ctx context.Context) {
	s.mu.Lock()
	since := s.lastTime
	s.mu.Unlock()

	messages, err := s.client.GetNewMessages(ctx, since)
	if err != nil {
		event := domain.NewErrorEvent(err)
		s.handler.Handle(event)
		return
	}

	if len(messages) == 0 {
		s.logger.Println("[INFO] ✅ 체크 완료 - 새 이메일 없음")
		return
	}

	s.logger.Printf("[INFO] 📬 %d개의 새 이메일 발견\n", len(messages))

	// 메시지를 처리
	var latestTime time.Time
	for _, msg := range messages {
		event := domain.NewMessageEvent(msg)
		s.handler.Handle(event)

		// 가장 최신 시간 추적
		if msg.CreatedAt.After(latestTime) {
			latestTime = msg.CreatedAt
		}
	}

	// 마지막 시간 업데이트
	if !latestTime.IsZero() {
		s.mu.Lock()
		if latestTime.After(s.lastTime) {
			s.lastTime = latestTime
		}
		s.mu.Unlock()
	}
}
