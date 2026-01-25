package aimodel

import (
	"fmt"
	"os/exec"
)

// ClaudeHandler는 Claude Code 핸들러입니다.
type ClaudeHandler struct {
	hookServerPort int
	terminalType   string
}

// NewClaudeHandler는 새 Claude 핸들러를 생성합니다.
func NewClaudeHandler(hookServerPort int, terminalType string) *ClaudeHandler {
	return &ClaudeHandler{
		hookServerPort: hookServerPort,
		terminalType:   terminalType,
	}
}

func (h *ClaudeHandler) GetType() AIModelType {
	return AIModelClaude
}

func (h *ClaudeHandler) GetPlanModeOption() string {
	return "--permission-mode plan"
}

func (h *ClaudeHandler) BuildInvokeScript(workDir, promptFilePath, workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		return h.buildWarpScript(workDir, promptFilePath, workerID)
	}
	return h.buildTerminalScript(workDir, promptFilePath, workerID)
}

func (h *ClaudeHandler) buildTerminalScript(workDir, promptFilePath, workerID string) string {
	// Terminal.app에서 새 창을 열고 custom title 설정 후 Claude 실행
	return fmt.Sprintf(`
tell application "Terminal"
	activate
	do script "cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
	set customTitle to "%s"
	set custom title of selected tab of front window to customTitle
end tell
`, workDir, promptFilePath, promptFilePath, workerID)
}

func (h *ClaudeHandler) buildWarpScript(workDir, promptFilePath, workerID string) string {
	// Warp에서 새 창을 열고 Claude 실행
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

func (h *ClaudeHandler) BuildTerminateScript(workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		// Warp는 AppleScript에서 window 조회를 지원하지 않아 창 닫기 제외
		return ""
	}
	// Terminal.app: custom title로 창 찾아 종료
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

func (h *ClaudeHandler) Terminate(workerID string) error {
	script := h.BuildTerminateScript(workerID)
	if script == "" {
		return nil // Warp 등 종료 생략
	}
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude 터미널 종료 실패: %w", err)
	}
	return nil
}

func (h *ClaudeHandler) GetTaskCompleteInstruction() string {
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
