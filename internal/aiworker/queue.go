package aiworker

import (
	"context"
	"sync"
	"time"
)

// QueuedTask는 큐에 저장된 태스크입니다.
type QueuedTask struct {
	TaskID    string    // ClickUp 태스크 ID
	ListID    string    // 소속 리스트 ID
	EnqueueAt time.Time // 큐에 추가된 시간
}

// TaskQueue는 처리 대기 태스크 큐입니다.
// FIFO 순서를 보장하며 동시성 안전합니다.
type TaskQueue struct {
	mu       sync.Mutex
	tasks    []*QueuedTask
	notifyCh chan struct{}
}

// NewTaskQueue는 새 태스크 큐를 생성합니다.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:    make([]*QueuedTask, 0),
		notifyCh: make(chan struct{}, 1),
	}
}

// Enqueue는 태스크를 큐에 추가합니다.
func (q *TaskQueue) Enqueue(taskID, listID string) {
	q.mu.Lock()
	q.tasks = append(q.tasks, &QueuedTask{
		TaskID:    taskID,
		ListID:    listID,
		EnqueueAt: time.Now(),
	})
	q.mu.Unlock()

	// 대기 중인 Dequeue에 알림
	select {
	case q.notifyCh <- struct{}{}:
	default:
	}
}

// Dequeue는 다음 태스크를 가져옵니다.
// 큐가 비어있으면 태스크가 추가될 때까지 블로킹합니다.
// 컨텍스트가 취소되면 에러를 반환합니다.
func (q *TaskQueue) Dequeue(ctx context.Context) (*QueuedTask, error) {
	for {
		// 먼저 큐에서 태스크 확인
		if task := q.TryDequeue(); task != nil {
			return task, nil
		}

		// 큐가 비어있으면 대기
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.notifyCh:
			// 알림을 받으면 다시 확인
		}
	}
}

// TryDequeue는 논블로킹으로 태스크를 가져옵니다.
// 큐가 비어있으면 nil을 반환합니다.
func (q *TaskQueue) TryDequeue() *QueuedTask {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tasks) == 0 {
		return nil
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task
}

// Len은 큐의 현재 길이를 반환합니다.
func (q *TaskQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.tasks)
}

// Clear는 큐를 비웁니다.
func (q *TaskQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = q.tasks[:0]
}
