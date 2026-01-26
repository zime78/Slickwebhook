package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppInfo는 애플리케이션 정보입니다.
type AppInfo struct {
	Name        string
	Description string
	Version     string
	ConfigFile  string
	Usage       string
}

// ParseArgs는 명령줄 인자를 파싱합니다.
// --help, -h, --version, -v 옵션을 처리합니다.
// 이 옵션들이 사용되면 정보를 출력하고 true를 반환합니다.
func ParseArgs(info AppInfo) bool {
	if len(os.Args) < 2 {
		return false
	}

	arg := os.Args[1]

	switch arg {
	case "-h", "--help":
		printHelp(info)
		return true
	case "-v", "--version":
		printVersion(info)
		return true
	}

	return false
}

func printHelp(info AppInfo) {
	fmt.Printf("%s - %s\n\n", info.Name, info.Description)
	fmt.Printf("버전: %s\n\n", info.Version)
	fmt.Println("사용법:")
	fmt.Printf("  %s [옵션]\n\n", strings.ToLower(info.Name))
	fmt.Println("옵션:")
	fmt.Println("  -h, --help      도움말 표시")
	fmt.Println("  -v, --version   버전 정보 표시")
	fmt.Println()
	fmt.Println("설정 파일:")
	fmt.Printf("  %s (바이너리와 같은 디렉토리)\n\n", info.ConfigFile)
	if info.Usage != "" {
		fmt.Println("상세 사용법:")
		fmt.Println(info.Usage)
	}
	fmt.Println("환경변수:")
	fmt.Println("  LOG_TO_FILE=1   로그를 파일로 저장 (logs/ 디렉토리)")
	fmt.Println()
	fmt.Println("백그라운드 실행:")
	fmt.Printf("  LOG_TO_FILE=1 nohup ./%s > /dev/null 2>&1 &\n", strings.ToLower(info.Name))
}

func printVersion(info AppInfo) {
	fmt.Printf("%s v%s\n", info.Name, info.Version)
}

// GetVersion은 VERSION 파일에서 버전을 읽습니다.
func GetVersion() string {
	exePath, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	exeDir := filepath.Dir(exePath)

	// VERSION 파일 경로들
	paths := []string{
		filepath.Join(exeDir, "VERSION"),
		"VERSION",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return "1.5.0" // 기본값
}
