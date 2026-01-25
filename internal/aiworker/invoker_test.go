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

// TestDefaultInvoker_AddTDDSuffix는 TDD 문구와 작업 완료 알림 지시 추가를 테스트합니다.
func TestDefaultInvoker_AddTDDSuffix(t *testing.T) {
	invoker := NewDefaultInvoker()

	t.Run("일반 프롬프트", func(t *testing.T) {
		prompt := "버그 수정"
		result := invoker.AddTDDSuffix(prompt)

		// TDD 문구가 추가되어야 함
		if !strings.Contains(result, "TDD 방식으로 개발 진행") {
			t.Error("TDD 문구가 포함되어야 함")
		}

		// 원본 프롬프트도 포함되어야 함
		if !strings.Contains(result, "버그 수정") {
			t.Error("원본 프롬프트가 포함되어야 함")
		}

		// 작업 완료 알림 지시가 추가되어야 함
		if !strings.Contains(result, "task-complete") {
			t.Error("작업 완료 알림 지시가 포함되어야 함")
		}
	})

	t.Run("이미 TDD 포함", func(t *testing.T) {
		prompt := "TDD 방식으로 개발 진행해주세요"
		result := invoker.AddTDDSuffix(prompt)

		// TDD 문구가 중복 추가되면 안됨 (원본 프롬프트에 하나만 있어야)
		count := strings.Count(result, "TDD 방식으로 개발 진행")
		if count != 1 {
			t.Errorf("TDD 문구는 1회만 포함되어야 함: %d회 발견", count)
		}

		// 작업 완료 알림 지시는 여전히 추가되어야 함
		if !strings.Contains(result, "task-complete") {
			t.Error("작업 완료 알림 지시가 포함되어야 함")
		}
	})
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
