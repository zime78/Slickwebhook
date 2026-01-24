package hookserver

// StopHookPayload는 Claude Code Stop Hook 페이로드입니다.
type StopHookPayload struct {
	Cwd            string `json:"cwd"`             // 작업 디렉토리
	SessionID      string `json:"session_id"`      // 세션 ID (있는 경우)
	TranscriptPath string `json:"transcript_path"` // 트랜스크립트 경로 (있는 경우)
	ExitCode       int    `json:"exit_code"`       // 종료 코드
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
