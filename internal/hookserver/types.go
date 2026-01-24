package hookserver

// StopHookPayload는 Claude Code Stop Hook 페이로드입니다.
type StopHookPayload struct {
	Cwd            string `json:"cwd"`              // 작업 디렉토리
	SessionID      string `json:"session_id"`       // 세션 ID (있는 경우)
	TranscriptPath string `json:"transcript_path"`  // 트랜스크립트 경로 (있는 경우)
	ExitCode       int    `json:"exit_code"`        // 종료 코드
	PermissionMode string `json:"permission_mode"`  // 권한 모드 (예: "plan", "default")
	StopHookActive bool   `json:"stop_hook_active"` // Stop Hook 활성 여부
}

// SessionEndPayload는 Claude Code SessionEnd Hook 페이로드입니다.
type SessionEndPayload struct {
	Cwd            string `json:"cwd"`             // 작업 디렉토리
	SessionID      string `json:"session_id"`      // 세션 ID
	TranscriptPath string `json:"transcript_path"` // 트랜스크립트 경로
	Reason         string `json:"reason"`          // 종료 이유: clear, logout, prompt_input_exit, other
	HookEventName  string `json:"hook_event_name"` // 이벤트 이름 (SessionEnd)
}

// 종료 이유 상수
const (
	ReasonClear           = "clear"             // /clear 명령으로 세션 삭제
	ReasonLogout          = "logout"            // 사용자 로그아웃
	ReasonPromptInputExit = "prompt_input_exit" // 사용자가 프롬프트 입력 중 종료 (취소)
	ReasonOther           = "other"             // 정상 종료 등 기타
)

// HookCallback은 Stop Hook 수신 시 호출되는 콜백입니다.
type HookCallback func(payload *StopHookPayload)

// SessionEndCallback은 SessionEnd Hook 수신 시 호출되는 콜백입니다.
type SessionEndCallback func(payload *SessionEndPayload)

// PlanReadyPayload는 Claude Code Plan 완료 알림 페이로드입니다.
// Claude가 프롬프트 지시에 따라 curl로 전송합니다.
type PlanReadyPayload struct {
	Cwd       string `json:"cwd"`        // 작업 디렉토리
	TaskID    string `json:"task_id"`    // ClickUp 태스크 ID (선택)
	TaskName  string `json:"task_name"`  // 태스크 이름 (선택)
	PlanTitle string `json:"plan_title"` // Plan 제목 (선택)
}

// PlanReadyCallback은 Plan 완료 알림 수신 시 호출되는 콜백입니다.
type PlanReadyCallback func(payload *PlanReadyPayload)

// TaskCompletePayload는 작업 완료 알림 페이로드입니다.
// Claude가 프롬프트 지시에 따라 작업 완료 시 curl로 전송합니다.
type TaskCompletePayload struct {
	Cwd    string `json:"cwd"`    // 작업 디렉토리
	Status string `json:"status"` // 상태 (예: "completed")
}

// TaskCompleteCallback은 작업 완료 알림 수신 시 호출되는 콜백입니다.
type TaskCompleteCallback func(payload *TaskCompletePayload)
