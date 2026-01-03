package history

import (
	"testing"
)

// TestNewMemoryStore는 저장소 생성을 테스트합니다.
func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore(50)

	if store == nil {
		t.Fatal("저장소가 nil입니다")
	}
	if store.maxSize != 50 {
		t.Errorf("maxSize가 올바르지 않음: %d", store.maxSize)
	}
}

// TestNewMemoryStore_DefaultSize는 기본 크기 설정을 테스트합니다.
func TestNewMemoryStore_DefaultSize(t *testing.T) {
	store := NewMemoryStore(0)

	if store.maxSize != DefaultMaxSize {
		t.Errorf("기본 maxSize가 적용되지 않음: got %d, want %d", store.maxSize, DefaultMaxSize)
	}
}

// TestMemoryStore_Add는 레코드 추가를 테스트합니다.
func TestMemoryStore_Add(t *testing.T) {
	store := NewMemoryStore(100)

	record := &Record{
		SlackTimestamp: "1704153600.000001",
		ClickUpTaskID:  "task123",
		MessageText:    "테스트 메시지",
		Success:        true,
	}

	store.Add(record)

	if store.Count() != 1 {
		t.Errorf("레코드 수가 올바르지 않음: %d", store.Count())
	}
	if record.ID != 1 {
		t.Errorf("자동 ID가 할당되지 않음: %d", record.ID)
	}
	if record.CreatedAt.IsZero() {
		t.Error("CreatedAt이 자동 설정되지 않음")
	}
}

// TestMemoryStore_MaxSize는 최대 개수 초과 시 삭제를 테스트합니다.
func TestMemoryStore_MaxSize(t *testing.T) {
	maxSize := 5
	store := NewMemoryStore(maxSize)

	// maxSize + 2개 추가
	for i := 0; i < maxSize+2; i++ {
		store.Add(&Record{
			SlackTimestamp: "ts",
			MessageText:    "msg",
			Success:        true,
		})
	}

	// 최대 개수만 유지되어야 함
	if store.Count() != maxSize {
		t.Errorf("최대 개수 초과 시 삭제가 안됨: got %d, want %d", store.Count(), maxSize)
	}

	// 첫 번째 레코드는 삭제되어야 함 (ID 1, 2는 삭제됨)
	records := store.GetAll()
	if records[0].ID != 3 {
		t.Errorf("가장 오래된 레코드가 삭제되지 않음: 첫 ID = %d", records[0].ID)
	}
}

// TestMemoryStore_GetAll은 전체 조회를 테스트합니다.
func TestMemoryStore_GetAll(t *testing.T) {
	store := NewMemoryStore(100)

	store.Add(&Record{MessageText: "msg1", Success: true})
	store.Add(&Record{MessageText: "msg2", Success: true})

	records := store.GetAll()

	if len(records) != 2 {
		t.Errorf("레코드 수가 올바르지 않음: %d", len(records))
	}
}

// TestMemoryStore_GetRecent는 최근 N개 조회를 테스트합니다.
func TestMemoryStore_GetRecent(t *testing.T) {
	store := NewMemoryStore(100)

	store.Add(&Record{MessageText: "msg1", Success: true})
	store.Add(&Record{MessageText: "msg2", Success: true})
	store.Add(&Record{MessageText: "msg3", Success: true})

	// 최근 2개
	records := store.GetRecent(2)

	if len(records) != 2 {
		t.Errorf("레코드 수가 올바르지 않음: %d", len(records))
	}
	// 최신 순이므로 msg3가 첫 번째
	if records[0].MessageText != "msg3" {
		t.Errorf("최신 레코드가 첫 번째가 아님: %s", records[0].MessageText)
	}
	if records[1].MessageText != "msg2" {
		t.Errorf("두 번째 레코드가 올바르지 않음: %s", records[1].MessageText)
	}
}

// TestMemoryStore_Concurrency는 동시성 안전성을 테스트합니다.
func TestMemoryStore_Concurrency(t *testing.T) {
	store := NewMemoryStore(100)
	done := make(chan bool)

	// 여러 고루틴에서 동시에 추가
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				store.Add(&Record{MessageText: "test", Success: true})
			}
			done <- true
		}()
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < 10; i++ {
		<-done
	}

	// 최대 100개만 유지되어야 함
	if store.Count() != 100 {
		t.Errorf("동시성 처리 후 레코드 수: %d", store.Count())
	}
}
