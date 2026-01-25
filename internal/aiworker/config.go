package aiworker

// TerminalType은 사용할 터미널 종류입니다.
type TerminalType string

const (
	TerminalTypeDefault TerminalType = "terminal" // macOS 기본 터미널
	TerminalTypeWarp    TerminalType = "warp"     // Warp 터미널
)

// Config는 AI Worker 전체 설정입니다.
type Config struct {
	Workers         []WorkerConfig // 각 Worker별 설정
	StatusWorking   string         // 작업중 상태명 (기본: "작업중")
	StatusCompleted string         // 완료 상태명 (기본: "개발완료")
	CompletedListID string         // 완료된 태스크 이동 목표 리스트 ID
	HookServerPort  int            // Hook 서버 포트 (기본: 8081)
	WebhookPort     int            // Webhook 서버 포트 (기본: 8080)
	SlackChannel    string         // Slack 알림 채널 ID
	TerminalType    TerminalType   // 터미널 종류 (기본: "terminal")
}

// WorkerConfig는 개별 Worker 설정입니다.
type WorkerConfig struct {
	ID      string // Worker ID (예: "AI_01")
	ListID  string // ClickUp 리스트 ID
	SrcPath string // Claude Code 실행 경로
}

// DefaultConfig는 기본 설정을 반환합니다.
func DefaultConfig() Config {
	return Config{
		Workers:         make([]WorkerConfig, 0),
		StatusWorking:   "작업중",
		StatusCompleted: "개발완료",
		HookServerPort:  8081,
		WebhookPort:     8080,
		TerminalType:    TerminalTypeDefault,
	}
}

// AddWorker는 Worker 설정을 추가합니다.
func (c *Config) AddWorker(id, listID, srcPath string) {
	c.Workers = append(c.Workers, WorkerConfig{
		ID:      id,
		ListID:  listID,
		SrcPath: srcPath,
	})
}

// GetWorkerByListID는 리스트 ID로 Worker 설정을 찾습니다.
func (c *Config) GetWorkerByListID(listID string) *WorkerConfig {
	for i := range c.Workers {
		if c.Workers[i].ListID == listID {
			return &c.Workers[i]
		}
	}
	return nil
}

// GetWorkerBySrcPath는 소스 경로로 Worker 설정을 찾습니다.
func (c *Config) GetWorkerBySrcPath(srcPath string) *WorkerConfig {
	for i := range c.Workers {
		if c.Workers[i].SrcPath == srcPath {
			return &c.Workers[i]
		}
	}
	return nil
}

// GetAllListIDs는 모든 리스트 ID 목록을 반환합니다.
func (c *Config) GetAllListIDs() []string {
	ids := make([]string, len(c.Workers))
	for i, w := range c.Workers {
		ids[i] = w.ListID
	}
	return ids
}
