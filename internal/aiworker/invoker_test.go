package aiworker

import (
	"context"
	"strings"
	"testing"
)

// TestInvokeResult는 InvokeResult 구조체를 테스트합니다.
func TestInvokeResult(t *testing.T) {
	result := &InvokeResult{
		WorkDir:   "/test/dir",
		Prompt:    "테스트 프롬프트",
		StartedAt: "2024-01-01T00:00:00Z",
	}

	if result.WorkDir != "/test/dir" {
		t.Errorf("WorkDir 불일치: %s", result.WorkDir)
	}
}

// TestDefaultInvoker_BuildCommand는 Claude Code 명령어 생성을 테스트합니다.
func TestDefaultInvoker_BuildCommand(t *testing.T) {
	invoker := NewDefaultInvoker()

	prompt := "버그를 수정해주세요"
	cmd := invoker.BuildCommand(prompt)

	// 명령어에 claude가 포함되어야 함
	if !strings.Contains(cmd, "claude") {
		t.Error("명령어에 claude가 포함되어야 함")
	}

	// --plan 옵션이 포함되어야 함
	if !strings.Contains(cmd, "--plan") {
		t.Error("명령어에 --plan 옵션이 포함되어야 함")
	}

	// TDD 문구가 추가되어야 함
	if !strings.Contains(cmd, "TDD 방식으로 개발 진행") {
		t.Error("TDD 문구가 포함되어야 함")
	}
}

// TestDefaultInvoker_BuildAppleScript는 AppleScript 생성을 테스트합니다.
func TestDefaultInvoker_BuildAppleScript(t *testing.T) {
	invoker := NewDefaultInvoker()

	workDir := "/test/project"
	prompt := "기능 구현"
	script := invoker.BuildAppleScript(workDir, prompt)

	// Terminal.app 실행 명령 포함
	if !strings.Contains(script, "Terminal") {
		t.Error("AppleScript에 Terminal 명령이 포함되어야 함")
	}

	// 작업 디렉토리 이동 명령 포함
	if !strings.Contains(script, workDir) {
		t.Error("AppleScript에 작업 디렉토리가 포함되어야 함")
	}

	// claude 명령 포함
	if !strings.Contains(script, "claude") {
		t.Error("AppleScript에 claude 명령이 포함되어야 함")
	}
}

// TestDefaultInvoker_AddTDDSuffix는 TDD 문구 추가를 테스트합니다.
func TestDefaultInvoker_AddTDDSuffix(t *testing.T) {
	invoker := NewDefaultInvoker()

	tests := []struct {
		name     string
		prompt   string
		expected string
	}{
		{
			name:     "일반 프롬프트",
			prompt:   "버그 수정",
			expected: "버그 수정\n\nTDD 방식으로 개발 진행.",
		},
		{
			name:     "이미 TDD 포함",
			prompt:   "TDD 방식으로 개발 진행해주세요",
			expected: "TDD 방식으로 개발 진행해주세요", // 중복 추가 안함
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := invoker.AddTDDSuffix(tt.prompt)
			if result != tt.expected {
				t.Errorf("결과 불일치: got %s, want %s", result, tt.expected)
			}
		})
	}
}

// MockInvoker는 테스트용 Mock Invoker입니다.
type MockInvoker struct {
	InvokeCalled bool
	LastWorkDir  string
	LastPrompt   string
	Result       *InvokeResult
	Err          error
}

func (m *MockInvoker) InvokePlan(ctx context.Context, workDir, prompt string) (*InvokeResult, error) {
	m.InvokeCalled = true
	m.LastWorkDir = workDir
	m.LastPrompt = prompt
	return m.Result, m.Err
}

// TestMockInvoker는 Mock Invoker 동작을 테스트합니다.
func TestMockInvoker(t *testing.T) {
	mock := &MockInvoker{
		Result: &InvokeResult{
			WorkDir:   "/test",
			Prompt:    "test prompt",
			StartedAt: "2024-01-01T00:00:00Z",
		},
	}

	ctx := context.Background()
	result, err := mock.InvokePlan(ctx, "/work", "prompt")

	if err != nil {
		t.Fatalf("에러 발생: %v", err)
	}
	if !mock.InvokeCalled {
		t.Error("InvokePlan이 호출되어야 함")
	}
	if mock.LastWorkDir != "/work" {
		t.Errorf("WorkDir 불일치: %s", mock.LastWorkDir)
	}
	if result == nil {
		t.Error("결과가 nil이면 안됨")
	}
}
