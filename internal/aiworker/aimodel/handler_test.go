package aimodel

import (
	"strings"
	"testing"
)

// TestAIModelType은 AIModelType 상수를 테스트합니다.
func TestAIModelType(t *testing.T) {
	tests := []struct {
		name     string
		model    AIModelType
		expected string
	}{
		{"Claude", AIModelClaude, "claude"},
		{"OpenCode", AIModelOpenCode, "opencode"},
		{"Ampcode", AIModelAmpcode, "ampcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.model) != tt.expected {
				t.Errorf("AIModelType %s = %s, want %s", tt.name, tt.model, tt.expected)
			}
		})
	}
}

// TestGetAIModelHandler_Factory는 팩토리 함수를 테스트합니다.
func TestGetAIModelHandler_Factory(t *testing.T) {
	tests := []struct {
		name      string
		modelType AIModelType
		expected  AIModelType
	}{
		{"Claude 기본", AIModelClaude, AIModelClaude},
		{"OpenCode", AIModelOpenCode, AIModelOpenCode},
		{"Ampcode", AIModelAmpcode, AIModelAmpcode},
		{"빈 문자열은 Claude", "", AIModelClaude},
		{"알 수 없는 타입은 Claude", "unknown", AIModelClaude},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := GetAIModelHandler(tt.modelType, 8081, "terminal")
			if handler.GetType() != tt.expected {
				t.Errorf("GetAIModelHandler(%s) type = %s, want %s", tt.modelType, handler.GetType(), tt.expected)
			}
		})
	}
}

// TestClaudeHandler는 Claude 핸들러를 테스트합니다.
func TestClaudeHandler(t *testing.T) {
	handler := NewClaudeHandler(8081, "terminal")

	t.Run("GetType", func(t *testing.T) {
		if handler.GetType() != AIModelClaude {
			t.Errorf("GetType() = %s, want %s", handler.GetType(), AIModelClaude)
		}
	})

	t.Run("GetPlanModeOption", func(t *testing.T) {
		option := handler.GetPlanModeOption()
		if option != "--permission-mode plan" {
			t.Errorf("GetPlanModeOption() = %s, want --permission-mode plan", option)
		}
	})

	t.Run("BuildInvokeScript Terminal", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// Terminal 명령이 포함되어야 함
		if !strings.Contains(script, "Terminal") {
			t.Error("스크립트에 Terminal이 포함되어야 함")
		}

		// claude 명령이 포함되어야 함
		if !strings.Contains(script, "claude") {
			t.Error("스크립트에 claude가 포함되어야 함")
		}

		// --permission-mode plan이 포함되어야 함
		if !strings.Contains(script, "--permission-mode plan") {
			t.Error("스크립트에 --permission-mode plan이 포함되어야 함")
		}

		// 작업 디렉토리가 포함되어야 함
		if !strings.Contains(script, "/test/dir") {
			t.Error("스크립트에 작업 디렉토리가 포함되어야 함")
		}

		// 프롬프트 파일이 포함되어야 함
		if !strings.Contains(script, "/tmp/prompt.txt") {
			t.Error("스크립트에 프롬프트 파일이 포함되어야 함")
		}

		// Worker ID가 포함되어야 함 (custom title)
		if !strings.Contains(script, "AI_01") {
			t.Error("스크립트에 Worker ID가 포함되어야 함")
		}
	})

	t.Run("GetTaskCompleteInstruction", func(t *testing.T) {
		instruction := handler.GetTaskCompleteInstruction()

		// task-complete 엔드포인트가 포함되어야 함
		if !strings.Contains(instruction, "task-complete") {
			t.Error("지시에 task-complete가 포함되어야 함")
		}

		// 포트 번호가 포함되어야 함
		if !strings.Contains(instruction, "8081") {
			t.Error("지시에 포트 번호가 포함되어야 함")
		}
	})
}

// TestOpenCodeHandler는 OpenCode 핸들러를 테스트합니다.
func TestOpenCodeHandler(t *testing.T) {
	handler := NewOpenCodeHandler(8081, "terminal")

	t.Run("GetType", func(t *testing.T) {
		if handler.GetType() != AIModelOpenCode {
			t.Errorf("GetType() = %s, want %s", handler.GetType(), AIModelOpenCode)
		}
	})

	t.Run("GetPlanModeOption", func(t *testing.T) {
		option := handler.GetPlanModeOption()
		if option != "run" {
			t.Errorf("GetPlanModeOption() = %s, want run", option)
		}
	})

	t.Run("BuildInvokeScript", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// opencode 명령이 포함되어야 함
		if !strings.Contains(script, "opencode") {
			t.Error("스크립트에 opencode가 포함되어야 함")
		}

		// 작업 디렉토리가 포함되어야 함
		if !strings.Contains(script, "/test/dir") {
			t.Error("스크립트에 작업 디렉토리가 포함되어야 함")
		}
	})
}

// TestAmpcodeHandler는 Ampcode 핸들러를 테스트합니다.
func TestAmpcodeHandler(t *testing.T) {
	handler := NewAmpcodeHandler(8081, "terminal")

	t.Run("GetType", func(t *testing.T) {
		if handler.GetType() != AIModelAmpcode {
			t.Errorf("GetType() = %s, want %s", handler.GetType(), AIModelAmpcode)
		}
	})

	t.Run("GetPlanModeOption", func(t *testing.T) {
		// Ampcode는 별도 plan 모드 옵션 없음
		option := handler.GetPlanModeOption()
		if option != "" {
			t.Errorf("GetPlanModeOption() = %s, want empty", option)
		}
	})

	t.Run("BuildInvokeScript", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// amp 명령이 포함되어야 함
		if !strings.Contains(script, "amp") {
			t.Error("스크립트에 amp가 포함되어야 함")
		}

		// 작업 디렉토리가 포함되어야 함
		if !strings.Contains(script, "/test/dir") {
			t.Error("스크립트에 작업 디렉토리가 포함되어야 함")
		}
	})
}

// TestAIModelHandler_Interface는 모든 핸들러가 인터페이스를 만족하는지 테스트합니다.
func TestAIModelHandler_Interface(t *testing.T) {
	handlers := []AIModelHandler{
		NewClaudeHandler(8081, "terminal"),
		NewOpenCodeHandler(8081, "terminal"),
		NewAmpcodeHandler(8081, "terminal"),
	}

	for _, h := range handlers {
		t.Run(string(h.GetType()), func(t *testing.T) {
			// 인터페이스 메서드 호출 가능 확인
			_ = h.GetType()
			_ = h.BuildInvokeScript("/test", "/tmp/prompt.txt", "AI_01")
			_ = h.BuildTerminateScript("AI_01")
			_ = h.GetPlanModeOption()
			_ = h.GetTaskCompleteInstruction()
		})
	}
}

// TestTerminalType은 터미널 타입 상수를 테스트합니다.
func TestTerminalType(t *testing.T) {
	tests := []struct {
		name     string
		terminal TerminalType
		expected string
	}{
		{"Terminal", TerminalTypeDefault, "terminal"},
		{"Warp", TerminalTypeWarp, "warp"},
		{"iTerm2", TerminalTypeITerm2, "iterm2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.terminal) != tt.expected {
				t.Errorf("TerminalType %s = %s, want %s", tt.name, tt.terminal, tt.expected)
			}
		})
	}
}

// TestClaudeHandler_iTerm2는 Claude 핸들러의 iTerm2 스크립트를 테스트합니다.
func TestClaudeHandler_iTerm2(t *testing.T) {
	handler := NewClaudeHandler(8081, "iterm2")

	t.Run("BuildInvokeScript iTerm2", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}

		// claude 명령이 포함되어야 함
		if !strings.Contains(script, "claude") {
			t.Error("스크립트에 claude가 포함되어야 함")
		}

		// Worker ID (세션 이름)가 포함되어야 함
		if !strings.Contains(script, "AI_01") {
			t.Error("스크립트에 Worker ID가 포함되어야 함")
		}

		// split vertically가 포함되어야 함 (분할 모드)
		if !strings.Contains(script, "split vertically") {
			t.Error("스크립트에 'split vertically'가 포함되어야 함")
		}
	})

	t.Run("BuildTerminateScript iTerm2", func(t *testing.T) {
		script := handler.BuildTerminateScript("AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}

		// session name으로 찾는 로직이 포함되어야 함 (완전 일치 검색)
		if !strings.Contains(script, "name of s is") {
			t.Error("스크립트에 session name 검색 로직이 포함되어야 함")
		}
	})
}

// TestOpenCodeHandler_iTerm2는 OpenCode 핸들러의 iTerm2 스크립트를 테스트합니다.
func TestOpenCodeHandler_iTerm2(t *testing.T) {
	handler := NewOpenCodeHandler(8081, "iterm2")

	t.Run("BuildInvokeScript iTerm2", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}

		// opencode 명령이 포함되어야 함
		if !strings.Contains(script, "opencode") {
			t.Error("스크립트에 opencode가 포함되어야 함")
		}

		// Worker ID (세션 이름)가 포함되어야 함
		if !strings.Contains(script, "AI_01") {
			t.Error("스크립트에 Worker ID가 포함되어야 함")
		}
	})

	t.Run("BuildTerminateScript iTerm2", func(t *testing.T) {
		script := handler.BuildTerminateScript("AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}
	})
}

// TestAmpcodeHandler_iTerm2는 Ampcode 핸들러의 iTerm2 스크립트를 테스트합니다.
func TestAmpcodeHandler_iTerm2(t *testing.T) {
	handler := NewAmpcodeHandler(8081, "iterm2")

	t.Run("BuildInvokeScript iTerm2", func(t *testing.T) {
		script := handler.BuildInvokeScript("/test/dir", "/tmp/prompt.txt", "AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}

		// amp 명령이 포함되어야 함
		if !strings.Contains(script, "amp") {
			t.Error("스크립트에 amp가 포함되어야 함")
		}

		// Worker ID (세션 이름)가 포함되어야 함
		if !strings.Contains(script, "AI_01") {
			t.Error("스크립트에 Worker ID가 포함되어야 함")
		}
	})

	t.Run("BuildTerminateScript iTerm2", func(t *testing.T) {
		script := handler.BuildTerminateScript("AI_01")

		// iTerm 명령이 포함되어야 함
		if !strings.Contains(script, "iTerm") {
			t.Error("스크립트에 iTerm이 포함되어야 함")
		}
	})
}
