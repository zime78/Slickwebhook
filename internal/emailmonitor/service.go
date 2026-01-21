package emailmonitor

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/zime/slickwebhook/internal/domain"
	"github.com/zime/slickwebhook/internal/gmail"
	"github.com/zime/slickwebhook/internal/handler"
	"github.com/zime/slickwebhook/internal/store"
)

// Config는 Email 모니터 서비스 설정입니다.
type Config struct {
	PollInterval     time.Duration
	LookbackDuration time.Duration
	RetentionDays    int // DB 보관 기간 (기본: 90일)
}

const DefaultPollInterval = 30 * time.Second
const DefaultRetentionDays = 90
const Version = "1.1.0" // Jira 본문 재구성 + 이미지 업로드 기능 추가

// Service는 Email 모니터링 서비스입니다.
type Service struct {
	config         Config
	client         gmail.Client
	handler        handler.EventHandler
	processedStore store.ProcessedStore
	logger         *log.Logger
	lastTime       time.Time
	mu             sync.Mutex
	stopChan       chan struct{}
	running        bool
}

// NewService는 새로운 Email 모니터 서비스를 생성합니다.
func NewService(config Config, client gmail.Client, eventHandler handler.EventHandler, processedStore store.ProcessedStore, logger *log.Logger) *Service {
	if config.PollInterval == 0 {
		config.PollInterval = DefaultPollInterval
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = DefaultRetentionDays
	}

	return &Service{
		config:         config,
		client:         client,
		handler:        eventHandler,
		processedStore: processedStore,
		logger:         logger,
		stopChan:       make(chan struct{}),
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

	s.logger.Printf("[INFO] 📧 서비스 시작 (폴링 간격: %v)\n", s.config.PollInterval)

	if s.config.LookbackDuration > 0 {
		s.lastTime = time.Now().Add(-s.config.LookbackDuration)
		s.logger.Printf("[INFO] 📅 과거 %v 이내 이메일부터 모니터링\n", s.config.LookbackDuration)
	} else {
		s.lastTime = time.Now()
		s.logger.Println("[INFO] 📅 프로그램 시작 시점부터 모니터링")
	}

	// DB 레코드 수 출력
	if count, err := s.processedStore.GetCount(); err == nil {
		s.logger.Printf("[INFO] 💾 처리된 이메일 DB: %d개 레코드\n", count)
	}

	// 시작 시 오래된 레코드 정리
	if deleted, err := s.processedStore.Cleanup(s.config.RetentionDays); err == nil && deleted > 0 {
		s.logger.Printf("[INFO] 🧹 %d개의 오래된 레코드 정리됨 (%d일 이전)\n", deleted, s.config.RetentionDays)
	}

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	// 시작 직후 첫 체크 수행
	s.checkForNewEmails(ctx)

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

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
}

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

	// DB 기반 중복 제거
	var newMessages []*domain.Message
	for _, msg := range messages {
		// Message-ID 또는 UID 기반 중복 체크
		id := msg.MessageID
		if id == "" {
			id = msg.Timestamp // UID 사용
		}

		processed, err := s.processedStore.IsProcessed(id)
		if err != nil {
			s.logger.Printf("[WARN] ⚠️ 중복 체크 실패: %v\n", err)
			continue
		}

		if !processed {
			newMessages = append(newMessages, msg)
		}
	}

	if len(newMessages) == 0 {
		s.logger.Printf("[INFO] ✅ 체크 완료 - 새 이메일 없음 (총 %d개 이미 처리됨)\n", len(messages))
		return
	}

	s.logger.Printf("[INFO] 📬 %d개의 새 이메일 발견 (총 %d개 중)\n", len(newMessages), len(messages))

	// 새 메시지 처리
	var latestTime time.Time
	for _, msg := range newMessages {
		event := domain.NewMessageEvent(msg)
		s.handler.Handle(event)

		// 처리됨으로 마킹
		id := msg.MessageID
		if id == "" {
			id = msg.Timestamp
		}
		if err := s.processedStore.MarkProcessed(id, msg.Subject); err != nil {
			s.logger.Printf("[WARN] ⚠️ 처리 마킹 실패: %v\n", err)
		}

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
