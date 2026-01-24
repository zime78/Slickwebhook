package aiworker

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestTaskQueue_EnqueueDequeue는 기본 Enqueue/Dequeue를 테스트합니다.
func TestTaskQueue_EnqueueDequeue(t *testing.T) {
	q := NewTaskQueue()

	// Enqueue
	q.Enqueue("task1", "list1")
	q.Enqueue("task2", "list1")

	if q.Len() != 2 {
		t.Errorf("큐 길이 불일치: got %d, want 2", q.Len())
	}

	// Dequeue
	ctx := context.Background()
	task1, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue 실패: %v", err)
	}
	if task1.TaskID != "task1" {
		t.Errorf("태스크 ID 불일치: got %s, want task1", task1.TaskID)
	}

	task2, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue 실패: %v", err)
	}
	if task2.TaskID != "task2" {
		t.Errorf("태스크 ID 불일치: got %s, want task2", task2.TaskID)
	}

	if q.Len() != 0 {
		t.Errorf("큐가 비어있어야 함: got %d", q.Len())
	}
}

// TestTaskQueue_DequeueBlocking은 블로킹 Dequeue를 테스트합니다.
func TestTaskQueue_DequeueBlocking(t *testing.T) {
	q := NewTaskQueue()
	ctx := context.Background()

	var wg sync.WaitGroup
	var result *QueuedTask

	// 백그라운드에서 Dequeue 대기
	wg.Add(1)
	go func() {
		defer wg.Done()
		task, _ := q.Dequeue(ctx)
		result = task
	}()

	// 잠시 대기 후 Enqueue
	time.Sleep(50 * time.Millisecond)
	q.Enqueue("delayed-task", "list1")

	wg.Wait()

	if result == nil || result.TaskID != "delayed-task" {
		t.Error("블로킹 Dequeue가 정상 동작하지 않음")
	}
}

// TestTaskQueue_DequeueContext는 컨텍스트 취소 시 동작을 테스트합니다.
func TestTaskQueue_DequeueContext(t *testing.T) {
	q := NewTaskQueue()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := q.Dequeue(ctx)

	if err == nil {
		t.Error("컨텍스트 취소 시 에러가 발생해야 함")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("DeadlineExceeded 에러여야 함: got %v", err)
	}
}

// TestTaskQueue_Concurrent는 동시성 안전성을 테스트합니다.
func TestTaskQueue_Concurrent(t *testing.T) {
	q := NewTaskQueue()
	ctx := context.Background()
	const numTasks = 100

	// 동시에 Enqueue
	var wg sync.WaitGroup
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			q.Enqueue("task", "list1")
		}(i)
	}
	wg.Wait()

	if q.Len() != numTasks {
		t.Errorf("큐 길이 불일치: got %d, want %d", q.Len(), numTasks)
	}

	// 동시에 Dequeue
	dequeued := make(chan struct{}, numTasks)
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Dequeue(ctx)
			if err == nil {
				dequeued <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(dequeued)

	count := 0
	for range dequeued {
		count++
	}
	if count != numTasks {
		t.Errorf("Dequeue 개수 불일치: got %d, want %d", count, numTasks)
	}
}

// TestTaskQueue_FIFO는 FIFO 순서를 테스트합니다.
func TestTaskQueue_FIFO(t *testing.T) {
	q := NewTaskQueue()
	ctx := context.Background()

	// 순서대로 Enqueue
	for i := 1; i <= 5; i++ {
		q.Enqueue("task"+string(rune('0'+i)), "list1")
	}

	// 순서대로 Dequeue
	for i := 1; i <= 5; i++ {
		task, _ := q.Dequeue(ctx)
		expected := "task" + string(rune('0'+i))
		if task.TaskID != expected {
			t.Errorf("FIFO 순서 불일치: got %s, want %s", task.TaskID, expected)
		}
	}
}

// TestTaskQueue_TryDequeue는 논블로킹 TryDequeue를 테스트합니다.
func TestTaskQueue_TryDequeue(t *testing.T) {
	q := NewTaskQueue()

	// 빈 큐에서 TryDequeue
	task := q.TryDequeue()
	if task != nil {
		t.Error("빈 큐에서 nil이 반환되어야 함")
	}

	// 태스크 추가 후 TryDequeue
	q.Enqueue("task1", "list1")
	task = q.TryDequeue()
	if task == nil || task.TaskID != "task1" {
		t.Error("TryDequeue가 정상 동작하지 않음")
	}
}
