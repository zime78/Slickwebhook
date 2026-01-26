package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
// --help, -h, --version, -v, --bg, --status, --stop 옵션을 처리합니다.
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
	case "--bg":
		startBackground(info)
		return true
	case "--status":
		showStatus(info)
		return true
	case "--stop":
		stopProcess(info)
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
	fmt.Println("  --bg            백그라운드로 실행")
	fmt.Println("  --status        실행 상태 확인")
	fmt.Println("  --stop          실행 중인 프로세스 종료")
	fmt.Println()
	fmt.Println("설정 파일:")
	fmt.Printf("  %s (바이너리와 같은 디렉토리)\n\n", info.ConfigFile)
	if info.Usage != "" {
		fmt.Println("상세 사용법:")
		fmt.Println(info.Usage)
		fmt.Println()
	}
	fmt.Println("환경변수:")
	fmt.Println("  LOG_TO_FILE=1   로그를 파일로 저장 (logs/ 디렉토리)")
}

func printVersion(info AppInfo) {
	fmt.Printf("%s v%s\n", info.Name, info.Version)
}

// startBackground는 백그라운드로 프로세스를 시작합니다.
func startBackground(info AppInfo) {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ 실행 파일 경로 조회 실패: %v\n", err)
		os.Exit(1)
	}

	exeDir := filepath.Dir(exePath)
	logDir := filepath.Join(exeDir, "logs")
	os.MkdirAll(logDir, 0755)

	// PID 파일 경로
	pidFile := getPIDFile(info, exeDir)

	// 이미 실행 중인지 확인
	if isRunning(pidFile) {
		fmt.Printf("⚠️  %s가 이미 실행 중입니다.\n", info.Name)
		fmt.Printf("   상태 확인: %s --status\n", strings.ToLower(info.Name))
		return
	}

	// 로그 파일 설정
	logFile := filepath.Join(logDir, strings.ToLower(info.Name)+".log")

	// 백그라운드로 실행 (LOG_TO_FILE=1 환경변수 설정)
	cmd := exec.Command(exePath)
	cmd.Dir = exeDir
	cmd.Env = append(os.Environ(), "LOG_TO_FILE=1")

	// stdout/stderr를 로그 파일로 리다이렉트
	outFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("❌ 로그 파일 생성 실패: %v\n", err)
		os.Exit(1)
	}
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	// 새 프로세스 그룹으로 실행 (터미널 종료해도 유지)
	cmd.SysProcAttr = getSysProcAttr()

	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ 백그라운드 실행 실패: %v\n", err)
		os.Exit(1)
	}

	// PID 저장
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	fmt.Printf("✅ %s가 백그라운드로 시작되었습니다.\n", info.Name)
	fmt.Printf("   PID: %d\n", cmd.Process.Pid)
	fmt.Printf("   로그: %s\n", logFile)
	fmt.Printf("   상태: %s --status\n", strings.ToLower(info.Name))
	fmt.Printf("   종료: %s --stop\n", strings.ToLower(info.Name))
}

// showStatus는 프로세스 상태를 표시합니다.
func showStatus(info AppInfo) {
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	pidFile := getPIDFile(info, exeDir)

	if !isRunning(pidFile) {
		fmt.Printf("❌ %s가 실행 중이 아닙니다.\n", info.Name)
		return
	}

	pidBytes, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidBytes)))

	fmt.Printf("✅ %s가 실행 중입니다.\n", info.Name)
	fmt.Printf("   PID: %d\n", pid)

	logDir := filepath.Join(exeDir, "logs")
	logFile := filepath.Join(logDir, strings.ToLower(info.Name)+".log")
	if _, err := os.Stat(logFile); err == nil {
		fmt.Printf("   로그: %s\n", logFile)
	}
}

// stopProcess는 실행 중인 프로세스를 종료합니다.
func stopProcess(info AppInfo) {
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	pidFile := getPIDFile(info, exeDir)

	if !isRunning(pidFile) {
		fmt.Printf("❌ %s가 실행 중이 아닙니다.\n", info.Name)
		return
	}

	pidBytes, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidBytes)))

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("❌ 프로세스 찾기 실패: %v\n", err)
		return
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("❌ 프로세스 종료 실패: %v\n", err)
		return
	}

	os.Remove(pidFile)
	fmt.Printf("✅ %s가 종료되었습니다. (PID: %d)\n", info.Name, pid)
}

// getPIDFile는 PID 파일 경로를 반환합니다.
func getPIDFile(info AppInfo, exeDir string) string {
	return filepath.Join(exeDir, strings.ToLower(info.Name)+".pid")
}

// isRunning은 프로세스가 실행 중인지 확인합니다.
func isRunning(pidFile string) bool {
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 프로세스가 실제로 존재하는지 확인 (signal 0 전송)
	err = process.Signal(syscall.Signal(0))
	return err == nil
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
