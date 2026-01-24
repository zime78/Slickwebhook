package aiworker

import (
	"testing"
)

// TestManager_Creation은 Manager 생성을 테스트합니다.
func TestManager_Creation(t *testing.T) {
	config := DefaultConfig()
	config.AddWorker("AI_01", "list1", "/path1")
	config.AddWorker("AI_02", "list2", "/path2")

	manager := NewManager(config)

	if manager == nil {
		t.Fatal("Manager가 nil입니다")
	}

	if len(manager.GetWorkers()) != 2 {
		t.Errorf("Worker 개수 불일치: got %d, want 2", len(manager.GetWorkers()))
	}
}

// TestManager_GetWorkerByListID는 리스트 ID로 Worker 조회를 테스트합니다.
func TestManager_GetWorkerByListID(t *testing.T) {
	config := DefaultConfig()
	config.AddWorker("AI_01", "list1", "/path1")
	config.AddWorker("AI_02", "list2", "/path2")

	manager := NewManager(config)

	worker := manager.GetWorkerByListID("list1")
	if worker == nil {
		t.Fatal("Worker를 찾을 수 없음")
	}
	if worker.GetConfig().ID != "AI_01" {
		t.Errorf("Worker ID 불일치: got %s, want AI_01", worker.GetConfig().ID)
	}

	// 존재하지 않는 리스트
	notFound := manager.GetWorkerByListID("invalid")
	if notFound != nil {
		t.Error("존재하지 않는 리스트에서 nil이 반환되어야 함")
	}
}

// TestManager_GetWorkerBySrcPath는 소스 경로로 Worker 조회를 테스트합니다.
func TestManager_GetWorkerBySrcPath(t *testing.T) {
	config := DefaultConfig()
	config.AddWorker("AI_01", "list1", "/path1")
	config.AddWorker("AI_02", "list2", "/path2")

	manager := NewManager(config)

	worker := manager.GetWorkerBySrcPath("/path2")
	if worker == nil {
		t.Fatal("Worker를 찾을 수 없음")
	}
	if worker.GetConfig().ID != "AI_02" {
		t.Errorf("Worker ID 불일치: got %s, want AI_02", worker.GetConfig().ID)
	}
}

// TestManager_IsAIList는 AI 리스트 여부 확인을 테스트합니다.
func TestManager_IsAIList(t *testing.T) {
	config := DefaultConfig()
	config.AddWorker("AI_01", "list1", "/path1")
	config.AddWorker("AI_02", "list2", "/path2")

	manager := NewManager(config)

	if !manager.IsAIList("list1") {
		t.Error("list1은 AI 리스트여야 함")
	}
	if !manager.IsAIList("list2") {
		t.Error("list2는 AI 리스트여야 함")
	}
	if manager.IsAIList("list3") {
		t.Error("list3는 AI 리스트가 아니어야 함")
	}
}

// TestManager_AllIdle는 모든 Worker가 유휴 상태인지 확인을 테스트합니다.
func TestManager_AllIdle(t *testing.T) {
	config := DefaultConfig()
	config.AddWorker("AI_01", "list1", "/path1")

	manager := NewManager(config)

	// 초기 상태는 모두 유휴
	if !manager.AllIdle() {
		t.Error("초기 상태는 AllIdle이어야 함")
	}

	// Worker를 busy로 설정
	workers := manager.GetWorkers()
	workers[0].SetProcessing("task1")

	if manager.AllIdle() {
		t.Error("처리 중일 때 AllIdle이 아니어야 함")
	}

	// 다시 유휴로
	workers[0].ClearProcessing()
	if !manager.AllIdle() {
		t.Error("처리 완료 후 AllIdle이어야 함")
	}
}
