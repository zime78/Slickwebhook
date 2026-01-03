package history

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewFileStore는 파일 저장소 생성을 테스트합니다.
func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_history.json")

	store, err := NewFileStore(filePath, 50)
	if err != nil {
		t.Fatalf("저장소 생성 실패: %v", err)
	}

	if store == nil {
		t.Fatal("저장소가 nil입니다")
	}
	if store.maxSize != 50 {
		t.Errorf("maxSize가 올바르지 않음: %d", store.maxSize)
	}
}

// TestFileStore_AddAndPersist는 추가 및 영속성을 테스트합니다.
func TestFileStore_AddAndPersist(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_history.json")

	// 첫 번째 저장소
	store1, err := NewFileStore(filePath, 100)
	if err != nil {
		t.Fatalf("저장소 생성 실패: %v", err)
	}

	store1.Add(&Record{
		SlackTimestamp: "123.456",
		MessageText:    "테스트 메시지",
		Success:        true,
	})

	if store1.Count() != 1 {
		t.Errorf("레코드 수가 올바르지 않음: %d", store1.Count())
	}

	// 파일이 생성되었는지 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("파일이 생성되지 않음")
	}

	// 두 번째 저장소 (같은 파일에서 로드)
	store2, err := NewFileStore(filePath, 100)
	if err != nil {
		t.Fatalf("저장소 재생성 실패: %v", err)
	}

	// 이전 데이터가 로드되었는지 확인
	if store2.Count() != 1 {
		t.Errorf("재로드 후 레코드 수가 올바르지 않음: %d", store2.Count())
	}

	records := store2.GetAll()
	if records[0].MessageText != "테스트 메시지" {
		t.Errorf("메시지 내용이 올바르지 않음: %s", records[0].MessageText)
	}
}

// TestFileStore_MaxSize는 최대 개수 제한을 테스트합니다.
func TestFileStore_MaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_history.json")

	store, _ := NewFileStore(filePath, 5)

	// 7개 추가 (5개만 유지되어야 함)
	for i := 0; i < 7; i++ {
		store.Add(&Record{MessageText: "msg", Success: true})
	}

	if store.Count() != 5 {
		t.Errorf("최대 개수 초과: got %d, want 5", store.Count())
	}

	// 첫 번째 레코드 ID는 3이어야 함 (1, 2는 삭제됨)
	records := store.GetAll()
	if records[0].ID != 3 {
		t.Errorf("가장 오래된 레코드 ID: got %d, want 3", records[0].ID)
	}
}

// TestFileStore_DefaultPath는 기본 경로를 테스트합니다.
func TestFileStore_DefaultPath(t *testing.T) {
	// 기본 경로로 생성 (실제 파일 생성은 안 함)
	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".slickwebhook", "history.json")

	store, err := NewFileStore("", 100)
	if err != nil {
		t.Fatalf("저장소 생성 실패: %v", err)
	}

	if store.FilePath() != expectedPath {
		t.Errorf("기본 경로가 올바르지 않음: %s", store.FilePath())
	}
}
