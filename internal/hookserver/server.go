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
	port                 int
	callback             HookCallback
	sessionEndCallback   SessionEndCallback
	planReadyCallback    PlanReadyCallback
	taskCompleteCallback TaskCompleteCallback
	httpServer           *http.Server
	logger               *log.Logger
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

// SetSessionEndCallback은 SessionEnd 콜백을 설정합니다.
func (s *Server) SetSessionEndCallback(callback SessionEndCallback) {
	s.sessionEndCallback = callback
}

// SetPlanReadyCallback은 Plan Ready 콜백을 설정합니다.
func (s *Server) SetPlanReadyCallback(callback PlanReadyCallback) {
	s.planReadyCallback = callback
}

// SetTaskCompleteCallback은 Task Complete 콜백을 설정합니다.
func (s *Server) SetTaskCompleteCallback(callback TaskCompleteCallback) {
	s.taskCompleteCallback = callback
}

// Start는 서버를 시작합니다.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/hook/stop", s.handleHook)
	mux.HandleFunc("/hook/session-end", s.handleSessionEnd)
	mux.HandleFunc("/hook/plan-ready", s.handlePlanReady)
	mux.HandleFunc("/hook/task-complete", s.handleTaskComplete)
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

	// 원본 페이로드 로깅 (디버깅용)
	s.logInfo("Stop Hook 원본 데이터: %s", string(body))

	// 페이로드 파싱
	var payload StopHookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logError("페이로드 파싱 실패: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	s.logInfo("Stop Hook 파싱 결과: cwd=%s, permission_mode=%s, exit_code=%d",
		payload.Cwd, payload.PermissionMode, payload.ExitCode)

	// 콜백 호출
	if s.callback != nil {
		s.callback(&payload)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleSessionEnd는 SessionEnd Hook 요청을 처리합니다.
func (s *Server) handleSessionEnd(w http.ResponseWriter, r *http.Request) {
	// POST 메서드만 허용
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 페이로드 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logError("SessionEnd 페이로드 읽기 실패: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 페이로드 파싱
	var payload SessionEndPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logError("SessionEnd 페이로드 파싱 실패: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	// 종료 사유 로깅
	reasonDesc := s.getReasonDescription(payload.Reason)
	s.logInfo("SessionEnd 수신: cwd=%s, reason=%s (%s)", payload.Cwd, payload.Reason, reasonDesc)

	// 콜백 호출
	if s.sessionEndCallback != nil {
		s.sessionEndCallback(&payload)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handlePlanReady는 Claude Code Plan 완료 알림을 처리합니다.
// Claude가 프롬프트 지시에 따라 curl로 호출합니다.
func (s *Server) handlePlanReady(w http.ResponseWriter, r *http.Request) {
	// POST 메서드만 허용
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 페이로드 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logError("PlanReady 페이로드 읽기 실패: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 페이로드 파싱
	var payload PlanReadyPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logError("PlanReady 페이로드 파싱 실패: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	s.logInfo("PlanReady 수신: cwd=%s, task=%s", payload.Cwd, payload.TaskName)

	// 콜백 호출
	if s.planReadyCallback != nil {
		s.planReadyCallback(&payload)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleTaskComplete는 작업 완료 알림을 처리합니다.
// Claude가 프롬프트 지시에 따라 작업 완료 시 curl로 호출합니다.
func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	// POST 메서드만 허용
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 페이로드 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logError("TaskComplete 페이로드 읽기 실패: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 페이로드 파싱
	var payload TaskCompletePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.logError("TaskComplete 페이로드 파싱 실패: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	s.logInfo("TaskComplete 수신: cwd=%s, status=%s", payload.Cwd, payload.Status)

	// 콜백 호출
	if s.taskCompleteCallback != nil {
		s.taskCompleteCallback(&payload)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// getReasonDescription은 종료 사유에 대한 설명을 반환합니다.
func (s *Server) getReasonDescription(reason string) string {
	switch reason {
	case ReasonClear:
		return "세션 삭제"
	case ReasonLogout:
		return "로그아웃"
	case ReasonPromptInputExit:
		return "사용자 취소"
	case ReasonOther:
		return "정상 종료"
	default:
		return "알 수 없음"
	}
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
