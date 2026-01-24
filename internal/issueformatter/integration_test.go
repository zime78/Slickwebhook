package issueformatter

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/zime/slickwebhook/internal/clickup"
)

// TestIntegration_RealAPI는 실제 ClickUp API를 사용한 통합 테스트입니다.
// 실행: CLICKUP_API_TOKEN=pk_xxx go test ./internal/issueformatter/... -v -run TestIntegration
func TestIntegration_RealAPI(t *testing.T) {
	// API 토큰이 없으면 스킵
	apiToken := os.Getenv("CLICKUP_API_TOKEN")
	if apiToken == "" {
		t.Skip("CLICKUP_API_TOKEN 환경변수가 설정되지 않아 통합 테스트를 건너뜁니다")
	}

	// ClickUp 클라이언트 생성
	clickupConfig := clickup.Config{
		APIToken: apiToken,
		ListID:   "901413896178",
	}
	client := clickup.NewClickUpClient(clickupConfig)

	// 태스크 조회 (샘플 태스크 ID)
	taskID := "86b88u78r"
	ctx := context.Background()

	task, err := client.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("태스크 조회 실패: %v", err)
	}

	t.Logf("태스크 조회 성공: %s", task.Name)
	t.Logf("첨부파일 수: %d", len(task.Attachments))

	// 포맷터 설정
	config := Config{
		OutputDir:     "/tmp/issueformatter_test",
		MaxFrames:     3,
		FrameInterval: 5,
	}

	// 포맷터 생성 및 실행
	formatter := NewIssueFormatter(config)
	prompt, err := formatter.Format(ctx, task)
	if err != nil {
		t.Fatalf("포맷팅 실패: %v", err)
	}

	// 결과 출력
	t.Log("\n========== AI 전달용 텍스트 ==========")
	t.Log(prompt.Text)

	t.Log("\n========== 이미지 경로 ==========")
	for i, path := range prompt.ImagePaths {
		t.Logf("%d. %s", i+1, path)
	}

	// 검증
	if prompt.Text == "" {
		t.Error("텍스트가 비어있습니다")
	}

	if len(prompt.ImagePaths) == 0 {
		t.Error("이미지 경로가 비어있습니다")
	}
}

// TestIntegration_PrintOutput은 결과를 콘솔에 출력합니다.
// 실행: CLICKUP_API_TOKEN=pk_xxx go test ./internal/issueformatter/... -v -run TestIntegration_PrintOutput
func TestIntegration_PrintOutput(t *testing.T) {
	apiToken := os.Getenv("CLICKUP_API_TOKEN")
	if apiToken == "" {
		t.Skip("CLICKUP_API_TOKEN 환경변수가 설정되지 않아 통합 테스트를 건너뜁니다")
	}

	clickupConfig := clickup.Config{
		APIToken: apiToken,
		ListID:   "901413896178",
	}
	client := clickup.NewClickUpClient(clickupConfig)

	taskID := "86b88u78r"
	ctx := context.Background()

	task, err := client.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("태스크 조회 실패: %v", err)
	}

	config := Config{
		OutputDir:     "/tmp/issueformatter_test",
		MaxFrames:     3,
		FrameInterval: 5,
	}

	formatter := NewIssueFormatter(config)
	prompt, err := formatter.Format(ctx, task)
	if err != nil {
		t.Fatalf("포맷팅 실패: %v", err)
	}

	// 최종 결과 출력
	fmt.Println("\n" + prompt.Text)
	fmt.Println("\n이미지 경로:")
	for _, path := range prompt.ImagePaths {
		fmt.Printf("  - %s\n", path)
	}
}
