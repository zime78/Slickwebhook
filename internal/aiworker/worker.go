package aiworker

import (
	"context"
	"fmt"
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
	completedListID string // 완료된 태스크 이동 목표 리스트 ID

	// 상태 관리
	mu            sync.Mutex
	processing    bool
	currentTaskID string
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
	}
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

	// 처리 상태 설정
	w.SetProcessing(taskID)

	// 상태를 "작업중"으로 변경
	if err := w.clickupClient.UpdateTaskStatus(ctx, taskID, w.statusWorking); err != nil {
		return fmt.Errorf("상태 변경 실패: %w", err)
	}

	// 프롬프트 생성
	prompt := w.buildPrompt(ctx, task)

	// Claude Code 실행
	_, err = w.invoker.InvokePlan(ctx, w.config.SrcPath, prompt)
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
		if err := w.clickupClient.MoveTaskToList(ctx, taskID, w.completedListID); err != nil {
			return fmt.Errorf("완료 리스트 이동 실패: %w", err)
		}
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
func (w *Worker) SetProcessing(taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processing = true
	w.currentTaskID = taskID
}

// ClearProcessing은 처리 상태를 클리어합니다.
func (w *Worker) ClearProcessing() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processing = false
	w.currentTaskID = ""
}

// GetCurrentTaskID는 현재 처리 중인 태스크 ID를 반환합니다.
func (w *Worker) GetCurrentTaskID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentTaskID
}

// GetConfig는 Worker 설정을 반환합니다.
func (w *Worker) GetConfig() WorkerConfig {
	return w.config
}

// GetPendingTasks는 리스트에서 대기 중인 태스크 목록을 조회합니다.
func (w *Worker) GetPendingTasks(ctx context.Context) ([]*clickup.Task, error) {
	opts := &clickup.GetTasksOptions{
		OrderBy: "created",
		Reverse: false, // 오래된 순 (등록순)
	}
	return w.clickupClient.GetTasks(ctx, w.config.ListID, opts)
}
