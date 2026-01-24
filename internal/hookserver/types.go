package hookserver

// StopHookPayload는 Claude Code Stop Hook 페이로드입니다.
type StopHookPayload struct {
	Cwd            string `json:"cwd"`             // 작업 디렉토리
	SessionID      string `json:"session_id"`      // 세션 ID (있는 경우)
	TranscriptPath string `json:"transcript_path"` // 트랜스크립트 경로 (있는 경우)
	ExitCode       int    `json:"exit_code"`       // 종료 코드
}

// HookCallback은 Hook 수신 시 호출되는 콜백입니다.
type HookCallback func(payload *StopHookPayload)
