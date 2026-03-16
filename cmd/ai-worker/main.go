package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zime/slickwebhook/internal/aiworker"
	"github.com/zime/slickwebhook/internal/claudehook"
	"github.com/zime/slickwebhook/internal/cli"
	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/config"
	"github.com/zime/slickwebhook/internal/hookserver"
	"github.com/zime/slickwebhook/internal/issueformatter"
	"github.com/zime/slickwebhook/internal/slack"
	"github.com/zime/slickwebhook/internal/webhook"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// CLI 인자 파싱
	if cli.ParseArgs(cli.AppInfo{
		Name:        "AI-Worker",
		Description: "AI 작업 자동화 워커 (Claude Code, ClickUp 연동)",
		Version:     cli.GetVersion(),
		ConfigFile:  "config.aiworker.ini",
	}) {
		return
	}

	// 로거 설정 (LOG_TO_FILE 환경변수로 파일 로깅 활성화)
	var logWriter io.Writer = os.Stdout

	if os.Getenv("LOG_TO_FILE") == "1" {
		exeDir, _ := filepath.Abs(".")
		logDir := filepath.Join(exeDir, "logs")
		os.MkdirAll(logDir, 0755)

		logWriter = &lumberjack.Logger{
			Filename:   filepath.Join(logDir, "aiworker.log"),
			MaxSize:    100, // MB
			MaxBackups: 5,   // 파일 개수
			MaxAge:     30,  // 일
			Compress:   true,
		}
	}

	logger := log.New(logWriter, "", log.LstdFlags)
	logger.Println("[AI Worker] 시작...")

	// 설정 파일 로드 (AI Worker 전용 파일 우선, 없으면 email 설정 폴백)
	exeDir, _ := config.GetExecutableDir()
	configPath := filepath.Join(exeDir, "config.aiworker.ini")

	// AI Worker 전용 설정 파일이 없으면 email 설정 파일 사용
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(exeDir, "config.email.ini")
	}

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

	// AI 모델 Invoker 생성 (Hook 서버 포트, 터미널 타입, AI 모델 타입 설정)
	invoker := aiworker.NewDefaultInvokerWithModel(workerConfig.HookServerPort, workerConfig.TerminalType, workerConfig.AIModelType)

	// Manager 생성 및 의존성 주입
	manager := aiworker.NewManager(workerConfig)
	manager.SetLogger(logger)
	manager.SetClickUpClient(clickupClient)
	manager.SetInvoker(invoker)

	// 각 Worker에 formatter 및 터미널 타입 설정
	for _, worker := range manager.GetWorkers() {
		worker.SetFormatter(formatter)
		worker.SetTerminalType(workerConfig.TerminalType)
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

		// Plan 완료 상태인지 확인 (hook의 permission_mode로 판단)
		// Plan 모드면 transcript 분석 없이 바로 알림 전송하고 리턴 (Session ID 잠금 불필요)
		if payload.PermissionMode == "plan" {
			logger.Printf("[AI Worker] Plan 모드 Stop 감지 - Slack 알림 전송")
			planPayload := &hookserver.PlanReadyPayload{
				Cwd:       payload.Cwd,
				PlanTitle: "계획 수립 완료",
			}
			sendPlanReadySlackNotification(ctx, slackClient, workerConfig.SlackChannel, worker, planPayload, payload.SessionID)
			return
		}

		// transcript 파일 및 마지막 메시지에서 Stop 원인 분석 (plan 모드가 아닌 경우)
		stopReason := analyzeStopReason(payload.TranscriptPath, payload.LastAssistantMessage, logger)
		logger.Printf("[AI Worker] Stop 원인 분석: %s", stopReason)

		// 실제 작업 완료 이벤트일 경우에만 현재 세션 ID 필터링/잠금 적용
		// 이렇게 해야 좀비 탭에서 알 수 없는 원인(unknown 등)으로 발생한 Stop Hook이 세션을 선점하는 것을 방지할 수 있음
		if stopReason == StopReasonTaskComplete {
			if worker.GetActiveSessionCount() == 0 {
				worker.AddActiveSession(payload.SessionID)
				logger.Printf("[AI Worker] 작업 완료(TaskComplete) 감지 - 첫 세션 ID 등록: %s", payload.SessionID)
			} else if !worker.HasActiveSession(payload.SessionID) {
				// 이미 다른 세션들이 진행중이고, 현재 세션은 목록에 없는 경우에도 일단 완료 처리를 허용해야 하는가?
				// 만약 완전히 다른 프로젝트 창(하지만 경로는 같음)에서 온 것이라면, 완료 처리를 막는게 맞을지 허용하는게 맞을지 모호함.
				// 하지만 '다중 창 지원'이 핵심이므로, 미등록 SessionID라도 TaskComplete를 외치고 있다면 즉시 등록해주고 완료를 진행시킴.
				worker.AddActiveSession(payload.SessionID)
				logger.Printf("[AI Worker] 작업 완료(TaskComplete) 감지 - 다중 세션 추가 등록: %s", payload.SessionID)
			}
		}

		switch stopReason {
		case StopReasonPlanReady:
			// Plan 완료 - 검토 요청 알림 (fallback)
			logger.Printf("[AI Worker] Plan 완료 감지 (transcript) - Slack 알림 전송")
			planPayload := &hookserver.PlanReadyPayload{
				Cwd:       payload.Cwd,
				PlanTitle: "계획 수립 완료",
			}
			sendPlanReadySlackNotification(ctx, slackClient, workerConfig.SlackChannel, worker, planPayload, payload.SessionID)

		case StopReasonTaskComplete:
			logger.Printf("[AI Worker] Stop Hook에서 작업 완료 감지 (자동 완료 처리 진행)")
			taskID := worker.GetCurrentTaskID()
			taskName := worker.GetCurrentTaskName()
			jiraID := worker.GetCurrentJiraID()
			workerID := worker.GetConfig().ID

			if err := manager.OnHookReceived(ctx, payload.Cwd); err != nil {
				logger.Printf("[AI Worker] 자동 완료 처리 실패 (Stop Hook): %v", err)
			} else {
				logger.Printf("[AI Worker] 자동 완료 처리 성공 (Stop Hook)")
				// Slack 알림 전송
				sendSlackNotificationWithInfo(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, taskID, taskName, jiraID)

				// 0.5초 후 Claude 프로세스 종료
				go func() {
					time.Sleep(500 * time.Millisecond)
					logger.Printf("[AI Worker] Claude 프로세스 종료 중 (Worker: %s)", workerID)
					if err := worker.TerminateClaude(); err != nil {
						logger.Printf("[AI Worker] Claude 종료 실패: %v", err)
					}
				}()
			}

		case StopReasonRateLimit:
			// Rate Limit 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, "⚠️ Rate Limit", "API 사용량 한도에 도달했습니다. 잠시 후 재시도됩니다.")

		case StopReasonContextExceeded:
			// Context 초과 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, "⚠️ Context 초과", "컨텍스트 윈도우 한도를 초과했습니다.")

		case StopReasonAPIError:
			// API 에러 알림
			sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, "❌ API 에러", "Claude API 호출 중 에러가 발생했습니다.")

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

		// 현재 활성화된 세션 중 이번에 종료된 세션이 아니라면 무시
		if worker.GetActiveSessionCount() > 0 && !worker.HasActiveSession(payload.SessionID) {
			logger.Printf("[AI Worker] SessionEnd 무시: 미등록 또는 이전 세션(%s)의 잔류 이벤트", payload.SessionID)
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
			worker.RemoveActiveSession(payload.SessionID) // 세션 목록에서 해당 ID 제거

		default:
			// ReasonOther, ReasonClear 등은 단순 종료 기호일 수 있으므로(예: 창 닫기, clear) 태스크를 완료처리하지 않음
			logger.Printf("[AI Worker] 세션 종료 (완료 처리는 명시적 Hook과 분석에 위임됨)")
			worker.RemoveActiveSession(payload.SessionID) // 종료된 세션 ID 제거
		}
	}

	hookServer := hookserver.NewServer(workerConfig.HookServerPort, hookCallback)
	hookServer.SetLogger(logger)
	hookServer.SetSessionEndCallback(sessionEndCallback)

	// WorktreeCreate 콜백
	worktreeCreateCallback := func(payload *hookserver.WorktreeHookPayload) {
		logger.Printf("[AI Worker] Worktree 생성됨: cwd=%s, worktree=%s", payload.Cwd, payload.Worktree)
		// 워커 식별
		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		workerID := "알 수 없음"
		if worker != nil {
			workerID = worker.GetConfig().ID
		}

		// Slack 알림 (원치 않을 경우 주석 처리)
		sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, "🌳 Worktree 생성됨", fmt.Sprintf("경로: %s", payload.Worktree))
	}
	hookServer.SetWorktreeCreateCallback(worktreeCreateCallback)

	// WorktreeRemove 콜백
	worktreeRemoveCallback := func(payload *hookserver.WorktreeHookPayload) {
		logger.Printf("[AI Worker] Worktree 삭제됨: cwd=%s, worktree=%s", payload.Cwd, payload.Worktree)
		// 워커 식별
		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		workerID := "알 수 없음"
		if worker != nil {
			workerID = worker.GetConfig().ID
		}

		// Slack 알림 (원치 않을 경우 주석 처리)
		sendStopEventNotification(ctx, slackClient, workerConfig.SlackChannel, workerID, payload.SessionID, "🗑️ Worktree 삭제됨", fmt.Sprintf("경로: %s", payload.Worktree))
	}
	hookServer.SetWorktreeRemoveCallback(worktreeRemoveCallback)

	// Plan Ready 콜백 (Plan 완료 시 Slack 알림)
	planReadyCallback := func(payload *hookserver.PlanReadyPayload) {
		logger.Printf("[AI Worker] Plan Ready 수신: cwd=%s, plan=%s", payload.Cwd, payload.PlanTitle)

		// 해당 cwd에 매칭되는 Worker 찾기
		worker := manager.GetWorkerBySrcPath(payload.Cwd)
		if worker == nil {
			logger.Printf("[AI Worker] Plan Ready: 매칭되는 Worker 없음 (cwd=%s)", payload.Cwd)
			return
		}

		// Slack 알림 전송 (명시적 Plan Ready의 경우 sessionID 없음)
		sendPlanReadySlackNotification(ctx, slackClient, workerConfig.SlackChannel, worker, payload, "")
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
			// Slack 알림 전송 (명시적 완료의 경우 sessionID는 없음)
			sendSlackNotificationWithInfo(ctx, slackClient, workerConfig.SlackChannel, workerID, "", taskID, taskName, jiraID)

			// 0.5초 후 Claude 프로세스 종료
			go func() {
				time.Sleep(500 * time.Millisecond)
				logger.Printf("[AI Worker] Claude 프로세스 종료 중 (Worker: %s)", workerID)
				if err := worker.TerminateClaude(); err != nil {
					logger.Printf("[AI Worker] Claude 종료 실패: %v", err)
				}
			}()
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

	// 터미널 타입 설정 (terminal/warp, 기본: terminal)
	terminalType := os.Getenv("TERMINAL_TYPE")
	if terminalType == "warp" {
		config.TerminalType = aiworker.TerminalTypeWarp
		logger.Printf("[AI Worker] 터미널 타입 설정: Warp")
	} else {
		config.TerminalType = aiworker.TerminalTypeDefault
		logger.Printf("[AI Worker] 터미널 타입 설정: Terminal")
	}

	// AI 모델 타입 설정 (claude/opencode/ampcode, 기본: claude)
	aiModelType := os.Getenv("AI_MODEL_TYPE")
	switch aiModelType {
	case "opencode":
		config.AIModelType = aiworker.AIModelOpenCode
		logger.Printf("[AI Worker] AI 모델 설정: OpenCode")
	case "ampcode":
		config.AIModelType = aiworker.AIModelAmpcode
		logger.Printf("[AI Worker] AI 모델 설정: Ampcode")
	default:
		config.AIModelType = aiworker.AIModelClaude
		logger.Printf("[AI Worker] AI 모델 설정: Claude")
	}

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
func sendSlackNotificationWithInfo(ctx context.Context, client *slack.SlackClient, channelID, workerID, sessionID, taskID, taskName, jiraID string) {
	if channelID == "" {
		return
	}

	displayWorkerID := workerID
	if sessionID != "" {
		shortID := sessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		displayWorkerID = fmt.Sprintf("%s(%s)", workerID, shortID)
	}

	message := "✅ AI 작업이 완료되었습니다.\n"
	message += "Worker: " + displayWorkerID + "\n"

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
func sendPlanReadySlackNotification(ctx context.Context, client *slack.SlackClient, channelID string, worker *aiworker.Worker, payload *hookserver.PlanReadyPayload, sessionID string) {
	if channelID == "" {
		return
	}

	config := worker.GetConfig()
	taskID := worker.GetCurrentTaskID()
	taskName := worker.GetCurrentTaskName()
	jiraID := worker.GetCurrentJiraID()

	displayWorkerID := config.ID
	if sessionID != "" {
		shortID := sessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		displayWorkerID = fmt.Sprintf("%s(%s)", config.ID, shortID)
	}

	message := "📋 *계획 수립 완료 - 검토 필요*\n"
	message += "Worker: " + displayWorkerID + "\n"

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
	StopReasonTaskComplete    StopReason = "task_complete"
	StopReasonUnknown         StopReason = "unknown"
)

// analyzeStopReason은 transcript 파일과 마지막 메시지를 분석하여 Stop 원인을 반환합니다.
func analyzeStopReason(transcriptPath string, lastAssistantMessage string, logger *log.Logger) StopReason {
	// 마지막 메시지에서 작업 완료 확인
	if lastAssistantMessage != "" {
		msgLower := strings.ToLower(lastAssistantMessage)
		completeKeywords := []string{
			"작업이 완료", "모든 작업이 완료", "수정이 완료",
			"task completed", "tasks completed", "작업 완료",
			"수정 내역 코멘트 등록 완료",
		}
		for _, keyword := range completeKeywords {
			if strings.Contains(msgLower, keyword) {
				return StopReasonTaskComplete
			}
		}
	}

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

	// API 에러 확인 (너무 포괄적인 error, failed, exception 제외)
	errorKeywords := []string{"api error", "anthropic api", "failed to fetch", "bad gateway", "internal server error", "claude api error", "timeout"}
	for _, keyword := range errorKeywords {
		if strings.Contains(content, keyword) {
			return StopReasonAPIError
		}
	}

	return StopReasonUnknown
}

// sendStopEventNotification은 Stop 이벤트에 대한 Slack 알림을 전송합니다.
func sendStopEventNotification(ctx context.Context, client *slack.SlackClient, channelID, workerID, sessionID, eventType, description string) {
	if channelID == "" {
		return
	}

	displayWorkerID := workerID
	if sessionID != "" {
		shortID := sessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		displayWorkerID = fmt.Sprintf("%s(%s)", workerID, shortID)
	}

	message := eventType + "\n"
	message += "Worker: " + displayWorkerID + "\n"
	message += description

	client.PostMessage(ctx, channelID, nil, message)
}
