package hookserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Server는 Claude Code Hook을 수신하는 HTTP 서버입니다.
type Server struct {
	port       int
	callback   HookCallback
	httpServer *http.Server
	logger     *log.Logger
}

// NewServer는 새 Hook 서버를 생성합니다.
func NewServer(port int, callback HookCallback) *Server {
	return &Server{
		port:     port,
		callback: callback,
	}
}

// SetLogger는 로거를 설정합니다.
func (s *Server) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Start는 서버를 시작합니다.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hook/stop", s.handleHook)
	mux.HandleFunc("/health", s.healthHandler)

	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if s.logger != nil {
		s.logger.Printf("[Hook Server] 시작: %s", addr)
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
		s.logger.Println("[Hook Server] 종료 중...")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// handleHook은 Hook 요청을 처리합니다.
func (s *Server) handleHook(w http.ResponseWriter, r *http.Request) {
	// POST 메서드만 허용
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 페이로드 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logError("페이로드 읽기 실패: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 페이로드 파싱
	var payload StopHookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logError("페이로드 파싱 실패: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	s.logInfo("Hook 수신: cwd=%s, exit_code=%d", payload.Cwd, payload.ExitCode)

	// 콜백 호출
	if s.callback != nil {
		s.callback(&payload)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// healthHandler는 헬스체크 엔드포인트입니다.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) logInfo(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Printf("[Hook Server] "+format, args...)
	}
}

func (s *Server) logError(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Printf("[Hook Server ERROR] "+format, args...)
	}
}
