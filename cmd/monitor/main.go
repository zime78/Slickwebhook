package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/config"
	"github.com/zime/slickwebhook/internal/handler"
	"github.com/zime/slickwebhook/internal/history"
	"github.com/zime/slickwebhook/internal/monitor"
	"github.com/zime/slickwebhook/internal/slack"
)

func main() {
	// 로거 설정
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// 실행 파일 디렉토리 가져오기
	exeDir, err := config.GetExecutableDir()
	if err != nil {
		logger.Printf("[WARN] ⚠️ 실행 파일 디렉토리 조회 실패: %v\n", err)
		exeDir = "." // 현재 디렉토리 사용
	}

	// config.ini 파일 로드 (바이너리와 같은 위치)
	configPath := filepath.Join(exeDir, "config.ini")
	if err := config.LoadEnvFile(configPath); err != nil {
		logger.Printf("[WARN] ⚠️ config.env 로드 실패: %v\n", err)
	} else {
		if _, err := os.Stat(configPath); err == nil {
			logger.Printf("[CONFIG] 설정 파일: %s\n", configPath)
		}
	}

	// 환경변수에서 설정 읽기
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		logger.Fatalf("[ERROR] ❌ SLACK_BOT_TOKEN 환경변수가 설정되지 않았습니다\n   config.env 파일을 확인하세요: %s", configPath)
	}

	channelID := os.Getenv("SLACK_CHANNEL_ID")
	if channelID == "" {
		logger.Fatalf("[ERROR] ❌ SLACK_CHANNEL_ID 환경변수가 설정되지 않았습니다\n   config.env 파일을 확인하세요: %s", configPath)
	}

	pollInterval := parseDuration(os.Getenv("POLL_INTERVAL"), 10*time.Second)

	// ClickUp 설정 (선택)
	clickupToken := os.Getenv("CLICKUP_API_TOKEN")
	clickupListID := os.Getenv("CLICKUP_LIST_ID")
	clickupEnabled := clickupToken != "" && clickupListID != ""

	// 히스토리 최대 크기
	historyMaxSize := parseInt(os.Getenv("HISTORY_MAX_SIZE"), 100)

	// 필터 설정
	filterBotOnly := os.Getenv("FILTER_BOT_ONLY") == "true"
	allowedBotIDsStr := os.Getenv("ALLOWED_BOT_IDS")
	var allowedBotIDs []string
	if allowedBotIDsStr != "" {
		for _, id := range strings.Split(allowedBotIDsStr, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				allowedBotIDs = append(allowedBotIDs, id)
			}
		}
	}

	// 히스토리 파일 경로 (바이너리와 같은 위치)
	historyPath := filepath.Join(exeDir, "history.json")

	logger.Println("====================================")
	logger.Println("   Slack 채널 모니터링 서비스")
	logger.Println("====================================")
	logger.Printf("[CONFIG] 실행 디렉토리: %s\n", exeDir)
	logger.Printf("[CONFIG] 채널 ID: %s\n", channelID)
	logger.Printf("[CONFIG] 폴링 간격: %v\n", pollInterval)
	if clickupEnabled {
		logger.Printf("[CONFIG] ClickUp 연동: ✅ 활성화 (List: %s)\n", clickupListID)
		logger.Printf("[CONFIG] 히스토리 파일: %s\n", historyPath)
		logger.Printf("[CONFIG] 히스토리 최대: %d개\n", historyMaxSize)
		if filterBotOnly {
			logger.Println("[CONFIG] 필터링: ✅ 봇 메시지만 처리")
			if len(allowedBotIDs) > 0 {
				logger.Printf("[CONFIG] 허용 봇: %v\n", allowedBotIDs)
			}
		}
	} else {
		logger.Println("[CONFIG] ClickUp 연동: ❌ 비활성화")
	}
	logger.Println("------------------------------------")

	// Slack 클라이언트 생성
	slackClient := slack.NewSlackClient(slackToken)

	// 로그 핸들러 생성
	logHandler := handler.NewLogHandler(logger)

	// 이벤트 핸들러 설정
	var eventHandler handler.EventHandler

	if clickupEnabled {
		// ClickUp 클라이언트 생성
		clickupClient := clickup.NewClickUpClient(clickup.Config{
			APIToken: clickupToken,
			ListID:   clickupListID,
		})

		// 히스토리 저장소 생성 (바이너리와 같은 위치)
		historyStore, err := history.NewFileStore(historyPath, historyMaxSize)
		if err != nil {
			logger.Fatalf("[ERROR] ❌ 히스토리 저장소 생성 실패: %v", err)
		}

		// Forward 핸들러 생성
		forwardHandler := handler.NewForwardHandler(handler.ForwardHandlerConfig{
			ClickUpClient: clickupClient,
			HistoryStore:  historyStore,
			Logger:        logger,
			Enabled:       true,
			FilterBotOnly: filterBotOnly,
			AllowedBotIDs: allowedBotIDs,
		})

		// 체인 핸들러 (로그 -> ClickUp 전송)
		eventHandler = handler.NewChainHandler(logHandler, forwardHandler)
	} else {
		eventHandler = logHandler
	}

	// 모니터 서비스 설정
	config := monitor.Config{
		ChannelID:    channelID,
		PollInterval: pollInterval,
	}

	// 모니터 서비스 생성
	service := monitor.NewService(config, slackClient, eventHandler, logger)

	// 시그널 핸들링 (Ctrl+C, SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Printf("[INFO] 🛑 %v 시그널 수신, 종료 중...\n", sig)
		cancel()
	}()

	// 모니터링 시작
	if err := service.Start(ctx); err != nil && err != context.Canceled {
		logger.Printf("[ERROR] ❌ 서비스 에러: %v\n", err)
		os.Exit(1)
	}

	logger.Println("[INFO] 👋 모니터링 서비스가 정상 종료되었습니다")
}

// parseDuration은 문자열을 Duration으로 파싱합니다.
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

// parseInt는 문자열을 정수로 파싱합니다.
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
