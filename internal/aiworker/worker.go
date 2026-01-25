package aiworker

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/zime/slickwebhook/internal/clickup"
	"github.com/zime/slickwebhook/internal/issueformatter"
)

// ClickUpClientInterface는 Worker에서 사용하는 ClickUp 클라이언트 인터페이스입니다.
type ClickUpClientInterface interface {
	GetTask(ctx context.Context, taskID string) (*clickup.Task, error)
	GetTasks(ctx context.Context, listID string, opts *clickup.GetTasksOptions) ([]*clickup.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID, status string) error
	MoveTaskToList(ctx context.Context, taskID, listID string) error
}

// Worker는 단일 AI 리스트를 담당하는 워커입니다.
type Worker struct {
	config          WorkerConfig
	clickupClient   ClickUpClientInterface
	invoker         ClaudeInvoker
	formatter       issueformatter.Formatter
	statusWorking   string
	statusCompleted string
	completedListID string       // 완료된 태스크 이동 목표 리스트 ID
	terminalType    TerminalType // 사용할 터미널 종류

	// 상태 관리
	mu              sync.Mutex
	processing      bool
	currentTaskID   string
	currentTaskName string // Slack 알림용 태스크 이름
	currentJiraID   string // Slack 알림용 Jira 이슈 ID
	originalStatus  string // 취소 시 롤백을 위한 원래 상태
	srcPath         string // 현재 작업 디렉토리 (터미널 종료용)
}

// NewWorker는 새 Worker를 생성합니다.
func NewWorker(
	config WorkerConfig,
	clickupClient ClickUpClientInterface,
	invoker ClaudeInvoker,
	statusWorking, statusCompleted, completedListID string,
) *Worker {
	return &Worker{
		config:          config,
		clickupClient:   clickupClient,
		invoker:         invoker,
		statusWorking:   statusWorking,
		statusCompleted: statusCompleted,
		completedListID: completedListID,
		terminalType:    TerminalTypeDefault,
	}
}

// SetTerminalType은 터미널 타입을 설정합니다.
func (w *Worker) SetTerminalType(terminalType TerminalType) {
	w.terminalType = terminalType
}

// SetFormatter는 이슈 포맷터를 설정합니다.
func (w *Worker) SetFormatter(formatter issueformatter.Formatter) {
	w.formatter = formatter
}

// ProcessTask는 단일 태스크를 처리합니다.
func (w *Worker) ProcessTask(ctx context.Context, taskID string) error {
	// 태스크 조회
	task, err := w.clickupClient.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("태스크 조회 실패: %w", err)
	}
	if task == nil {
		return fmt.Errorf("태스크를 찾을 수 없음: %s", taskID)
	}

	// 원래 상태 저장 (롤백용)
	originalStatus := task.Status.Status

	// Description에서 Jira 이슈 ID 추출
	jiraID := extractJiraID(task.Description)

	// 처리 상태 설정 (태스크 ID, 이름, Jira ID, 원래 상태)
	w.SetProcessing(taskID, task.Name, jiraID, originalStatus)

	// 상태를 "작업중"으로 변경
	if err := w.clickupClient.UpdateTaskStatus(ctx, taskID, w.statusWorking); err != nil {
		w.ClearProcessing()
		return fmt.Errorf("상태 변경 실패: %w", err)
	}

	// 프롬프트 생성
	prompt := w.buildPrompt(ctx, task)

	// Claude Code 실행 (Worker ID 전달)
	_, err = w.invoker.InvokePlan(ctx, w.config.SrcPath, prompt, w.config.ID)
	if err != nil {
		return fmt.Errorf("Claude Code 실행 실패: %w", err)
	}

	return nil
}

// buildPrompt는 태스크에서 AI 프롬프트를 생성합니다.
func (w *Worker) buildPrompt(ctx context.Context, task *clickup.Task) string {
	// issueformatter가 설정되어 있으면 사용
	if w.formatter != nil {
		aiPrompt, err := w.formatter.Format(ctx, task)
		if err == nil && aiPrompt != nil {
			return aiPrompt.Text
		}
	}

	// 기본 프롬프트 생성
	return fmt.Sprintf("# %s\n\n%s\n\n링크: %s", task.Name, task.Description, task.URL)
}

// CompleteTask는 태스크 완료 처리를 수행합니다.
func (w *Worker) CompleteTask(ctx context.Context) error {
	w.mu.Lock()
	taskID := w.currentTaskID
	w.mu.Unlock()

	if taskID == "" {
		return fmt.Errorf("처리 중인 태스크가 없음")
	}

	// 상태를 "개발완료"로 변경
	if err := w.clickupClient.UpdateTaskStatus(ctx, taskID, w.statusCompleted); err != nil {
		return fmt.Errorf("완료 상태 변경 실패: %w", err)
	}

	// 완료 리스트로 이동 (설정된 경우에만)
	if w.completedListID != "" {
		fmt.Printf("[%s] 리스트 이동 시도: 태스크=%s, 목표리스트=%s\n", w.config.ID, taskID, w.completedListID)
		if err := w.clickupClient.MoveTaskToList(ctx, taskID, w.completedListID); err != nil {
			return fmt.Errorf("완료 리스트 이동 실패: %w", err)
		}
		fmt.Printf("[%s] 리스트 이동 성공\n", w.config.ID)
	} else {
		fmt.Printf("[%s] completedListID가 비어있어 리스트 이동 생략\n", w.config.ID)
	}

	// 처리 상태 클리어
	w.ClearProcessing()

	return nil
}

// IsProcessing은 현재 처리 중인지 반환합니다.
func (w *Worker) IsProcessing() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.processing
}

// SetProcessing은 처리 상태를 설정합니다.
func (w *Worker) SetProcessing(taskID, taskName, jiraID, originalStatus string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processing = true
	w.currentTaskID = taskID
	w.currentTaskName = taskName
	w.currentJiraID = jiraID
	w.originalStatus = originalStatus
}

// ClearProcessing은 처리 상태를 클리어합니다.
func (w *Worker) ClearProcessing() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processing = false
	w.currentTaskID = ""
	w.currentTaskName = ""
	w.currentJiraID = ""
	w.originalStatus = ""
}

// RollbackStatus는 취소 시 태스크 상태를 원래 상태로 되돌립니다.
func (w *Worker) RollbackStatus(ctx context.Context) error {
	w.mu.Lock()
	taskID := w.currentTaskID
	originalStatus := w.originalStatus
	w.mu.Unlock()

	if taskID == "" {
		return nil // 처리 중인 태스크 없음
	}

	if originalStatus == "" {
		w.ClearProcessing()
		return nil // 원래 상태 미저장
	}

	// 원래 상태로 변경
	if err := w.clickupClient.UpdateTaskStatus(ctx, taskID, originalStatus); err != nil {
		w.ClearProcessing()
		return fmt.Errorf("상태 롤백 실패: %w", err)
	}

	w.ClearProcessing()
	return nil
}

// GetOriginalStatus는 원래 상태를 반환합니다.
func (w *Worker) GetOriginalStatus() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.originalStatus
}

// GetCurrentTaskID는 현재 처리 중인 태스크 ID를 반환합니다.
func (w *Worker) GetCurrentTaskID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentTaskID
}

// GetCurrentTaskName는 현재 처리 중인 태스크 이름을 반환합니다.
func (w *Worker) GetCurrentTaskName() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentTaskName
}

// GetConfig는 Worker 설정을 반환합니다.
func (w *Worker) GetConfig() WorkerConfig {
	return w.config
}

// GetPendingTasks는 리스트에서 대기 중인 태스크 목록을 조회합니다.
// 완료 상태("개발완료", "배포(QA)", "취소", "완료됨(스토어)")의 태스크는 제외됩니다.
func (w *Worker) GetPendingTasks(ctx context.Context) ([]*clickup.Task, error) {
	opts := &clickup.GetTasksOptions{
		OrderBy: "created",
		Reverse: false, // 오래된 순 (등록순)
	}
	tasks, err := w.clickupClient.GetTasks(ctx, w.config.ListID, opts)
	if err != nil {
		return nil, err
	}

	// 완료 상태 목록 (다시 처리하지 않음)
	completedStatuses := map[string]bool{
		"개발완료":     true,
		"배포(QA)":   true,
		"취소":       true,
		"완료됨(스토어)": true,
		"보류":       true,
	}

	// 완료 상태 태스크 필터링
	var pendingTasks []*clickup.Task
	for _, task := range tasks {
		status := task.Status.Status
		if !completedStatuses[status] {
			pendingTasks = append(pendingTasks, task)
		}
	}

	return pendingTasks, nil
}

// GetCurrentJiraID는 현재 처리 중인 Jira 이슈 ID를 반환합니다.
func (w *Worker) GetCurrentJiraID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentJiraID
}

// extractJiraID는 description에서 Jira 이슈 ID를 추출합니다.
// 예: "[ITSM-5168](https://...)" 또는 "ITSM-5168" 패턴 인식
func extractJiraID(description string) string {
	// [ITSM-xxxx] 또는 ITSM-xxxx 패턴 매칭
	re := regexp.MustCompile(`([A-Z]+-\d+)`)
	if match := re.FindString(description); match != "" {
		return match
	}
	return ""
}

// SetSrcPath는 현재 작업 디렉토리를 설정합니다. (터미널 종료용)
func (w *Worker) SetSrcPath(srcPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.srcPath = srcPath
}

// GetSrcPath는 현재 작업 디렉토리를 반환합니다.
func (w *Worker) GetSrcPath() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.srcPath
}

// TerminateClaude는 현재 실행 중인 Claude 터미널 창을 종료합니다.
// Worker ID로 터미널 창을 식별하여 종료합니다.
func (w *Worker) TerminateClaude() error {
	w.mu.Lock()
	terminalType := w.terminalType
	workerID := w.config.ID
	w.mu.Unlock()

	if workerID == "" {
		return fmt.Errorf("Worker ID가 설정되지 않음")
	}

	// TerminalHandler를 통해 Worker ID로 창 찾아 종료
	handler := GetTerminalHandler(terminalType)
	return handler.Terminate(workerID)
}
