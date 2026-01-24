package claudehook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestManager_GenerateHookConfig는 Hook 설정 생성을 테스트합니다.
func TestManager_GenerateHookConfig(t *testing.T) {
	manager := NewManager(8081)

	config := manager.GenerateHookConfig()

	if config == nil {
		t.Fatal("설정이 nil입니다")
	}

	hooks, ok := config["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks 필드가 없습니다")
	}

	stopHooks, ok := hooks["Stop"].([]interface{})
	if !ok || len(stopHooks) == 0 {
		t.Fatal("Stop hooks가 없습니다")
	}
}

// TestManager_GenerateCurlCommand는 curl 명령어 생성을 테스트합니다.
func TestManager_GenerateCurlCommand(t *testing.T) {
	manager := NewManager(8081)

	cmd := manager.GenerateCurlCommand()

	if cmd == "" {
		t.Error("명령어가 비어있습니다")
	}
	if !contains(cmd, "localhost:8081") {
		t.Error("포트가 포함되어야 합니다")
	}
	if !contains(cmd, "/hook/stop") {
		t.Error("엔드포인트가 포함되어야 합니다")
	}
}

// TestManager_WriteSettings는 설정 파일 쓰기를 테스트합니다.
func TestManager_WriteSettings(t *testing.T) {
	// 임시 디렉토리 생성
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	manager := NewManager(8081)

	err := manager.WriteSettings(settingsPath)
	if err != nil {
		t.Fatalf("설정 쓰기 실패: %v", err)
	}

	// 파일 확인
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("설정 파일 읽기 실패: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}

	if _, ok := config["hooks"]; !ok {
		t.Error("hooks 필드가 있어야 합니다")
	}
}

// TestManager_MergeSettings는 기존 설정과 병합을 테스트합니다.
func TestManager_MergeSettings(t *testing.T) {
	// 임시 디렉토리 생성
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// 기존 설정 작성
	existingConfig := map[string]interface{}{
		"existingKey": "existingValue",
		"hooks": map[string]interface{}{
			"OtherHook": []interface{}{"something"},
		},
	}
	data, _ := json.MarshalIndent(existingConfig, "", "  ")
	os.WriteFile(settingsPath, data, 0644)

	manager := NewManager(8081)

	err := manager.MergeSettings(settingsPath)
	if err != nil {
		t.Fatalf("설정 병합 실패: %v", err)
	}

	// 파일 확인
	data, _ = os.ReadFile(settingsPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	// 기존 키 유지 확인
	if config["existingKey"] != "existingValue" {
		t.Error("기존 설정이 유지되어야 합니다")
	}

	// hooks 병합 확인
	hooks := config["hooks"].(map[string]interface{})
	if _, ok := hooks["Stop"]; !ok {
		t.Error("Stop hook이 추가되어야 합니다")
	}
	if _, ok := hooks["OtherHook"]; !ok {
		t.Error("기존 OtherHook이 유지되어야 합니다")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
