package aiworker

import (
	"fmt"
	"os/exec"
)

// TerminalHandler는 터미널 종류별 작업을 처리하는 인터페이스입니다.
type TerminalHandler interface {
	// GetType은 터미널 타입을 반환합니다.
	GetType() TerminalType

	// BuildInvokeScript는 Claude Code를 실행하는 AppleScript를 생성합니다.
	// workerID는 창 식별에 사용됩니다 (예: "AI_01")
	BuildInvokeScript(workDir, promptFilePath, workerID string) string

	// BuildTerminateScript는 터미널 창을 종료하는 AppleScript를 생성합니다.
	// workerID로 특정 창을 찾아 종료합니다.
	BuildTerminateScript(workerID string) string

	// Terminate는 터미널 창을 종료합니다.
	Terminate(workerID string) error
}

// GetTerminalHandler는 터미널 타입에 맞는 핸들러를 반환합니다.
func GetTerminalHandler(terminalType TerminalType) TerminalHandler {
	switch terminalType {
	case TerminalTypeWarp:
		return &WarpTerminalHandler{}
	default:
		return &DefaultTerminalHandler{}
	}
}

// DefaultTerminalHandler는 macOS 기본 Terminal 핸들러입니다.
type DefaultTerminalHandler struct{}

func (h *DefaultTerminalHandler) GetType() TerminalType {
	return TerminalTypeDefault
}

func (h *DefaultTerminalHandler) BuildInvokeScript(workDir, promptFilePath, workerID string) string {
	// Terminal.app에서 새 창을 열고 custom title 설정 후 Claude 실행
	// do script 결과로 탭 참조를 받아 custom title 설정
	return fmt.Sprintf(`
tell application "Terminal"
	activate
	do script "cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
	set customTitle to "%s"
	set custom title of selected tab of front window to customTitle
end tell
`, workDir, promptFilePath, promptFilePath, workerID)
}

func (h *DefaultTerminalHandler) BuildTerminateScript(workerID string) string {
	// custom title이 workerID와 일치하는 창을 찾아 종료
	return fmt.Sprintf(`
tell application "Terminal"
	set windowList to every window
	repeat with w in windowList
		try
			set t to selected tab of w
			if custom title of t is "%s" then
				do script "exit" in t
				delay 0.2
				close w
				return
			end if
		end try
	end repeat
end tell
`, workerID)
}

func (h *DefaultTerminalHandler) Terminate(workerID string) error {
	script := h.BuildTerminateScript(workerID)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Terminal 종료 실패: %w", err)
	}
	return nil
}

// WarpTerminalHandler는 Warp 터미널 핸들러입니다.
type WarpTerminalHandler struct{}

func (h *WarpTerminalHandler) GetType() TerminalType {
	return TerminalTypeWarp
}

func (h *WarpTerminalHandler) BuildInvokeScript(workDir, promptFilePath, workerID string) string {
	// Warp에서 새 창을 열고 탭 타이틀 설정 후 명령어 실행
	// ANSI escape sequence로 탭 타이틀 설정: \033]0;TITLE\007
	return fmt.Sprintf(`
do shell script "open -a Warp '%s'"
delay 1
tell application "System Events"
	tell process "Warp"
		keystroke "t" using {command down}
		delay 0.3
		keystroke "echo -ne '\\033]0;%s\\007' && cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
		keystroke return
	end tell
end tell
`, workDir, workerID, workDir, promptFilePath, promptFilePath)
}

func (h *WarpTerminalHandler) BuildTerminateScript(workerID string) string {
	// Warp는 AppleScript에서 window 조회를 지원하지 않아 창 닫기 제외
	return ""
}

func (h *WarpTerminalHandler) Terminate(workerID string) error {
	// Warp는 AppleScript에서 특정 창 식별이 불가능하여 창 닫기 생략
	// 사용자가 수동으로 창을 닫거나, Claude 완료 후 자동 종료됨
	return nil
}
