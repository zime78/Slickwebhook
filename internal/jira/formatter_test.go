package jira

import (
	"strings"
	"testing"
)

func TestReformatDescription_AllSections(t *testing.T) {
	// 모든 섹션이 있는 정상 케이스
	input := `[재현 스텝]
1. Q App 실행 > 이메일/SNS 계정 로그인
2. [전체메뉴] > [내 정보 수정하기] 선택
3. 프로필 > [탈퇴하기] 선택

[현 결과]
내 정보 > 탈퇴하기 시 계정 인증 절차 없이 탈퇴 처리되고 있습니다.

[기대 결과]
내 정보 > 탈퇴하기 시 계정 인증 절차 이후 인증 성공 시 탈퇴 처리되도록 수정을 요청드립니다.

[추가 정보]
테스트 환경: QA`

	attachments := []Attachment{
		{Filename: "image1.png", MimeType: "image/png", Content: "https://example.com/image1.png"},
		{Filename: "image2.png", MimeType: "image/png", Content: "https://example.com/image2.png"},
	}
	result := ReformatDescription(input, attachments)

	// 섹션 라벨 변환 확인
	if !strings.Contains(result, "[재현 스텝]") {
		t.Error("[재현 스텝] 섹션이 있어야 함")
	}
	if !strings.Contains(result, "[오류내용]") {
		t.Error("[현 결과]가 [오류내용]으로 변환되어야 함")
	}
	if !strings.Contains(result, "[수정요청]") {
		t.Error("[기대 결과]가 [수정요청]으로 변환되어야 함")
	}
	if !strings.Contains(result, "[추가 정보]") {
		t.Error("[추가 정보] 섹션이 있어야 함")
	}

	// 실제 파일명 표시 확인
	if !strings.Contains(result, "이미지: image1.png") {
		t.Error("이미지 파일명이 표시되어야 함")
	}
	if !strings.Contains(result, "이미지: image2.png") {
		t.Error("두 번째 이미지 파일명이 표시되어야 함")
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_NoImages(t *testing.T) {
	// 이미지 없는 케이스
	input := `[재현 스텝]
1. 앱 실행

[현 결과]
오류 발생

[기대 결과]
정상 동작`

	result := ReformatDescription(input, nil) // 이미지 없음

	// 이미지 안내가 없어야 함
	if strings.Contains(result, "이미지 첨부") {
		t.Error("이미지가 없으면 이미지 안내도 없어야 함")
	}

	// 섹션 변환은 정상 동작해야 함
	if !strings.Contains(result, "[오류내용]") {
		t.Error("[현 결과]가 [오류내용]으로 변환되어야 함")
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_InvalidFormat(t *testing.T) {
	// 형식에 맞지 않는 케이스 - 섹션 헤더가 없음
	input := `이것은 그냥 일반 텍스트입니다.
특별한 형식이 없습니다.
그냥 본문만 있습니다.`

	result := ReformatDescription(input, nil)

	// 원본 그대로 반환되어야 함
	if result != input {
		t.Errorf("형식에 맞지 않으면 원본 반환해야 함.\n원본: %s\n결과: %s", input, result)
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_PartialSections(t *testing.T) {
	// 일부 섹션만 있는 케이스
	input := `[재현 스텝]
1. 앱 실행
2. 버튼 클릭

[현 결과]
에러 발생`
	// [기대 결과]와 [추가 정보]가 없음

	result := ReformatDescription(input, nil)

	// 있는 섹션만 변환
	if !strings.Contains(result, "[재현 스텝]") {
		t.Error("[재현 스텝] 있어야 함")
	}
	if !strings.Contains(result, "[오류내용]") {
		t.Error("[오류내용]으로 변환되어야 함")
	}

	// 없는 섹션은 결과에도 없어야 함
	if strings.Contains(result, "[수정요청]") {
		t.Error("[수정요청] 없어야 함 (원본에 [기대 결과] 없음)")
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_EmptyInput(t *testing.T) {
	// 빈 문자열 케이스
	result := ReformatDescription("", nil)

	if result != "" {
		t.Errorf("빈 입력은 빈 결과를 반환해야 함: %s", result)
	}
}

func TestReformatDescription_OneImage(t *testing.T) {
	// 이미지 1개만 있는 케이스
	input := `[현 결과]
오류 화면

[기대 결과]
정상 화면`

	attachments := []Attachment{
		{Filename: "image.png", MimeType: "image/png", Content: "https://example.com/image.png"},
	}
	result := ReformatDescription(input, attachments)

	// [오류내용]에만 파일명 안내가 있어야 함
	count := strings.Count(result, "이미지: image.png")
	if count != 1 {
		t.Errorf("이미지 1개면 안내도 1개: got %d", count)
	}

	t.Logf("결과:\n%s", result)
}

func TestFilterMediaAttachments(t *testing.T) {
	attachments := []Attachment{
		{ID: "1", Filename: "image.png", MimeType: "image/png"},
		{ID: "2", Filename: "doc.pdf", MimeType: "application/pdf"},
		{ID: "3", Filename: "photo.jpg", MimeType: "image/jpeg"},
		{ID: "4", Filename: "video.mp4", MimeType: "video/mp4"},
		{ID: "5", Filename: "movie.mov", MimeType: "video/quicktime"},
	}

	media := FilterMediaAttachments(attachments)

	// 이미지 2개 + 동영상 2개 = 4개
	if len(media) != 4 {
		t.Errorf("미디어 4개여야 함: got %d", len(media))
	}

	for _, m := range media {
		if !strings.HasPrefix(m.MimeType, "image/") && !strings.HasPrefix(m.MimeType, "video/") {
			t.Errorf("이미지나 동영상이 아닌 파일 포함됨: %s", m.MimeType)
		}
	}
}

func TestReformatDescription_DirectHeaders(t *testing.T) {
	// Jira에서 [오류내용], [수정요청] 헤더를 직접 사용하는 케이스
	input := `[오류내용]
1. AOS > 내정보 > 프로필 > 이메일 계정 로그인 시, 이메일 영역이 미노출됩니다.

[수정요청]
1. 이메일 로그인 > 내 정보 > 프로필 진입시 이메일 영역 노출되도록 수정을 요청드립니다
2. [Q 글로벌 -APP] 내정보 / 5P

[추가 정보]
테스트 환경
서버 환경 : QA
테스트 디바이스 : Galaxy Z flip 5(16)
테스트 버전 : AOS 2.0.0(2)
테스트 계정 : ianbyun02@gmail.com`

	attachments := []Attachment{
		{Filename: "image.png", MimeType: "image/png", Content: "https://example.com/image.png"},
	}
	result := ReformatDescription(input, attachments)

	// [오류내용] 섹션이 있어야 함
	if !strings.Contains(result, "[오류내용]") {
		t.Error("[오류내용] 섹션이 있어야 함")
	}

	// [수정요청] 섹션이 있어야 함
	if !strings.Contains(result, "[수정요청]") {
		t.Error("[수정요청] 섹션이 있어야 함")
	}

	// [추가 정보] 섹션이 있어야 함
	if !strings.Contains(result, "[추가 정보]") {
		t.Error("[추가 정보] 섹션이 있어야 함")
	}

	// 내용이 제대로 파싱되었는지 확인
	if !strings.Contains(result, "이메일 영역이 미노출") {
		t.Error("오류내용 내용이 포함되어야 함")
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_VideoAttachment(t *testing.T) {
	// 동영상 첨부파일이 "동영상: filename"으로 표시되는지 확인
	input := `[현 결과]
오류 화면

[기대 결과]
정상 화면`

	attachments := []Attachment{
		{Filename: "screen_recording.mp4", MimeType: "video/mp4", Content: "https://example.com/video.mp4"},
		{Filename: "expected.png", MimeType: "image/png", Content: "https://example.com/image.png"},
	}
	result := ReformatDescription(input, attachments)

	if !strings.Contains(result, "동영상: screen_recording.mp4") {
		t.Error("동영상 파일명이 표시되어야 함")
	}
	if !strings.Contains(result, "이미지: expected.png") {
		t.Error("이미지 파일명이 표시되어야 함")
	}

	t.Logf("결과:\n%s", result)
}

func TestReformatDescription_MixedAttachments(t *testing.T) {
	// 이미지 + 동영상 혼합 시 각각 올바른 라벨 표시 확인
	input := `[현 결과]
설정 값이 서로 연동됩니다

[기대 결과]
각각 설정되어야 합니다`

	attachments := []Attachment{
		{Filename: "image-20260128.png", MimeType: "image/png", Content: "https://example.com/img.png"},
		{Filename: "스크린트레이닝.mp4", MimeType: "video/mp4", Content: "https://example.com/v1.mp4"},
		{Filename: "Screen_Recording.mp4", MimeType: "video/mp4", Content: "https://example.com/v2.mp4"},
	}
	result := ReformatDescription(input, attachments)

	// [오류내용]에 모든 첨부파일 표시
	if !strings.Contains(result, "이미지: image-20260128.png") {
		t.Error("이미지 파일명이 [오류내용]에 표시되어야 함")
	}
	if !strings.Contains(result, "동영상: 스크린트레이닝.mp4") {
		t.Error("동영상 파일명이 [오류내용]에 표시되어야 함")
	}
	if !strings.Contains(result, "동영상: Screen_Recording.mp4") {
		t.Error("두 번째 동영상 파일명이 [오류내용]에 표시되어야 함")
	}

	// [수정요청]에도 두 번째 이후 첨부파일 표시
	if !strings.Contains(result, "[수정요청]") {
		t.Error("[수정요청] 섹션이 있어야 함")
	}

	t.Logf("결과:\n%s", result)
}

func TestFormatAttachmentLabel(t *testing.T) {
	tests := []struct {
		name string
		att  Attachment
		want string
	}{
		{"이미지", Attachment{Filename: "bug.png", MimeType: "image/png"}, "이미지: bug.png"},
		{"동영상", Attachment{Filename: "demo.mp4", MimeType: "video/mp4"}, "동영상: demo.mp4"},
		{"동영상_MOV", Attachment{Filename: "record.mov", MimeType: "video/quicktime"}, "동영상: record.mov"},
		{"기타", Attachment{Filename: "log.txt", MimeType: "text/plain"}, "첨부: log.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAttachmentLabel(tt.att)
			if got != tt.want {
				t.Errorf("formatAttachmentLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
