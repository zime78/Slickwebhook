package claudehook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manager는 Claude Code Hook 설정을 관리합니다.
type Manager struct {
	hookServerPort int
}

// NewManager는 새 Manager를 생성합니다.
func NewManager(hookServerPort int) *Manager {
	return &Manager{
		hookServerPort: hookServerPort,
	}
}

// GenerateHookConfig는 Claude Code Hook 설정을 생성합니다.
func (m *Manager) GenerateHookConfig() map[string]interface{} {
	stopCurlCommand := m.GenerateStopCurlCommand()
	sessionEndCurlCommand := m.GenerateSessionEndCurlCommand()

	return map[string]interface{}{
		"hooks": map[string]interface{}{
			"Stop": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": stopCurlCommand,
							"timeout": 5000,
						},
					},
				},
			},
			"SessionEnd": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": sessionEndCurlCommand,
							"timeout": 5000,
						},
					},
				},
			},
		},
	}
}

// GenerateStopCurlCommand는 Stop Hook 서버로 알림을 보내는 curl 명령어를 생성합니다.
func (m *Manager) GenerateStopCurlCommand() string {
	return fmt.Sprintf(
		`curl -s -X POST http://localhost:%d/hook/stop -H 'Content-Type: application/json' -d '{"cwd": "'"$PWD"'"}'`,
		m.hookServerPort,
	)
}

// GenerateSessionEndCurlCommand는 SessionEnd Hook 서버로 알림을 보내는 curl 명령어를 생성합니다.
// 표준 입력에서 JSON 페이로드를 읽어서 전달합니다.
func (m *Manager) GenerateSessionEndCurlCommand() string {
	return fmt.Sprintf(
		`curl -s -X POST http://localhost:%d/hook/session-end -H 'Content-Type: application/json' -d @-`,
		m.hookServerPort,
	)
}

// WriteSettings는 Claude Code 설정 파일에 Hook 설정을 작성합니다.
func (m *Manager) WriteSettings(settingsPath string) error {
	// 디렉토리 생성
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	config := m.GenerateHookConfig()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}

// MergeSettings는 기존 설정과 병합하여 Hook 설정을 추가합니다.
func (m *Manager) MergeSettings(settingsPath string) error {
	var existingConfig map[string]interface{}

	// 기존 설정 읽기
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			existingConfig = make(map[string]interface{})
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	// hooks 필드 가져오기 또는 생성
	hooks, ok := existingConfig["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
	}

	// Stop hook 설정 추가/업데이트
	newConfig := m.GenerateHookConfig()
	newHooks := newConfig["hooks"].(map[string]interface{})
	hooks["Stop"] = newHooks["Stop"]

	existingConfig["hooks"] = hooks

	// 디렉토리 생성
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// 파일 쓰기
	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}

// GetDefaultSettingsPath는 기본 Claude Code 설정 파일 경로를 반환합니다.
func GetDefaultSettingsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".claude", "settings.json")
}
