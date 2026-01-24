package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/zime/slickwebhook/internal/aiworker"
	"github.com/zime/slickwebhook/internal/claudehook"
	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/config"
	"github.com/zime/slickwebhook/internal/hookserver"
	"github.com/zime/slickwebhook/internal/issueformatter"
	"github.com/zime/slickwebhook/internal/slack"
	"github.com/zime/slickwebhook/internal/webhook"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("[AI Worker] 시작...")

	// 설정 파일 로드
	exeDir, _ := config.GetExecutableDir()
	configPath := filepath.Join(exeDir, "config.email.ini")
	logger.Printf("[AI Worker] 설정 파일 로드: %s", configPath)

	if err := config.LoadEnvFile(configPath); err != nil {
		logger.Printf("[AI Worker] 설정 파일 로드 실패 (무시): %v", err)
	}

	// AI Worker 설정 구성
	workerConfig := loadWorkerConfig(logger)

	// ClickUp 클라이언트 생성
	clickupClient := clickup.NewClickUpClient(clickup.Config{
		APIToken: os.Getenv("CLICKUP_API_TOKEN"),
		TeamID:   os.Getenv("CLICKUP_TEAM_ID"),
	})

	// Slack 클라이언트 생성
	slackClient := slack.NewSlackClient(os.Getenv("SLACK_BOT_TOKEN"))

	// issueformatter 생성
	formatter := issueformatter.NewIssueFormatter(issueformatter.DefaultConfig())

	// Claude Code Invoker 생성
	invoker := aiworker.NewDefaultInvoker()

	// Manager 생성 및 의존성 주입
	manager := aiworker.NewManager(workerConfig)
	manager.SetLogger(logger)
	manager.SetClickUpClient(clickupClient)
	manager.SetInvoker(invoker)

	// 각 Worker에 formatter 설정
	for _, worker := range manager.GetWorkers() {
		worker.SetFormatter(formatter)
	}

	// Claude Code Hook 설정
	hookManager := claudehook.NewManager(workerConfig.HookServerPort)
	settingsPath := claudehook.GetDefaultSettingsPath()
	if err := hookManager.MergeSettings(settingsPath); err != nil {
		logger.Printf("[AI Worker] Claude Hook 설정 실패 (무시): %v", err)
	} else {
		logger.Printf("[AI Worker] Claude Hook 설정 완료: %s", settingsPath)
	}

	// 컨텍스트 설정
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 시그널 핸들링
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Hook 서버 시작 (Claude Code Stop Hook 수신 - 로그만 남김)
	// 참고: Stop Hook은 Rate Limit 등 비정상 종료에도 발생하므로 완료 처리하지 않음
	hookCallback := func(payload *hookserver.StopHookPayload) {
		logger.Printf("[AI Worker] Claude Code Stop Hook 수신: %s (완료 처리는 SessionEnd에서 수행)", payload.Cwd)
	}

	// SessionEnd 콜백 (취소 시 롤백 / 정상 종료 시 완료 처리)
	sessionEndCallback := func(payload *hookserver.SessionEndPayload) {
		logger.Printf("[AI Worker] 세션 종료: cwd=%s, reason=%s", payload.Cwd, payload.Reason)

		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		if worker == nil || !worker.IsProcessing() {
			return
		}

		switch payload.Reason {
		case hookserver.ReasonPromptInputExit:
			// 사용자 취소 시 상태 롤백
			logger.Printf("[AI Worker] 사용자 취소 감지, 상태 롤백 시작...")
			taskID := worker.GetCurrentTaskID()
			originalStatus := worker.GetOriginalStatus()

			if err := worker.RollbackStatus(ctx); err != nil {
				logger.Printf("[AI Worker] 상태 롤백 실패: %v", err)
			} else {
				logger.Printf("[AI Worker] 상태 롤백 완료: 태스크=%s, 원래상태=%s", taskID, originalStatus)
			}

		case hookserver.ReasonOther:
			// 정상 종료 시 완료 처리 (Stop Hook 미발생 fallback)
			logger.Printf("[AI Worker] 정상 종료 감지, 완료 처리 시작...")

			// 완료 처리 전에 태스크 정보 저장 (ClearProcessing 전에)
			taskID := worker.GetCurrentTaskID()
			taskName := worker.GetCurrentTaskName()
			jiraID := worker.GetCurrentJiraID()
			workerID := worker.GetConfig().ID

			if err := manager.OnHookReceived(ctx, payload.Cwd); err != nil {
				logger.Printf("[AI Worker] 완료 처리 실패: %v", err)
			} else {
				logger.Printf("[AI Worker] 완료 처리 성공")
				// Slack 알림 전송 (저장된 태스크 정보 사용)
				sendSlackNotificationWithInfo(ctx, slackClient, workerConfig.SlackChannel, workerID, taskID, taskName, jiraID)
			}
		}
	}

	hookServer := hookserver.NewServer(workerConfig.HookServerPort, hookCallback)
	hookServer.SetLogger(logger)
	hookServer.SetSessionEndCallback(sessionEndCallback)

	// Webhook 서버 시작 (ClickUp 이벤트 수신)
	webhookProcessor := &WebhookProcessor{manager: manager, logger: logger}
	webhookServer := webhook.NewServer(
		webhook.ServerConfig{
			Port:   workerConfig.WebhookPort,
			Secret: os.Getenv("WEBHOOK_SECRET"),
		},
		webhookProcessor,
	)
	webhookServer.SetLogger(logger)

	// 서버 시작
	errChan := make(chan error, 3)

	go func() {
		errChan <- hookServer.Start(ctx)
	}()

	go func() {
		errChan <- webhookServer.Start(ctx)
	}()

	go func() {
		manager.Start(ctx)
		errChan <- nil
	}()

	logger.Println("[AI Worker] 모든 서비스 시작 완료")

	// 종료 대기
	select {
	case sig := <-sigChan:
		logger.Printf("[AI Worker] %v 시그널 수신, 종료 중...", sig)
		cancel()
	case err := <-errChan:
		if err != nil {
			logger.Printf("[AI Worker] 서비스 에러: %v", err)
		}
	}

	logger.Println("[AI Worker] 종료됨")
}

// loadWorkerConfig는 환경변수에서 Worker 설정을 로드합니다.
func loadWorkerConfig(logger *log.Logger) aiworker.Config {
	config := aiworker.DefaultConfig()

	// AI Worker 설정 로드 (AI_01 ~ AI_04)
	for i := 1; i <= 4; i++ {
		prefix := "AI_0" + strconv.Itoa(i)
		listID := os.Getenv(prefix + "_LIST_ID")
		srcPath := os.Getenv(prefix + "_SRC_PATH")

		if listID != "" && srcPath != "" {
			config.AddWorker(prefix, listID, srcPath)
			logger.Printf("[AI Worker] Worker 설정 로드: %s (리스트: %s, 경로: %s)", prefix, listID, srcPath)
		}
	}

	// 레거시 설정 지원 (AI_LIST_IDS, AI_SRC_PATH)
	if len(config.Workers) == 0 {
		listIDsStr := os.Getenv("AI_LIST_IDS")
		srcPath := os.Getenv("AI_SRC_PATH")

		if listIDsStr != "" && srcPath != "" {
			listIDs := strings.Split(listIDsStr, ",")
			for i, listID := range listIDs {
				listID = strings.TrimSpace(listID)
				if listID != "" {
					id := "AI_0" + strconv.Itoa(i+1)
					config.AddWorker(id, listID, srcPath)
					logger.Printf("[AI Worker] Worker 설정 로드 (레거시): %s (리스트: %s)", id, listID)
				}
			}
		}
	}

	// 포트 설정
	if port := os.Getenv("WEBHOOK_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.WebhookPort = p
		}
	}
	if port := os.Getenv("HOOK_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.HookServerPort = p
		}
	}

	// 상태명 설정
	if status := os.Getenv("AI_STATUS_WORKING"); status != "" {
		config.StatusWorking = status
	}
	if status := os.Getenv("AI_STATUS_COMPLETED"); status != "" {
		config.StatusCompleted = status
	}

	// 완료된 태스크 이동 목표 리스트
	config.CompletedListID = os.Getenv("AI_COMPLETED_LIST_ID")
	if config.CompletedListID != "" {
		logger.Printf("[AI Worker] 완료 리스트 설정: %s", config.CompletedListID)
	}

	// Slack 채널
	config.SlackChannel = os.Getenv("SLACK_NOTIFY_CHANNEL")

	return config
}

// WebhookProcessor는 webhook.Processor 인터페이스를 구현합니다.
type WebhookProcessor struct {
	manager *aiworker.Manager
	logger  *log.Logger
}

func (p *WebhookProcessor) EnqueueTask(taskID, listID string) {
	worker := p.manager.GetWorkerByListID(listID)
	if worker != nil && !worker.IsProcessing() {
		p.logger.Printf("[WebhookProcessor] 태스크 처리 시작: %s", taskID)
		// 백그라운드에서 처리
		go func() {
			ctx := context.Background()
			if err := worker.ProcessTask(ctx, taskID); err != nil {
				p.logger.Printf("[WebhookProcessor] 태스크 처리 실패: %v", err)
			}
		}()
	}
}

func (p *WebhookProcessor) IsAIList(listID string) bool {
	return p.manager.IsAIList(listID)
}

// sendSlackNotification는 Slack에 완료 알림을 전송합니다.
func sendSlackNotification(ctx context.Context, client *slack.SlackClient, channelID, cwd string, manager *aiworker.Manager) {
	if channelID == "" {
		return
	}

	worker := manager.GetWorkerBySrcPath(cwd)
	if worker == nil {
		return
	}

	taskID := worker.GetCurrentTaskID()
	taskName := worker.GetCurrentTaskName()
	config := worker.GetConfig()

	message := "✅ AI 작업이 완료되었습니다.\n"
	message += "Worker: " + config.ID + "\n"

	if taskName != "" {
		message += "제목: " + taskName + "\n"
	}

	if taskID != "" {
		message += "ClickUP: https://app.clickup.com/t/" + taskID + "\n"
	}

	// Jira 이슈 ID 추출 (ITSM-xxxx, BUGS-xxxx 등)
	if taskName != "" {
		re := regexp.MustCompile(`([A-Z]+-\d+)`)
		if match := re.FindString(taskName); match != "" {
			message += "Jira 이슈: https://kakaovx.atlassian.net/browse/" + match + "\n"
		}
	}

	client.PostMessage(ctx, channelID, nil, message)
}

// sendSlackNotificationWithInfo는 저장된 태스크 정보로 Slack 알림을 전송합니다.
func sendSlackNotificationWithInfo(ctx context.Context, client *slack.SlackClient, channelID, workerID, taskID, taskName, jiraID string) {
	if channelID == "" {
		return
	}

	message := "✅ AI 작업이 완료되었습니다.\n"
	message += "Worker: " + workerID + "\n"

	if taskName != "" {
		message += "제목: " + taskName + "\n"
	}

	if taskID != "" {
		message += "ClickUP: https://app.clickup.com/t/" + taskID + "\n"
	}

	// Jira 이슈 링크 (description에서 추출된 ID 사용)
	if jiraID != "" {
		message += "Jira 이슈: https://kakaovx.atlassian.net/browse/" + jiraID + "\n"
	}

	client.PostMessage(ctx, channelID, nil, message)
}
