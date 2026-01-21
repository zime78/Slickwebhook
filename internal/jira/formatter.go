package jira

import (
	"fmt"
	"regexp"
	"strings"
)

// ReformatDescription은 Jira 본문을 재구성합니다.
// [현 결과] → [오류내용], [기대 결과] → [수정요청] 으로 변환
// 각 섹션에 번호를 매기고 빈 줄을 정리합니다.
func ReformatDescription(description string, attachmentURLs []string) string {
	if description == "" {
		return ""
	}

	// 섹션 분리를 위한 패턴
	sectionPattern := regexp.MustCompile(`\[(재현 ?스텝|현 ?결과|기대 ?결과|추가 ?정보)\]`)

	// 섹션별로 분리
	sections := make(map[string]string)
	currentSection := ""
	var currentContent strings.Builder

	lines := strings.Split(description, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 섹션 헤더 찾기
		if match := sectionPattern.FindString(trimmed); match != "" {
			// 이전 섹션 저장
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(currentContent.String())
			}
			// 새 섹션 시작
			currentSection = normalizeSection(match)
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

	// 결과 조합
	var result strings.Builder

	// [재현 스텝]
	if content, ok := sections["재현스텝"]; ok && content != "" {
		result.WriteString("[재현 스텝]\n")
		result.WriteString(formatAsNumberedList(content))
		result.WriteString("\n")
	}

	// [오류내용] (기존 [현 결과])
	if content, ok := sections["현결과"]; ok && content != "" {
		result.WriteString("[오류내용]\n")
		result.WriteString(formatAsNumberedList(content))
		// 첨부 이미지 추가
		if len(attachmentURLs) > 0 {
			result.WriteString(fmt.Sprintf("%d. 이미지 첨부\n", countNonEmptyLines(content)+1))
		}
		result.WriteString("\n")
	}

	// [수정요청] (기존 [기대 결과])
	if content, ok := sections["기대결과"]; ok && content != "" {
		result.WriteString("[수정요청]\n")
		result.WriteString(formatAsNumberedList(content))
		// 첨부 이미지 추가
		if len(attachmentURLs) > 1 {
			result.WriteString(fmt.Sprintf("%d. 이미지 첨부\n", countNonEmptyLines(content)+1))
		}
		result.WriteString("\n")
	}

	// [추가 정보] - 번호 없이 그대로
	if content, ok := sections["추가정보"]; ok && content != "" {
		result.WriteString("[추가 정보]\n")
		result.WriteString(collapseEmptyLines(content))
		result.WriteString("\n")
	}

	formatted := strings.TrimSpace(result.String())
	if formatted == "" {
		// 섹션 파싱 실패 시 원본 반환
		return description
	}

	return formatted
}

// formatAsNumberedList는 내용을 번호 있는 목록으로 변환합니다.
func formatAsNumberedList(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	lineNum := 1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // 빈 줄은 스킵
		}

		// 이미 번호가 있으면 그대로 사용
		if matched, _ := regexp.MatchString(`^\d+\.`, trimmed); matched {
			result.WriteString(trimmed)
		} else {
			result.WriteString(fmt.Sprintf("%d. %s", lineNum, trimmed))
		}
		result.WriteString("\n")
		lineNum++
	}

	return result.String()
}

// countNonEmptyLines는 비어있지 않은 줄 수를 계산합니다.
func countNonEmptyLines(content string) int {
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// collapseEmptyLines는 연속된 빈 줄을 하나로 합칩니다.
func collapseEmptyLines(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	prevEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !prevEmpty {
				result.WriteString("\n")
				prevEmpty = true
			}
		} else {
			result.WriteString(trimmed)
			result.WriteString("\n")
			prevEmpty = false
		}
	}

	return strings.TrimSpace(result.String())
}

// normalizeSection은 섹션 이름을 정규화합니다.
func normalizeSection(section string) string {
	// 공백 제거하고 정규화
	section = strings.ReplaceAll(section, " ", "")
	section = strings.Trim(section, "[]")
	return section
}

// FilterImageAttachments는 이미지 첨부파일만 필터링합니다.
func FilterImageAttachments(attachments []Attachment) []Attachment {
	var images []Attachment
	for _, att := range attachments {
		if strings.HasPrefix(att.MimeType, "image/") {
			images = append(images, att)
		}
	}
	return images
}
