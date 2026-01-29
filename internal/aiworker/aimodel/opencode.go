package aimodel

import (
	"fmt"
	"os/exec"
)

// OpenCodeHandler는 OpenCode 핸들러입니다.
type OpenCodeHandler struct {
	hookServerPort int
	terminalType   string
}

// NewOpenCodeHandler는 새 OpenCode 핸들러를 생성합니다.
func NewOpenCodeHandler(hookServerPort int, terminalType string) *OpenCodeHandler {
	return &OpenCodeHandler{
		hookServerPort: hookServerPort,
		terminalType:   terminalType,
	}
}

func (h *OpenCodeHandler) GetType() AIModelType {
	return AIModelOpenCode
}

func (h *OpenCodeHandler) GetPlanModeOption() string {
	// OpenCode는 `opencode run` 명령어로 프롬프트 실행
	return "run"
}

func (h *OpenCodeHandler) BuildInvokeScript(workDir, promptFilePath, workerID string) string {
	if h.terminalType == string(TerminalTypeWarp) {
		return h.buildWarpScript(workDir, promptFilePath, workerID)
	}
	if h.terminalType == string(TerminalTypeITerm2) {
		return h.buildITermScript(workDir, promptFilePath, workerID)
	}
	return h.buildTerminalScript(workDir, promptFilePath, workerID)
}

func (h *OpenCodeHandler) buildTerminalScript(workDir, promptFilePath, workerID string) string {
	// Terminal.app에서 OpenCode TUI 실행
	// --prompt 옵션으로 초기 프롬프트를 전달하면 TUI 모드에서 대화형으로 작업 가능
	return fmt.Sprintf(`
tell application "Terminal"
	activate
	do script "cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
	set customTitle to "%s"
	set custom title of selected tab of front window to customTitle
end tell
`, workDir, promptFilePath, promptFilePath, workerID)
}

func (h *OpenCodeHandler) buildWarpScript(workDir, promptFilePath, workerID string) string {
	// Warp 터미널에서 OpenCode 실행
	// 새 탭에서 명령어 실행 - delay를 늘려 안정성 확보
	return fmt.Sprintf(`
do shell script "open -a Warp '%s'"
delay 2
tell application "System Events"
	tell process "Warp"
		keystroke "t" using {command down}
		delay 1
		keystroke "cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
		delay 0.5
		keystroke return
	end tell
end tell
`, workDir, workDir, promptFilePath, promptFilePath)
}

func (h *OpenCodeHandler) buildITermScript(workDir, promptFilePath, workerID string) string {
	// iTerm2에서 기존 세션 재사용 또는 2x2 격자로 새 세션 생성
	// AI_01, AI_02: split vertically (좌우)
	// AI_03, AI_04: split horizontally (상하, 위쪽 세션에서 분할)

	workerNum := 1
	if len(workerID) >= 5 {
		if n := workerID[len(workerID)-1]; n >= '1' && n <= '4' {
			workerNum = int(n - '0')
		}
	}

	splitDirection := "vertically"
	targetSession := ""
	if workerNum >= 3 {
		splitDirection = "horizontally"
		targetSession = workerID[:len(workerID)-1] + string('0'+byte(workerNum-2))
	}

	if targetSession != "" {
		return fmt.Sprintf(`
tell application "iTerm"
	activate

	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if name of s is "%s" then
					tell s
						write text "cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
					end tell
					return
				end if
			end repeat
		end repeat
	end repeat

	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if name of s is "%s" then
					tell s
						set newSession to (split %s with default profile)
						tell newSession
							set name to "%s"
							write text "echo -ne '\\033]0;%s\\007' && cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
						end tell
					end tell
					return
				end if
			end repeat
		end repeat
	end repeat

	if (count of windows) = 0 then
		create window with default profile
	end if
	tell current window
		tell current session
			set newSession to (split %s with default profile)
			tell newSession
				set name to "%s"
				write text "echo -ne '\\033]0;%s\\007' && cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
			end tell
		end tell
	end tell
end tell
`, workerID, workDir, promptFilePath, promptFilePath,
			targetSession, splitDirection, workerID, workerID, workDir, promptFilePath, promptFilePath,
			splitDirection, workerID, workerID, workDir, promptFilePath, promptFilePath)
	}

	return fmt.Sprintf(`
tell application "iTerm"
	activate

	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if name of s is "%s" then
					tell s
						write text "cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
					end tell
					return
				end if
			end repeat
		end repeat
	end repeat

	if (count of windows) = 0 then
		create window with default profile
	end if

	tell current window
		tell current session
			set newSession to (split %s with default profile)
			tell newSession
				set name to "%s"
				write text "echo -ne '\\033]0;%s\\007' && cd '%s' && opencode --prompt \"$(cat '%s')\" && rm -f '%s'"
			end tell
		end tell
	end tell
end tell
`, workerID, workDir, promptFilePath, promptFilePath, splitDirection, workerID, workerID, workDir, promptFilePath, promptFilePath)
}

func (h *OpenCodeHandler) BuildTerminateScript(workerID string) string {
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

func (h *OpenCodeHandler) buildITermTerminateScript(workerID string) string {
	// iTerm2: session name으로 Worker ID가 설정된 세션 찾아 종료
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

func (h *OpenCodeHandler) Terminate(workerID string) error {
	script := h.BuildTerminateScript(workerID)
	if script == "" {
		return nil
	}
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("OpenCode 터미널 종료 실패: %w", err)
	}
	return nil
}

func (h *OpenCodeHandler) GetTaskCompleteInstruction() string {
	return fmt.Sprintf(`

ultrawork

---
## 중요: 작업 완료 알림

모든 작업이 완료되면 반드시 아래 명령을 실행하여 Slack으로 완료 알림을 보내세요:

`+"`"+`bash
curl -s -X POST http://localhost:%d/hook/task-complete -H 'Content-Type: application/json' -d '{"cwd": "'$(pwd)'", "status": "completed"}'
`+"`"+`

작업이 완료되지 않았거나 에러가 발생한 경우에는 이 명령을 실행하지 마세요.
---`, h.hookServerPort)
}
