package store

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// JiraIssueStore는 처리된 Jira 이슈 키를 관리하는 저장소입니다.
// 동일한 이슈에 대해 ClickUp/Slack으로 중복 전송을 방지합니다.
type JiraIssueStore interface {
	// IsProcessed는 해당 이슈 키가 이미 처리되었는지 확인합니다.
	IsProcessed(issueKey string) (bool, error)
	// MarkProcessed는 이슈 키를 처리됨으로 표시합니다.
	MarkProcessed(issueKey string, summary string) error
	// GetCount는 저장된 레코드 수를 반환합니다.
	GetCount() (int, error)
	// Cleanup은 오래된 레코드를 정리합니다 (retentionDays일 이전).
	Cleanup(retentionDays int) (int, error)
	// Close는 DB 연결을 닫습니다.
	Close() error
}

// SQLiteJiraIssueStore는 SQLite 기반 JiraIssueStore 구현입니다.
type SQLiteJiraIssueStore struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// NewSQLiteJiraIssueStore는 새로운 SQLite 기반 Jira 이슈 저장소를 생성합니다.
func NewSQLiteJiraIssueStore(dbPath string) (*SQLiteJiraIssueStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	// 테이블 생성
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS processed_jira_issues (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		issue_key TEXT UNIQUE NOT NULL,
		summary TEXT,
		processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_issue_key ON processed_jira_issues(issue_key);
	CREATE INDEX IF NOT EXISTS idx_jira_processed_at ON processed_jira_issues(processed_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("테이블 생성 실패: %w", err)
	}

	return &SQLiteJiraIssueStore{
		db:   db,
		path: dbPath,
	}, nil
}

// IsProcessed는 해당 이슈 키가 이미 처리되었는지 확인합니다.
func (s *SQLiteJiraIssueStore) IsProcessed(issueKey string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_jira_issues WHERE issue_key = ?", issueKey).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("조회 실패: %w", err)
	}

	return count > 0, nil
}

// MarkProcessed는 이슈 키를 처리됨으로 표시합니다.
func (s *SQLiteJiraIssueStore) MarkProcessed(issueKey string, summary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO processed_jira_issues (issue_key, summary, processed_at) VALUES (?, ?, ?)",
		issueKey, summary, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("삽입 실패: %w", err)
	}

	return nil
}

// GetCount는 저장된 레코드 수를 반환합니다.
func (s *SQLiteJiraIssueStore) GetCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_jira_issues").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("카운트 조회 실패: %w", err)
	}

	return count, nil
}

// Cleanup은 오래된 레코드를 정리합니다.
func (s *SQLiteJiraIssueStore) Cleanup(retentionDays int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result, err := s.db.Exec("DELETE FROM processed_jira_issues WHERE processed_at < ?", cutoff)
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
func (s *SQLiteJiraIssueStore) Close() error {
	return s.db.Close()
}
