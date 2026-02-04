package aiworker

import (
	"context"
	"log"
	"sync"
	"time"
)

// Manager는 여러 Worker를 관리합니다.
type Manager struct {
	config    Config
	workers   []*Worker
	aiListIDs map[string]bool // AI 리스트 ID 맵 (빠른 조회용)
	mu        sync.RWMutex
	logger    *log.Logger
}

// NewManager는 새 Manager를 생성합니다.
func NewManager(config Config) *Manager {
	m := &Manager{
		config:    config,
		workers:   make([]*Worker, 0, len(config.Workers)),
		aiListIDs: make(map[string]bool),
	}

	// Worker 생성
	for _, wc := range config.Workers {
		worker := NewWorker(wc, nil, nil, config.StatusWorking, config.StatusCompleted, config.CompletedListID)
		m.workers = append(m.workers, worker)
		m.aiListIDs[wc.ListID] = true
	}

	return m
}

// SetLogger는 로거를 설정합니다.
func (m *Manager) SetLogger(logger *log.Logger) {
	m.logger = logger
}

// SetClickUpClient는 모든 Worker에 ClickUp 클라이언트를 설정합니다.
func (m *Manager) SetClickUpClient(client ClickUpClientInterface) {
	for _, w := range m.workers {
		w.clickupClient = client
	}
}

// SetInvoker는 모든 Worker에 Invoker를 설정합니다.
func (m *Manager) SetInvoker(invoker ClaudeInvoker) {
	for _, w := range m.workers {
		w.invoker = invoker
	}
}

// GetWorkers는 모든 Worker를 반환합니다.
func (m *Manager) GetWorkers() []*Worker {
	return m.workers
}

// GetWorkerByListID는 리스트 ID로 Worker를 찾습니다.
func (m *Manager) GetWorkerByListID(listID string) *Worker {
	for _, w := range m.workers {
		if w.config.ListID == listID {
			return w
		}
	}
	return nil
}

// GetWorkerBySrcPath는 소스 경로로 Worker를 찾습니다.
// Claude Code Hook에서 cwd를 기반으로 Worker를 식별할 때 사용합니다.
// 동일한 srcPath의 Worker가 여러 개일 경우, 처리 중인 Worker를 우선 반환합니다.
func (m *Manager) GetWorkerBySrcPath(srcPath string) *Worker {
	var firstMatch *Worker
	for _, w := range m.workers {
		if w.config.SrcPath == srcPath {
			// 처리 중인 Worker 우선 반환
			if w.IsProcessing() {
				return w
			}
			// 첫 번째 매칭된 Worker 저장
			if firstMatch == nil {
				firstMatch = w
			}
		}
	}
	return firstMatch
}

// IsAIList는 주어진 리스트 ID가 AI 리스트인지 확인합니다.
func (m *Manager) IsAIList(listID string) bool {
	return m.aiListIDs[listID]
}

// AllIdle은 모든 Worker가 유휴 상태인지 확인합니다.
func (m *Manager) AllIdle() bool {
	for _, w := range m.workers {
		if w.IsProcessing() {
			return false
		}
	}
	return true
}

// Start는 모든 Worker를 시작합니다.
// 각 Worker는 고루틴에서 자신의 리스트를 모니터링합니다.
func (m *Manager) Start(ctx context.Context) {
	var wg sync.WaitGroup

	for _, worker := range m.workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			m.runWorker(ctx, w)
		}(worker)
	}

	wg.Wait()
}

// runWorker는 개별 Worker의 처리 루프를 실행합니다.
func (m *Manager) runWorker(ctx context.Context, worker *Worker) {
	config := worker.GetConfig()
	if m.logger != nil {
		m.logger.Printf("[%s] Worker 시작 (리스트: %s)", config.ID, config.ListID)
	}

	// 폴링 간격 (CPU 100% 방지)
	pollInterval := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			if m.logger != nil {
				m.logger.Printf("[%s] Worker 종료", config.ID)
			}
			return
		default:
			// 대기 중인 태스크 확인
			if !worker.IsProcessing() {
				tasks, err := worker.GetPendingTasks(ctx)
				if err != nil {
					if m.logger != nil {
						m.logger.Printf("[%s] 태스크 조회 실패: %v", config.ID, err)
					}
					// 에러 시에도 대기 후 재시도
					time.Sleep(pollInterval)
					continue
				}

				// 첫 번째 대기 태스크 처리
				if len(tasks) > 0 {
					task := tasks[0]
					if m.logger != nil {
						m.logger.Printf("[%s] 태스크 처리 시작: %s", config.ID, task.ID)
					}

					if err := worker.ProcessTask(ctx, task.ID); err != nil {
						if m.logger != nil {
							m.logger.Printf("[%s] 태스크 처리 실패: %v", config.ID, err)
						}
					}
				}
			}

			// 다음 폴링까지 대기 (CPU 100% busy-wait 방지)
			time.Sleep(pollInterval)
		}
	}
}

// OnHookReceived는 Claude Code Hook 수신 시 호출됩니다.
// srcPath를 기반으로 해당 Worker를 찾아 완료 처리합니다.
func (m *Manager) OnHookReceived(ctx context.Context, srcPath string) error {
	worker := m.GetWorkerBySrcPath(srcPath)
	if worker == nil {
		if m.logger != nil {
			m.logger.Printf("[Manager] 경로에 해당하는 Worker를 찾을 수 없음: %s", srcPath)
		}
		return nil
	}

	if m.logger != nil {
		m.logger.Printf("[%s] Hook 수신, 완료 처리 시작", worker.GetConfig().ID)
	}

	return worker.CompleteTask(ctx)
}

// GetConfig는 Manager 설정을 반환합니다.
func (m *Manager) GetConfig() Config {
	return m.config
}
