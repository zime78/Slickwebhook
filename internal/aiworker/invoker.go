package aiworker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClaudeInvoker는 Claude Code를 실행하는 인터페이스입니다.
type ClaudeInvoker interface {
	InvokePlan(ctx context.Context, workDir, prompt string) (*InvokeResult, error)
}

// InvokeResult는 Claude Code 실행 결과입니다.
type InvokeResult struct {
	WorkDir   string // 작업 디렉토리
	Prompt    string // 실행된 프롬프트
	StartedAt string // 시작 시간 (ISO 8601)
}

// DefaultInvoker는 실제 Claude Code를 실행합니다.
type DefaultInvoker struct{}

// NewDefaultInvoker는 새 DefaultInvoker를 생성합니다.
func NewDefaultInvoker() *DefaultInvoker {
	return &DefaultInvoker{}
}

// InvokePlan은 Claude Code를 플랜 모드로 실행합니다.
// macOS에서 새 터미널 창을 열어 실행합니다.
func (i *DefaultInvoker) InvokePlan(ctx context.Context, workDir, prompt string) (*InvokeResult, error) {
	// TDD 문구 추가
	fullPrompt := i.AddTDDSuffix(prompt)

	// AppleScript로 새 터미널에서 실행
	script := i.BuildAppleScript(workDir, fullPrompt)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Claude Code 실행 실패: %w", err)
	}

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

// AddTDDSuffix는 프롬프트에 TDD 문구를 추가합니다.
// 이미 TDD 관련 내용이 있으면 추가하지 않습니다.
func (i *DefaultInvoker) AddTDDSuffix(prompt string) string {
	if strings.Contains(prompt, "TDD") {
		return prompt
	}
	return prompt + "\n\nTDD 방식으로 개발 진행."
}
