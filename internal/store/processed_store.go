package store

import (
"database/sql"
"fmt"
"sync"
"time"

_ "github.com/mattn/go-sqlite3"
)

// ProcessedStore는 처리된 이메일 ID를 관리하는 저장소입니다.
type ProcessedStore interface {
	// IsProcessed는 해당 Message-ID가 이미 처리되었는지 확인합니다.
	IsProcessed(messageID string) (bool, error)
	// MarkProcessed는 Message-ID를 처리됨으로 표시합니다.
	MarkProcessed(messageID string, subject string) error
	// GetCount는 저장된 레코드 수를 반환합니다.
	GetCount() (int, error)
	// Cleanup은 오래된 레코드를 정리합니다 (retentionDays일 이전).
	Cleanup(retentionDays int) (int, error)
	// Close는 DB 연결을 닫습니다.
	Close() error
}

// SQLiteProcessedStore는 SQLite 기반 ProcessedStore 구현입니다.
type SQLiteProcessedStore struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// NewSQLiteProcessedStore는 새로운 SQLite 기반 저장소를 생성합니다.
func NewSQLiteProcessedStore(dbPath string) (*SQLiteProcessedStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	// 테이블 생성
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS processed_emails (
id INTEGER PRIMARY KEY AUTOINCREMENT,
message_id TEXT UNIQUE NOT NULL,
subject TEXT,
processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
	CREATE INDEX IF NOT EXISTS idx_message_id ON processed_emails(message_id);
	CREATE INDEX IF NOT EXISTS idx_processed_at ON processed_emails(processed_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("테이블 생성 실패: %w", err)
	}

	return &SQLiteProcessedStore{
		db:   db,
		path: dbPath,
	}, nil
}

// IsProcessed는 해당 Message-ID가 이미 처리되었는지 확인합니다.
func (s *SQLiteProcessedStore) IsProcessed(messageID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_emails WHERE message_id = ?", messageID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("조회 실패: %w", err)
	}

	return count > 0, nil
}

// MarkProcessed는 Message-ID를 처리됨으로 표시합니다.
func (s *SQLiteProcessedStore) MarkProcessed(messageID string, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
"INSERT OR IGNORE INTO processed_emails (message_id, subject, processed_at) VALUES (?, ?, ?)",
messageID, subject, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("삽입 실패: %w", err)
	}

	return nil
}

// GetCount는 저장된 레코드 수를 반환합니다.
func (s *SQLiteProcessedStore) GetCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_emails").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("카운트 조회 실패: %w", err)
	}

	return count, nil
}

// Cleanup은 오래된 레코드를 정리합니다.
func (s *SQLiteProcessedStore) Cleanup(retentionDays int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result, err := s.db.Exec("DELETE FROM processed_emails WHERE processed_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("정리 실패: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("삭제 개수 조회 실패: %w", err)
	}

	return int(deleted), nil
}

// Close는 DB 연결을 닫습니다.
func (s *SQLiteProcessedStore) Close() error {
	return s.db.Close()
}
