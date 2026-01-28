package aimodel

import (
	"fmt"
	"os/exec"
)

// AmpcodeHandler는 Ampcode 핸들러입니다.
type AmpcodeHandler struct {
	hookServerPort int
	terminalType   string
}

// NewAmpcodeHandler는 새 Ampcode 핸들러를 생성합니다.
func NewAmpcodeHandler(hookServerPort int, terminalType string) *AmpcodeHandler {
	return &AmpcodeHandler{
		hookServerPort: hookServerPort,
		terminalType:   terminalType,
	}
}

func (h *AmpcodeHandler) GetType() AIModelType {
	return AIModelAmpcode
}

func (h *AmpcodeHandler) GetPlanModeOption() string {
	// Ampcode는 별도의 plan 모드 옵션 없음
	return ""
}

func (h *AmpcodeHandler) BuildInvokeScript(workDir, promptFilePath, workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		return h.buildWarpScript(workDir, promptFilePath, workerID)
	}
	if h.terminalType == string(TerminalTypeITerm2) {
		return h.buildITermScript(workDir, promptFilePath, workerID)
	}
	return h.buildTerminalScript(workDir, promptFilePath, workerID)
}

func (h *AmpcodeHandler) buildTerminalScript(workDir, promptFilePath, workerID string) string {
	// Terminal.app에서 Ampcode 실행
	// Ampcode는 cat으로 프롬프트를 파이프하거나 인자로 전달
	return fmt.Sprintf(`
tell application "Terminal"
	activate
	do script "cd '%s' && cat '%s' | amp && rm -f '%s'"
	set customTitle to "%s"
	set custom title of selected tab of front window to customTitle
end tell
`, workDir, promptFilePath, promptFilePath, workerID)
}

func (h *AmpcodeHandler) buildWarpScript(workDir, promptFilePath, workerID string) string {
	// Warp 터미널에서 Ampcode 실행
	// delay를 늘려 안정성 확보
	return fmt.Sprintf(`
do shell script "open -a Warp '%s'"
delay 2
tell application "System Events"
	tell process "Warp"
		keystroke "t" using {command down}
		delay 1
		keystroke "cd '%s' && cat '%s' | amp && rm -f '%s'"
		delay 0.5
		keystroke return
	end tell
end tell
`, workDir, workDir, promptFilePath, promptFilePath)
}

func (h *AmpcodeHandler) buildITermScript(workDir, promptFilePath, workerID string) string {
	// iTerm2에서 새 탭을 열고 세션 이름 설정 후 Ampcode 실행
	// iTerm2는 AppleScript를 완벽 지원하여 세션 이름으로 특정 탭 타겟팅 가능
	return fmt.Sprintf(`
tell application "iTerm"
	activate
	tell current window
		create tab with default profile
		tell current session
			set name to "%s"
			write text "cd '%s' && cat '%s' | amp && rm -f '%s'"
		end tell
	end tell
end tell
`, workerID, workDir, promptFilePath, promptFilePath)
}

func (h *AmpcodeHandler) BuildTerminateScript(workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		return ""
	}
	if h.terminalType == string(TerminalTypeITerm2) {
		return h.buildITermTerminateScript(workerID)
	}
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

func (h *AmpcodeHandler) buildITermTerminateScript(workerID string) string {
	// iTerm2: 세션 이름으로 특정 세션 찾아 종료
	return fmt.Sprintf(`
tell application "iTerm"
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if name of s is "%s" then
					tell s to close
					return
				end if
			end repeat
		end repeat
	end repeat
end tell
`, workerID)
}

func (h *AmpcodeHandler) Terminate(workerID string) error {
	script := h.BuildTerminateScript(workerID)
	if script == "" {
		return nil
	}
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Ampcode 터미널 종료 실패: %w", err)
	}
	return nil
}

func (h *AmpcodeHandler) GetTaskCompleteInstruction() string {
	return fmt.Sprintf(`

---
## 중요: 작업 완료 알림

모든 작업이 완료되면 반드시 아래 명령을 실행하여 Slack으로 완료 알림을 보내세요:

`+"`"+`bash
curl -s -X POST http://localhost:%d/hook/task-complete -H 'Content-Type: application/json' -d '{"cwd": "'$(pwd)'", "status": "completed"}'
`+"`"+`

작업이 완료되지 않았거나 에러가 발생한 경우에는 이 명령을 실행하지 마세요.
---`, h.hookServerPort)
}
