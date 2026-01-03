package history

import (
	"sync"
	"time"
)

// Record는 이벤트 전송 히스토리 레코드입니다.
type Record struct {
	// ID는 레코드 고유 ID입니다
	ID int64
	// SlackTimestamp는 Slack 메시지 타임스탬프입니다
	SlackTimestamp string
	// ClickUpTaskID는 생성된 ClickUp 태스크 ID입니다
	ClickUpTaskID string
	// ClickUpTaskURL는 태스크 URL입니다
	ClickUpTaskURL string
	// MessageText는 원본 메시지 텍스트 (요약)입니다
	MessageText string
	// CreatedAt는 레코드 생성 시간입니다
	CreatedAt time.Time
	// Success는 전송 성공 여부입니다
	Success bool
	// ErrorMessage는 실패 시 에러 메시지입니다
	ErrorMessage string
}

// Store는 히스토리 저장소 인터페이스입니다.
type Store interface {
	// Add는 새 레코드를 추가합니다.
	Add(record *Record)
	// GetAll은 모든 레코드를 반환합니다.
	GetAll() []*Record
	// GetRecent는 최근 N개의 레코드를 반환합니다.
	GetRecent(n int) []*Record
	// Count는 전체 레코드 수를 반환합니다.
	Count() int
}

// MemoryStore는 메모리 기반 히스토리 저장소입니다.
type MemoryStore struct {
	records []*Record
	maxSize int
	counter int64
	mu      sync.RWMutex
}

// DefaultMaxSize는 기본 최대 저장 개수입니다.
const DefaultMaxSize = 100

// NewMemoryStore는 새로운 메모리 저장소를 생성합니다.
func NewMemoryStore(maxSize int) *MemoryStore {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &MemoryStore{
		records: make([]*Record, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add는 새 레코드를 추가합니다.
// 최대 개수를 초과하면 가장 오래된 레코드를 삭제합니다.
func (s *MemoryStore) Add(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ID 자동 할당
	s.counter++
	record.ID = s.counter

	// 생성 시간 자동 설정
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	// 최대 개수 초과 시 가장 오래된 레코드 삭제
	if len(s.records) >= s.maxSize {
		s.records = s.records[1:] // 첫 번째 요소 제거
	}

	s.records = append(s.records, record)
}

// GetAll은 모든 레코드를 반환합니다.
func (s *MemoryStore) GetAll() []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 복사본 반환
	result := make([]*Record, len(s.records))
	copy(result, s.records)
	return result
}

// GetRecent는 최근 N개의 레코드를 반환합니다 (최신 순).
func (s *MemoryStore) GetRecent(n int) []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || len(s.records) == 0 {
		return []*Record{}
	}

	if n > len(s.records) {
		n = len(s.records)
	}

	// 최신 순으로 반환
	result := make([]*Record, n)
	for i := 0; i < n; i++ {
		result[i] = s.records[len(s.records)-1-i]
	}
	return result
}

// Count는 전체 레코드 수를 반환합니다.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
