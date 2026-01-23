package issueformatter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zime/slickwebhook/internal/clickup"
)

// IssueFormatter는 ClickUp 이슈를 AI 프롬프트로 변환합니다.
type IssueFormatter struct {
	config    Config
	processor MediaProcessor
}

// NewIssueFormatter는 새 포맷터를 생성합니다.
func NewIssueFormatter(config Config) *IssueFormatter {
	return &IssueFormatter{
		config:    config,
		processor: NewMediaProcessor(),
	}
}

// NewIssueFormatterWithProcessor는 커스텀 프로세서로 포맷터를 생성합니다.
func NewIssueFormatterWithProcessor(config Config, processor MediaProcessor) *IssueFormatter {
	return &IssueFormatter{
		config:    config,
		processor: processor,
	}
}

// Format은 태스크를 AI 프롬프트로 변환합니다.
func (f *IssueFormatter) Format(ctx context.Context, task *clickup.Task) (*AIPrompt, error) {
	if task == nil {
		return nil, fmt.Errorf("태스크가 nil입니다")
	}

	// 출력 디렉토리 생성
	taskDir := filepath.Join(f.config.OutputDir, fmt.Sprintf("issue_%s", task.ID))
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// 미디어 파일 처리
	imagePaths, err := f.processAttachments(ctx, task.Attachments, taskDir)
	if err != nil {
		return nil, fmt.Errorf("첨부파일 처리 실패: %w", err)
	}

	// 마크다운 텍스트 생성
	text := f.generateMarkdown(task, imagePaths)

	return &AIPrompt{
		Text:       text,
		ImagePaths: imagePaths,
	}, nil
}

// processAttachments는 첨부파일을 처리합니다.
func (f *IssueFormatter) processAttachments(ctx context.Context, attachments []clickup.Attachment, outputDir string) ([]string, error) {
	var imagePaths []string

	for i, att := range attachments {
		if IsImageFile(att.Title) {
			// 이미지 다운로드
			outputPath := filepath.Join(outputDir, fmt.Sprintf("image_%03d%s", i+1, filepath.Ext(att.Title)))
			if err := f.processor.DownloadImage(ctx, att.URL, outputPath); err != nil {
				return nil, fmt.Errorf("이미지 다운로드 실패 (%s): %w", att.Title, err)
			}
			imagePaths = append(imagePaths, outputPath)
		} else if IsVideoFile(att.Title) {
			// 동영상 프레임 추출
			frameDir := filepath.Join(outputDir, fmt.Sprintf("video_%03d_frames", i+1))
			frames, err := f.processor.ExtractVideoFrames(ctx, att.URL, frameDir, f.config.FrameInterval)
			if err != nil {
				return nil, fmt.Errorf("동영상 프레임 추출 실패 (%s): %w", att.Title, err)
			}
			// 최대 프레임 수 제한
			if len(frames) > f.config.MaxFrames {
				frames = frames[:f.config.MaxFrames]
			}
			imagePaths = append(imagePaths, frames...)
		}
	}

	return imagePaths, nil
}

// generateMarkdown은 마크다운 텍스트를 생성합니다.
func (f *IssueFormatter) generateMarkdown(task *clickup.Task, imagePaths []string) string {
	var sb strings.Builder

	// 제목 (태스크 이름에서 핵심만 추출)
	title := extractTitle(task.Name)
	sb.WriteString(fmt.Sprintf("# 버그: %s\n\n", title))

	// 설명에서 섹션 추출
	sections := parseDescription(task.Description)

	// 현재 동작
	if current, ok := sections["오류내용"]; ok {
		sb.WriteString("## 현재 동작\n")
		sb.WriteString(current)
		sb.WriteString("\n\n")
	}

	// 기대 동작
	if expected, ok := sections["수정요청"]; ok {
		sb.WriteString("## 기대 동작\n")
		sb.WriteString(expected)
		sb.WriteString("\n\n")
	}

	// 재현 스텝
	if steps, ok := sections["재현 스텝"]; ok {
		sb.WriteString("## 재현 스텝\n")
		sb.WriteString(steps)
		sb.WriteString("\n\n")
	}

	// 수정 포인트 (기대 동작 기반으로 요약)
	sb.WriteString("## 수정 포인트\n")
	sb.WriteString(extractFixPoint(task.Description))
	sb.WriteString("\n\n")

	// 첨부 이미지
	if len(imagePaths) > 0 {
		sb.WriteString("## 첨부 이미지\n")
		for i, path := range imagePaths {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, path))
		}
	}

	return sb.String()
}

// extractTitle은 태스크 이름에서 핵심 제목을 추출합니다.
func extractTitle(name string) string {
	// [프로젝트][플랫폼] 형식 제거
	title := name

	// ITSM-XXXX 제거
	if idx := strings.Index(title, " "); idx > 0 {
		prefix := title[:idx]
		if strings.HasPrefix(prefix, "ITSM-") || strings.HasPrefix(prefix, "JIRA-") {
			title = strings.TrimSpace(title[idx:])
		}
	}

	// [태그] 제거
	for strings.HasPrefix(title, "[") {
		if idx := strings.Index(title, "]"); idx > 0 {
			title = strings.TrimSpace(title[idx+1:])
		} else {
			break
		}
	}

	// 50자로 제한
	if len([]rune(title)) > 50 {
		title = string([]rune(title)[:50]) + "..."
	}

	return title
}

// parseDescription은 설명에서 섹션을 파싱합니다.
func parseDescription(description string) map[string]string {
	sections := make(map[string]string)

	lines := strings.Split(description, "\n")
	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 섹션 헤더 확인
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// 이전 섹션 저장
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(currentContent.String())
			}
			// 새 섹션 시작
			currentSection = strings.Trim(trimmed, "[]")
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// 마지막 섹션 저장
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(currentContent.String())
	}

	return sections
}

// extractFixPoint는 설명에서 수정 포인트를 추출합니다.
func extractFixPoint(description string) string {
	// 수정요청 섹션에서 첫 번째 항목 추출
	sections := parseDescription(description)
	if fix, ok := sections["수정요청"]; ok {
		lines := strings.Split(fix, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "1.") {
				return strings.TrimPrefix(trimmed, "1.")
			}
		}
		// 첫 줄 반환
		if len(lines) > 0 {
			return lines[0]
		}
	}
	return "이슈 설명 참조"
}
