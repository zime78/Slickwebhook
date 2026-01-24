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

	// Claude Code Invoker 생성 (Hook 서버 포트 전달하여 Plan Ready 알림 지원)
	invoker := aiworker.NewDefaultInvokerWithPort(workerConfig.HookServerPort)

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

	// Hook 서버 시작 (Claude Code Stop Hook 수신)
	// Stop 이벤트에 따라 다른 Slack 알림 전송
	hookCallback := func(payload *hookserver.StopHookPayload) {
		logger.Printf("[AI Worker] Claude Code Stop Hook 수신: cwd=%s, permission_mode=%s", payload.Cwd, payload.PermissionMode)

		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		if worker == nil || !worker.IsProcessing() {
			logger.Printf("[AI Worker] Stop Hook: 매칭되는 Worker 없거나 처리 중 아님")
			return
		}

		workerID := worker.GetConfig().ID

		// transcript 파일에서 Stop 원인 분석
		stopReason := analyzeStopReason(payload.TranscriptPath, logger)
		logger.Printf("[AI Worker] Stop 원인 분석: %s", stopReason)

		switch stopReason {
		case StopReasonPlanReady:
			// Plan 완료 - 검토 요청 알림
			if payload.PermissionMode == "plan" {
				logger.Printf("[AI Worker] Plan 완료 감지 - Slack 알림 전송")
				planPayload := &hookserver.PlanReadyPayload{
					Cwd:       payload.Cwd,
					PlanTitle: "계획 수립 완료",
				}
				sendPlanReadySlackNotification(ctx, slackClient, workerConfig.SlackChannel, worker, planPayload)
			}

		case StopReasonRateLimit:
			// Rate Limit 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, "⚠️ Rate Limit", "API 사용량 한도에 도달했습니다. 잠시 후 재시도됩니다.")

		case StopReasonContextExceeded:
			// Context 초과 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, "⚠️ Context 초과", "컨텍스트 윈도우 한도를 초과했습니다.")

		case StopReasonAPIError:
			// API 에러 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, "❌ API 에러", "Claude API 호출 중 에러가 발생했습니다.")

		case StopReasonUnknown:
			// 알 수 없는 Stop - 로그만 남김
			logger.Printf("[AI Worker] 알 수 없는 Stop 원인 (알림 생략)")
		}
	}

	// SessionEnd 콜백 (취소 시 롤백만 수행. 완료 처리는 TaskComplete에서)
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
			// 정상 종료 - 완료 처리는 TaskComplete 콜백에서 수행
			// (Claude가 명시적으로 curl을 호출했을 때만 완료 처리)
			logger.Printf("[AI Worker] 세션 정상 종료 (완료 처리는 Claude의 TaskComplete 알림 대기)")
		}
	}

	hookServer := hookserver.NewServer(workerConfig.HookServerPort, hookCallback)
	hookServer.SetLogger(logger)
	hookServer.SetSessionEndCallback(sessionEndCallback)

	// Plan Ready 콜백 (Plan 완료 시 Slack 알림)
	planReadyCallback := func(payload *hookserver.PlanReadyPayload) {
		logger.Printf("[AI Worker] Plan Ready 수신: cwd=%s, plan=%s", payload.Cwd, payload.PlanTitle)

		// 해당 cwd에 매칭되는 Worker 찾기
		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		if worker == nil {
			logger.Printf("[AI Worker] Plan Ready: 매칭되는 Worker 없음 (cwd=%s)", payload.Cwd)
			return
		}

		// Slack 알림 전송
		sendPlanReadySlackNotification(ctx, slackClient, workerConfig.SlackChannel, worker, payload)
	}
	hookServer.SetPlanReadyCallback(planReadyCallback)

	// TaskComplete 콜백 (Claude가 명시적으로 작업 완료 알림)
	taskCompleteCallback := func(payload *hookserver.TaskCompletePayload) {
		logger.Printf("[AI Worker] TaskComplete 수신: cwd=%s, status=%s", payload.Cwd, payload.Status)

		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		if worker == nil || !worker.IsProcessing() {
			logger.Printf("[AI Worker] TaskComplete: 매칭되는 Worker 없거나 처리 중 아님")
			return
		}

		// 완료 처리 전에 태스크 정보 저장
		taskID := worker.GetCurrentTaskID()
		taskName := worker.GetCurrentTaskName()
		jiraID := worker.GetCurrentJiraID()
		workerID := worker.GetConfig().ID

		if err := manager.OnHookReceived(ctx, payload.Cwd); err != nil {
			logger.Printf("[AI Worker] 완료 처리 실패: %v", err)
		} else {
			logger.Printf("[AI Worker] 완료 처리 성공 (Claude 명시적 완료)")
			// Slack 알림 전송
			sendSlackNotificationWithInfo(ctx, slackClient, workerConfig.SlackChannel, workerID, taskID, taskName, jiraID)
		}
	}
	hookServer.SetTaskCompleteCallback(taskCompleteCallback)

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

// sendPlanReadySlackNotification는 Plan 완료 시 Slack에 검토 요청 알림을 전송합니다.
func sendPlanReadySlackNotification(ctx context.Context, client *slack.SlackClient, channelID string, worker *aiworker.Worker, payload *hookserver.PlanReadyPayload) {
	if channelID == "" {
		return
	}

	config := worker.GetConfig()
	taskID := worker.GetCurrentTaskID()
	taskName := worker.GetCurrentTaskName()
	jiraID := worker.GetCurrentJiraID()

	message := "📋 *계획 수립 완료 - 검토 필요*\n"
	message += "Worker: " + config.ID + "\n"

	if taskName != "" {
		message += "제목: " + taskName + "\n"
	}

	if payload.PlanTitle != "" {
		message += "Plan: " + payload.PlanTitle + "\n"
	}

	if taskID != "" {
		message += "ClickUP: https://app.clickup.com/t/" + taskID + "\n"
	}

	// Jira 이슈 링크
	if jiraID != "" {
		message += "Jira 이슈: https://kakaovx.atlassian.net/browse/" + jiraID + "\n"
	}

	message += "\n⏳ 터미널에서 계획을 검토하고 승인해주세요."

	client.PostMessage(ctx, channelID, nil, message)
}

// Stop 원인 상수
type StopReason string

const (
	StopReasonPlanReady       StopReason = "plan_ready"
	StopReasonRateLimit       StopReason = "rate_limit"
	StopReasonContextExceeded StopReason = "context_exceeded"
	StopReasonAPIError        StopReason = "api_error"
	StopReasonUnknown         StopReason = "unknown"
)

// analyzeStopReason은 transcript 파일을 분석하여 Stop 원인을 반환합니다.
func analyzeStopReason(transcriptPath string, logger *log.Logger) StopReason {
	if transcriptPath == "" {
		return StopReasonUnknown
	}

	// transcript 파일 읽기 (마지막 4KB만 읽어서 성능 최적화)
	file, err := os.Open(transcriptPath)
	if err != nil {
		logger.Printf("[AI Worker] Transcript 파일 열기 실패: %v", err)
		return StopReasonUnknown
	}
	defer file.Close()

	// 파일 끝에서 4KB 읽기
	stat, _ := file.Stat()
	size := stat.Size()
	readSize := int64(4096)
	if size < readSize {
		readSize = size
	}
	file.Seek(-readSize, 2)

	buf := make([]byte, readSize)
	n, _ := file.Read(buf)
	content := strings.ToLower(string(buf[:n]))

	// Plan 완료 확인 (가장 먼저 체크)
	planReadyKeywords := []string{"would you like to proceed", "계획을 검토", "proceed?"}
	for _, keyword := range planReadyKeywords {
		if strings.Contains(content, keyword) {
			return StopReasonPlanReady
		}
	}

	// Rate Limit 확인
	rateLimitKeywords := []string{"hit your limit", "rate limit", "quota exceeded", "limit - resets"}
	for _, keyword := range rateLimitKeywords {
		if strings.Contains(content, keyword) {
			return StopReasonRateLimit
		}
	}

	// Context 초과 확인
	contextKeywords := []string{"context window", "context exceeded", "too long", "max tokens"}
	for _, keyword := range contextKeywords {
		if strings.Contains(content, keyword) {
			return StopReasonContextExceeded
		}
	}

	// API 에러 확인
	errorKeywords := []string{"error", "failed", "exception", "api error"}
	for _, keyword := range errorKeywords {
		if strings.Contains(content, keyword) {
			return StopReasonAPIError
		}
	}

	return StopReasonUnknown
}

// sendStopEventNotification은 Stop 이벤트에 대한 Slack 알림을 전송합니다.
func sendStopEventNotification(ctx context.Context, client *slack.SlackClient, channelID, workerID, eventType, description string) {
	if channelID == "" {
		return
	}

	message := eventType + "\n"
	message += "Worker: " + workerID + "\n"
	message += description

	client.PostMessage(ctx, channelID, nil, message)
}
