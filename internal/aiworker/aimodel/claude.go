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
	if h.terminalType == string(TerminalTypeITerm2) {
		return h.buildITermScript(workDir, promptFilePath, workerID)
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
		keystroke "cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
		keystroke return
	end tell
end tell
`, workDir, workDir, promptFilePath, promptFilePath)
}

func (h *ClaudeHandler) buildITermScript(workDir, promptFilePath, workerID string) string {
	// iTerm2에서 기존 세션 재사용 또는 분할하여 새 세션 생성
	// profile name으로 Worker ID를 설정하여 완전 고정된 식별자로 검색
	return fmt.Sprintf(`
tell application "iTerm"
	activate

	-- 기존 세션 찾기 (profile name으로 Worker ID 검색)
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if profile name of s is "%s" then
					tell s
						write text "cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
					end tell
					return
				end if
			end repeat
		end repeat
	end repeat

	-- 창이 없으면 새 창 생성
	if (count of windows) = 0 then
		create window with default profile
	end if

	-- 세션 없으면 분할하여 새로 생성
	tell current window
		tell current session
			set newSession to (split vertically with default profile)
			tell newSession
				set name to "%s"
				write text "echo -ne '\\033]0;%s\\007' && cd '%s' && cat '%s' | claude --permission-mode plan && rm -f '%s'"
			end tell
		end tell
	end tell
end tell
`, workerID, workDir, promptFilePath, promptFilePath, workerID, workerID, workDir, promptFilePath, promptFilePath)
}

func (h *ClaudeHandler) BuildTerminateScript(workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		// Warp는 AppleScript에서 window 조회를 지원하지 않아 창 닫기 제외
		return ""
	}
	if h.terminalType == string(TerminalTypeITerm2) {
		return h.buildITermTerminateScript(workerID)
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

func (h *ClaudeHandler) buildITermTerminateScript(workerID string) string {
	// iTerm2: profile name으로 Worker ID가 설정된 세션 찾아 종료
	return fmt.Sprintf(`
tell application "iTerm"
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if profile name of s is "%s" then
					tell s to close
					return
				end if
			end repeat
		end repeat
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
