package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetExecutableDir은 실행 파일 디렉토리 조회를 테스트합니다.
func TestGetExecutableDir(t *testing.T) {
	dir, err := GetExecutableDir()
	if err != nil {
		t.Fatalf("디렉토리 조회 실패: %v", err)
	}
	if dir == "" {
		t.Error("디렉토리가 비어있음")
	}
}

// TestLoadEnvFile은 env 파일 로드를 테스트합니다.
func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "config.env")

	// 테스트용 env 파일 생성
	content := `# 테스트 설정
export TEST_VAR1='value1'
TEST_VAR2=value2
TEST_VAR3="value3"
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("파일 생성 실패: %v", err)
	}

	// 기존 환경변수 삭제
	os.Unsetenv("TEST_VAR1")
	os.Unsetenv("TEST_VAR2")
	os.Unsetenv("TEST_VAR3")

	// 로드
	if err := LoadEnvFile(envPath); err != nil {
		t.Fatalf("로드 실패: %v", err)
	}

	// 검증
	tests := []struct {
		key  string
		want string
	}{
		{"TEST_VAR1", "value1"},
		{"TEST_VAR2", "value2"},
		{"TEST_VAR3", "value3"},
	}

	for _, tt := range tests {
		got := os.Getenv(tt.key)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.key, got, tt.want)
		}
	}
}

// TestLoadEnvFile_NotExists는 파일이 없을 때를 테스트합니다.
func TestLoadEnvFile_NotExists(t *testing.T) {
	err := LoadEnvFile("/nonexistent/path/config.env")
	if err != nil {
		t.Errorf("존재하지 않는 파일은 에러 없이 무시해야 함: %v", err)
	}
}

// TestLoadEnvFile_NoOverwrite는 기존 환경변수를 덮어쓰지 않는지 테스트합니다.
func TestLoadEnvFile_NoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "config.env")

	content := `TEST_EXISTING=new_value`
	os.WriteFile(envPath, []byte(content), 0644)

	// 기존 값 설정
	os.Setenv("TEST_EXISTING", "original_value")
	defer os.Unsetenv("TEST_EXISTING")

	LoadEnvFile(envPath)

	// 기존 값이 유지되어야 함
	if got := os.Getenv("TEST_EXISTING"); got != "original_value" {
		t.Errorf("기존 환경변수가 덮어써짐: %s", got)
	}
}
