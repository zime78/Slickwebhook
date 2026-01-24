package webhook

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ServerConfig는 웹훅 서버 설정입니다.
type ServerConfig struct {
	Port   int    // 수신 포트
	Secret string // 웹훅 시크릿 (서명 검증용)
}

// Server는 ClickUp 웹훅을 수신하는 HTTP 서버입니다.
type Server struct {
	config     ServerConfig
	handler    *Handler
	httpServer *http.Server
	logger     *log.Logger
}

// NewServer는 새 웹훅 서버를 생성합니다.
func NewServer(config ServerConfig, processor Processor) *Server {
	handler := NewHandler(processor, config.Secret)

	return &Server{
		config:  config,
		handler: handler,
	}
}

// SetLogger는 로거를 설정합니다.
func (s *Server) SetLogger(logger *log.Logger) {
	s.logger = logger
	s.handler.SetLogger(logger)
}

// Start는 서버를 시작합니다.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/clickup", s.handler.HandleWebhook)
	mux.HandleFunc("/health", s.healthHandler)

	addr := fmt.Sprintf(":%d", s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if s.logger != nil {
		s.logger.Printf("[Webhook Server] 시작: %s", addr)
	}

	// 컨텍스트 취소 시 서버 종료
	go func() {
		<-ctx.Done()
		s.Shutdown(context.Background())
	}()

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("서버 시작 실패: %w", err)
	}

	return nil
}

// Shutdown는 서버를 정상 종료합니다.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Println("[Webhook Server] 종료 중...")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// healthHandler는 헬스체크 엔드포인트입니다.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
