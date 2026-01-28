// Package aimodel은 다양한 AI 코딩 도구를 추상화하는 패키지입니다.
package aimodel

// AIModelType은 사용할 AI 모델 종류입니다.
type AIModelType string

const (
	AIModelClaude   AIModelType = "claude"   // Claude Code (기본값)
	AIModelOpenCode AIModelType = "opencode" // OpenCode
	AIModelAmpcode  AIModelType = "ampcode"  // Ampcode
)

// TerminalType은 사용할 터미널 종류입니다.
type TerminalType string

const (
	TerminalTypeDefault TerminalType = "terminal" // macOS 기본 터미널
	TerminalTypeWarp    TerminalType = "warp"     // Warp 터미널
	TerminalTypeITerm2  TerminalType = "iterm2"   // iTerm2 터미널
)

// AIModelHandler는 AI 모델별 작업을 처리하는 인터페이스입니다.
type AIModelHandler interface {
	// GetType은 AI 모델 타입을 반환합니다.
	GetType() AIModelType

	// BuildInvokeScript는 AI 도구를 실행하는 AppleScript를 생성합니다.
	// workerID는 창 식별에 사용됩니다 (예: "AI_01")
	BuildInvokeScript(workDir, promptFilePath, workerID string) string

	// BuildTerminateScript는 터미널 창을 종료하는 AppleScript를 생성합니다.
	BuildTerminateScript(workerID string) string

	// Terminate는 터미널 창을 종료합니다.
	Terminate(workerID string) error

	// GetPlanModeOption은 계획 모드 옵션을 반환합니다.
	// (claude: --permission-mode plan, opencode: plan, ampcode: "")
	GetPlanModeOption() string

	// GetTaskCompleteInstruction은 작업 완료 알림 지시를 반환합니다.
	GetTaskCompleteInstruction() string
}

// GetAIModelHandler는 AI 모델 타입에 맞는 핸들러를 반환합니다.
func GetAIModelHandler(modelType AIModelType, hookServerPort int, terminalType string) AIModelHandler {
	switch modelType {
	case AIModelOpenCode:
		return NewOpenCodeHandler(hookServerPort, terminalType)
	case AIModelAmpcode:
		return NewAmpcodeHandler(hookServerPort, terminalType)
	default:
		// 기본값은 Claude (빈 문자열, 알 수 없는 타입 포함)
		return NewClaudeHandler(hookServerPort, terminalType)
	}
}
