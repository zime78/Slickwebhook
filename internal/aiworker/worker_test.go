package aiworker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zime/slickwebhook/internal/clickup"
)

// MockClickUpClient는 테스트용 ClickUp 클라이언트입니다.
type MockClickUpClient struct {
	Tasks           []*clickup.Task
	StatusUpdates   []StatusUpdate
	MovedTasks      []MoveTask
	GetTasksCalled  bool
	UpdateCalled    bool
	MoveTaskCalled  bool
	mu              sync.Mutex
}

type StatusUpdate struct {
	TaskID string
	Status string
}

type MoveTask struct {
	TaskID string
	ListID string
}

func (m *MockClickUpClient) GetTasks(ctx context.Context, listID string, opts *clickup.GetTasksOptions) ([]*clickup.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetTasksCalled = true
	return m.Tasks, nil
}

func (m *MockClickUpClient) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalled = true
	m.StatusUpdates = append(m.StatusUpdates, StatusUpdate{TaskID: taskID, Status: status})
	return nil
}

func (m *MockClickUpClient) GetTask(ctx context.Context, taskID string) (*clickup.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, task := range m.Tasks {
		if task.ID == taskID {
			return task, nil
		}
	}
	return nil, nil
}

func (m *MockClickUpClient) MoveTaskToList(ctx context.Context, taskID, listID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MoveTaskCalled = true
	m.MovedTasks = append(m.MovedTasks, MoveTask{TaskID: taskID, ListID: listID})
	return nil
}

func (m *MockClickUpClient) CreateTask(ctx context.Context, msg interface{}) (*clickup.TaskResponse, error) {
	return nil, nil
}

func (m *MockClickUpClient) UploadAttachment(ctx context.Context, taskID, filename string, data []byte) error {
	return nil
}

// TestWorker_ProcessTask는 단일 태스크 처리를 테스트합니다.
func TestWorker_ProcessTask(t *testing.T) {
	mockClient := &MockClickUpClient{
		Tasks: []*clickup.Task{
			{
				ID:          "task1",
				Name:        "테스트 태스크",
				Description: "[재현 스텝]\n1. 앱 실행\n\n[오류내용]\n버그 발생",
				URL:         "https://clickup.com/task1",
			},
		},
	}

	mockInvoker := &MockInvoker{
		Result: &InvokeResult{
			WorkDir:   "/test",
			Prompt:    "test",
			StartedAt: time.Now().Format(time.RFC3339),
		},
	}

	config := WorkerConfig{
		ID:      "AI_01",
		ListID:  "list1",
		SrcPath: "/test/path",
	}

	worker := NewWorker(config, mockClient, mockInvoker, "작업중", "개발완료", "901413896178")

	// 태스크 처리
	ctx := context.Background()
	err := worker.ProcessTask(ctx, "task1")

	if err != nil {
		t.Fatalf("태스크 처리 실패: %v", err)
	}

	// 상태가 "작업중"으로 변경되었는지 확인
	if !mockClient.UpdateCalled {
		t.Error("UpdateTaskStatus가 호출되어야 함")
	}
	if len(mockClient.StatusUpdates) == 0 {
		t.Error("상태 업데이트가 있어야 함")
	}
	if mockClient.StatusUpdates[0].Status != "작업중" {
		t.Errorf("상태가 '작업중'이어야 함: got %s", mockClient.StatusUpdates[0].Status)
	}

	// Invoker가 호출되었는지 확인
	if !mockInvoker.InvokeCalled {
		t.Error("InvokePlan이 호출되어야 함")
	}
}

// TestWorker_State는 Worker 상태 관리를 테스트합니다.
func TestWorker_State(t *testing.T) {
	config := WorkerConfig{ID: "AI_01", ListID: "list1", SrcPath: "/test"}
	worker := NewWorker(config, nil, nil, "작업중", "개발완료", "")

	// 초기 상태
	if worker.IsProcessing() {
		t.Error("초기 상태는 processing이 아니어야 함")
	}

	// 처리 시작
	worker.SetProcessing("task1")
	if !worker.IsProcessing() {
		t.Error("SetProcessing 후 processing이어야 함")
	}
	if worker.GetCurrentTaskID() != "task1" {
		t.Error("현재 태스크 ID가 task1이어야 함")
	}

	// 처리 완료
	worker.ClearProcessing()
	if worker.IsProcessing() {
		t.Error("ClearProcessing 후 processing이 아니어야 함")
	}
}

// TestWorker_GetConfig는 Worker 설정 조회를 테스트합니다.
func TestWorker_GetConfig(t *testing.T) {
	config := WorkerConfig{
		ID:      "AI_02",
		ListID:  "list2",
		SrcPath: "/another/path",
	}
	worker := NewWorker(config, nil, nil, "작업중", "개발완료", "")

	if worker.GetConfig().ID != "AI_02" {
		t.Error("Worker ID 불일치")
	}
	if worker.GetConfig().ListID != "list2" {
		t.Error("ListID 불일치")
	}
}

// TestWorker_CompleteTask는 태스크 완료 처리를 테스트합니다.
func TestWorker_CompleteTask(t *testing.T) {
	mockClient := &MockClickUpClient{
		Tasks: []*clickup.Task{
			{ID: "task1", Name: "테스트 태스크"},
		},
	}

	config := WorkerConfig{ID: "AI_01", ListID: "list1", SrcPath: "/test"}
	worker := NewWorker(config, mockClient, nil, "작업중", "개발완료", "901413896178")

	// 태스크 처리 중 상태 설정
	worker.SetProcessing("task1")

	// 완료 처리
	ctx := context.Background()
	err := worker.CompleteTask(ctx)

	if err != nil {
		t.Fatalf("CompleteTask 실패: %v", err)
	}

	// 상태가 "개발완료"로 변경되었는지 확인
	if !mockClient.UpdateCalled {
		t.Error("UpdateTaskStatus가 호출되어야 함")
	}
	if len(mockClient.StatusUpdates) == 0 || mockClient.StatusUpdates[0].Status != "개발완료" {
		t.Error("상태가 '개발완료'로 변경되어야 함")
	}

	// 완료 리스트로 이동되었는지 확인
	if !mockClient.MoveTaskCalled {
		t.Error("MoveTaskToList가 호출되어야 함")
	}
	if len(mockClient.MovedTasks) == 0 {
		t.Error("이동된 태스크가 있어야 함")
	}
	if mockClient.MovedTasks[0].TaskID != "task1" {
		t.Errorf("태스크 ID 불일치: got %s", mockClient.MovedTasks[0].TaskID)
	}
	if mockClient.MovedTasks[0].ListID != "901413896178" {
		t.Errorf("목표 리스트 ID 불일치: got %s", mockClient.MovedTasks[0].ListID)
	}

	// 처리 상태가 클리어되었는지 확인
	if worker.IsProcessing() {
		t.Error("처리 상태가 클리어되어야 함")
	}
}

// TestWorker_CompleteTask_NoListMove는 완료 리스트 ID가 없을 때를 테스트합니다.
func TestWorker_CompleteTask_NoListMove(t *testing.T) {
	mockClient := &MockClickUpClient{
		Tasks: []*clickup.Task{
			{ID: "task1", Name: "테스트 태스크"},
		},
	}

	config := WorkerConfig{ID: "AI_01", ListID: "list1", SrcPath: "/test"}
	worker := NewWorker(config, mockClient, nil, "작업중", "개발완료", "") // 완료 리스트 ID 없음

	// 태스크 처리 중 상태 설정
	worker.SetProcessing("task1")

	// 완료 처리
	ctx := context.Background()
	err := worker.CompleteTask(ctx)

	if err != nil {
		t.Fatalf("CompleteTask 실패: %v", err)
	}

	// 상태 변경은 되어야 함
	if !mockClient.UpdateCalled {
		t.Error("UpdateTaskStatus가 호출되어야 함")
	}

	// 리스트 이동은 호출되지 않아야 함
	if mockClient.MoveTaskCalled {
		t.Error("completedListID가 없으면 MoveTaskToList가 호출되지 않아야 함")
	}
}
