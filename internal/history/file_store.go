package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore는 파일 기반 히스토리 저장소입니다.
// ~/.slickwebhook/history.json에 저장됩니다.
type FileStore struct {
	filePath string
	records  []*Record
	maxSize  int
	counter  int64
	mu       sync.RWMutex
}

// NewFileStore는 새로운 파일 저장소를 생성합니다.
// filePath가 비어있으면 기본값 ~/.slickwebhook/history.json을 사용합니다.
func NewFileStore(filePath string, maxSize int) (*FileStore, error) {
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		filePath = filepath.Join(homeDir, ".slickwebhook", "history.json")
	}

	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	store := &FileStore{
		filePath: filePath,
		records:  make([]*Record, 0, maxSize),
		maxSize:  maxSize,
	}

	// 기존 파일이 있으면 로드
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// Add는 새 레코드를 추가하고 파일에 저장합니다.
func (s *FileStore) Add(record *Record) {
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
		s.records = s.records[1:]
	}

	s.records = append(s.records, record)

	// 파일에 저장 (비동기 아님, 데이터 안정성 우선)
	_ = s.saveUnsafe()
}

// GetAll은 모든 레코드를 반환합니다.
func (s *FileStore) GetAll() []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Record, len(s.records))
	copy(result, s.records)
	return result
}

// GetRecent는 최근 N개의 레코드를 반환합니다 (최신 순).
func (s *FileStore) GetRecent(n int) []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || len(s.records) == 0 {
		return []*Record{}
	}

	if n > len(s.records) {
		n = len(s.records)
	}

	result := make([]*Record, n)
	for i := 0; i < n; i++ {
		result[i] = s.records[len(s.records)-1-i]
	}
	return result
}

// Count는 전체 레코드 수를 반환합니다.
func (s *FileStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// FilePath는 저장 파일 경로를 반환합니다.
func (s *FileStore) FilePath() string {
	return s.filePath
}

// fileData는 파일에 저장되는 데이터 구조입니다.
type fileData struct {
	Counter int64     `json:"counter"`
	Records []*Record `json:"records"`
}

// load는 파일에서 데이터를 로드합니다.
func (s *FileStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return err
	}

	s.records = fd.Records
	s.counter = fd.Counter

	// maxSize 초과 레코드 정리
	if len(s.records) > s.maxSize {
		s.records = s.records[len(s.records)-s.maxSize:]
	}

	return nil
}

// saveUnsafe는 파일에 데이터를 저장합니다 (락 없음).
func (s *FileStore) saveUnsafe() error {
	// 디렉토리 생성
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fd := fileData{
		Counter: s.counter,
		Records: s.records,
	}

	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}
