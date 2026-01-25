package aiworker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeInvoker는 Claude Code를 실행하는 인터페이스입니다.
type ClaudeInvoker interface {
	InvokePlan(ctx context.Context, workDir, prompt, workerID string) (*InvokeResult, error)
}

// InvokeResult는 Claude Code 실행 결과입니다.
type InvokeResult struct {
	WorkDir   string // 작업 디렉토리
	Prompt    string // 실행된 프롬프트
	StartedAt string // 시작 시간 (ISO 8601)
}

// DefaultInvoker는 실제 Claude Code를 실행합니다.
type DefaultInvoker struct {
	hookServerPort int
	terminalType   TerminalType
}

// NewDefaultInvoker는 새 DefaultInvoker를 생성합니다.
func NewDefaultInvoker() *DefaultInvoker {
	return &DefaultInvoker{
		hookServerPort: 8081,                // 기본 포트
		terminalType:   TerminalTypeDefault, // 기본 터미널
	}
}

// NewDefaultInvokerWithPort는 지정된 Hook 서버 포트로 DefaultInvoker를 생성합니다.
func NewDefaultInvokerWithPort(port int) *DefaultInvoker {
	return &DefaultInvoker{
		hookServerPort: port,
		terminalType:   TerminalTypeDefault,
	}
}

// NewDefaultInvokerWithConfig는 전체 설정으로 DefaultInvoker를 생성합니다.
func NewDefaultInvokerWithConfig(port int, terminalType TerminalType) *DefaultInvoker {
	return &DefaultInvoker{
		hookServerPort: port,
		terminalType:   terminalType,
	}
}

// GetTerminalType은 현재 터미널 타입을 반환합니다.
func (i *DefaultInvoker) GetTerminalType() TerminalType {
	return i.terminalType
}

// InvokePlan은 Claude Code를 플랜 모드로 실행합니다.
// macOS에서 새 터미널 창을 열어 실행합니다.
func (i *DefaultInvoker) InvokePlan(ctx context.Context, workDir, prompt, workerID string) (*InvokeResult, error) {
	// TDD 문구 추가
	fullPrompt := i.AddTDDSuffix(prompt)

	// 프롬프트를 임시 파일에 저장 (이스케이프 문제 회피)
	tmpFile, err := os.CreateTemp("", "claude_prompt_*.txt")
	if err != nil {
		return nil, fmt.Errorf("임시 파일 생성 실패: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(fullPrompt); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("프롬프트 저장 실패: %w", err)
	}
	tmpFile.Close()

	// AppleScript로 새 터미널에서 실행 (파일에서 프롬프트 읽기)
	script := i.BuildAppleScriptWithFile(workDir, tmpPath, workerID)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("Claude Code 실행 실패: %w", err)
	}

	// 임시 파일은 터미널에서 실행 후 삭제됨 (스크립트에서 처리)

	return &InvokeResult{
		WorkDir:   workDir,
		Prompt:    fullPrompt,
		StartedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// BuildCommand는 Claude Code 실행 명령어를 생성합니다.
func (i *DefaultInvoker) BuildCommand(prompt string) string {
	fullPrompt := i.AddTDDSuffix(prompt)
	// 쉘에서 안전하게 실행하기 위해 프롬프트 이스케이프
	escapedPrompt := strings.ReplaceAll(fullPrompt, "'", "'\\''")
	return fmt.Sprintf("claude --plan '%s'", escapedPrompt)
}

// BuildAppleScript는 터미널에서 Claude Code를 실행하는 AppleScript를 생성합니다.
func (i *DefaultInvoker) BuildAppleScript(workDir, prompt string) string {
	// 프롬프트 이스케이프 (쉘과 AppleScript 모두 고려)
	escapedPrompt := strings.ReplaceAll(prompt, "\\", "\\\\")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "\"", "\\\"")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "'", "'\\''")

	// 쉘 특수 문자 이스케이프 (리다이렉션, 파이프, 백그라운드 등)
	escapedPrompt = strings.ReplaceAll(escapedPrompt, ">", "\\>")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "<", "\\<")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "|", "\\|")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "&", "\\&")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "$", "\\$")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "`", "\\`")
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "!", "\\!")

	// 개행 문자를 쉘 호환 형식으로 변환
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "\n", "\\n")

	script := fmt.Sprintf(`
tell application "Terminal"
	activate
	do script "cd '%s' && claude --plan $'%s'"
end tell
`, workDir, escapedPrompt)

	return script
}

// AddTDDSuffix는 프롬프트에 TDD 문구와 작업 완료 알림 지시를 추가합니다.
// 이미 TDD 관련 내용이 있으면 TDD 문구는 추가하지 않습니다.
func (i *DefaultInvoker) AddTDDSuffix(prompt string) string {
	result := prompt

	// TDD 문구 추가
	if !strings.Contains(prompt, "TDD") {
		result += "\n\nTDD 방식으로 개발 진행."
	}

	// 작업 완료 알림 지시 추가
	result += i.GetTaskCompleteInstruction()

	return result
}

// GetTaskCompleteInstruction는 작업 완료 시 알림을 보내도록 하는 프롬프트 지시를 반환합니다.
func (i *DefaultInvoker) GetTaskCompleteInstruction() string {
	return fmt.Sprintf(`

---
## 중요: 작업 완료 알림

모든 작업이 완료되면 반드시 아래 명령을 실행하여 Slack으로 완료 알림을 보내세요:

`+"```bash"+`
curl -s -X POST http://localhost:%d/hook/task-complete -H 'Content-Type: application/json' -d '{"cwd": "'$(pwd)'", "status": "completed"}'
`+"```"+`

작업이 완료되지 않았거나 에러가 발생한 경우에는 이 명령을 실행하지 마세요.
---`, i.hookServerPort)
}

// BuildAppleScriptWithFile는 파일에서 프롬프트를 읽어 Claude Code를 실행하는 AppleScript를 생성합니다.
// 프롬프트를 임시 파일에 저장하고 cat으로 읽어서 claude에 전달합니다.
func (i *DefaultInvoker) BuildAppleScriptWithFile(workDir, promptFilePath, workerID string) string {
	// 경로에 작은따옴표가 있으면 이스케이프
	escapedWorkDir := strings.ReplaceAll(workDir, "'", "'\\''")
	escapedFilePath := strings.ReplaceAll(promptFilePath, "'", "'\\''")

	// TerminalHandler를 통해 AppleScript 생성
	handler := GetTerminalHandler(i.terminalType)
	return handler.BuildInvokeScript(escapedWorkDir, escapedFilePath, workerID)
}
