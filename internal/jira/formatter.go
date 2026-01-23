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
	// [오류내용], [수정요청]도 인식 (Jira에서 직접 사용되는 헤더)
	sectionPattern := regexp.MustCompile(`\[(재현 ?스텝|현 ?결과|오류 ?내용|기대 ?결과|수정 ?요청|추가 ?정보)\]`)

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

	// [오류내용] (기존 [현 결과] 또는 직접 [오류내용])
	errorContent := sections["현결과"]
	if errorContent == "" {
		errorContent = sections["오류내용"]
	}
	if errorContent != "" {
		result.WriteString("[오류내용]\n")
		result.WriteString(formatAsNumberedList(errorContent))
		// 첨부 이미지 추가
		if len(attachmentURLs) > 0 {
			result.WriteString(fmt.Sprintf("%d. 이미지 첨부\n", countNonEmptyLines(errorContent)+1))
		}
		result.WriteString("\n")
	}

	// [수정요청] (기존 [기대 결과] 또는 직접 [수정요청])
	fixContent := sections["기대결과"]
	if fixContent == "" {
		fixContent = sections["수정요청"]
	}
	if fixContent != "" {
		result.WriteString("[수정요청]\n")
		result.WriteString(formatAsNumberedList(fixContent))
		// 첨부 이미지 추가
		if len(attachmentURLs) > 1 {
			result.WriteString(fmt.Sprintf("%d. 이미지 첨부\n", countNonEmptyLines(fixContent)+1))
		}
		result.WriteString("\n")
	}

	// [추가 정보] - 번호 없이, 빈 줄 제거
	if content, ok := sections["추가정보"]; ok && content != "" {
		result.WriteString("[추가 정보]\n")
		result.WriteString(removeEmptyLines(content))
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

// removeEmptyLines는 빈 줄을 완전히 제거합니다.
func removeEmptyLines(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result.WriteString(trimmed)
			result.WriteString("\n")
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

// FilterMediaAttachments는 이미지와 동영상 첨부파일을 필터링합니다.
func FilterMediaAttachments(attachments []Attachment) []Attachment {
	var media []Attachment
	for _, att := range attachments {
		if strings.HasPrefix(att.MimeType, "image/") ||
			strings.HasPrefix(att.MimeType, "video/") {
			media = append(media, att)
		}
	}
	return media
}
