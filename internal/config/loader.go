package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GetExecutableDir은 실행 바이너리가 있는 디렉토리 경로를 반환합니다.
func GetExecutableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// 심볼릭 링크 해결
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// LoadEnvFile은 지정된 경로의 .env 파일을 로드합니다.
// 파일이 없으면 무시하고, 있으면 환경변수로 설정합니다.
func LoadEnvFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 파일 없으면 무시
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 빈 줄, 주석 무시
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// export 제거
		line = strings.TrimPrefix(line, "export ")

		// KEY=VALUE 파싱
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 따옴표 제거
		value = strings.Trim(value, `"'`)

		// 이미 설정된 환경변수는 덮어쓰지 않음
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}
