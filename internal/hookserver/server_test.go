package hookserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestServer_HandleHook은 Hook 핸들링을 테스트합니다.
func TestServer_HandleHook(t *testing.T) {
	var receivedPayload *StopHookPayload

	callback := func(payload *StopHookPayload) {
		receivedPayload = payload
	}

	server := NewServer(8081, callback)

	payload := StopHookPayload{
		Cwd:      "/test/project",
		ExitCode: 0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/hook/stop", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handleHook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusOK)
	}

	if receivedPayload == nil {
		t.Fatal("콜백이 호출되어야 함")
	}
	if receivedPayload.Cwd != "/test/project" {
		t.Errorf("Cwd 불일치: got %s, want /test/project", receivedPayload.Cwd)
	}
}

// TestServer_HandleHook_InvalidJSON은 잘못된 JSON을 테스트합니다.
func TestServer_HandleHook_InvalidJSON(t *testing.T) {
	callback := func(payload *StopHookPayload) {}
	server := NewServer(8081, callback)

	req := httptest.NewRequest("POST", "/hook/stop", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handleHook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestServer_HandleHook_MethodNotAllowed는 잘못된 메서드를 테스트합니다.
func TestServer_HandleHook_MethodNotAllowed(t *testing.T) {
	callback := func(payload *StopHookPayload) {}
	server := NewServer(8081, callback)

	req := httptest.NewRequest("GET", "/hook/stop", nil)

	w := httptest.NewRecorder()
	server.handleHook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestStopHookPayload_Parse는 페이로드 파싱을 테스트합니다.
func TestStopHookPayload_Parse(t *testing.T) {
	jsonData := `{
		"cwd": "/Users/zime/project",
		"session_id": "abc123",
		"transcript_path": "/tmp/transcript.json",
		"exit_code": 0
	}`

	var payload StopHookPayload
	err := json.Unmarshal([]byte(jsonData), &payload)

	if err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	if payload.Cwd != "/Users/zime/project" {
		t.Errorf("Cwd 불일치: %s", payload.Cwd)
	}
	if payload.SessionID != "abc123" {
		t.Errorf("SessionID 불일치: %s", payload.SessionID)
	}
	if payload.ExitCode != 0 {
		t.Errorf("ExitCode 불일치: %d", payload.ExitCode)
	}
}

// TestServer_HandlePlanReady는 Plan Ready 핸들링을 테스트합니다.
func TestServer_HandlePlanReady(t *testing.T) {
	var receivedPayload *PlanReadyPayload

	planReadyCallback := func(payload *PlanReadyPayload) {
		receivedPayload = payload
	}

	server := NewServer(8081, nil)
	server.SetPlanReadyCallback(planReadyCallback)

	payload := PlanReadyPayload{
		Cwd:       "/test/project",
		TaskName:  "테스트 태스크",
		PlanTitle: "계획 수립 완료",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/hook/plan-ready", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handlePlanReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("상태코드 불일치: got %d, want %d", w.Code, http.StatusOK)
	}

	if receivedPayload == nil {
		t.Fatal("콜백이 호출되어야 함")
	}
	if receivedPayload.Cwd != "/test/project" {
		t.Errorf("Cwd 불일치: got %s, want /test/project", receivedPayload.Cwd)
	}
	if receivedPayload.PlanTitle != "계획 수립 완료" {
		t.Errorf("PlanTitle 불일치: got %s", receivedPayload.PlanTitle)
	}
}

// TestPlanReadyPayload_Parse는 PlanReady 페이로드 파싱을 테스트합니다.
func TestPlanReadyPayload_Parse(t *testing.T) {
	jsonData := `{
		"cwd": "/Users/zime/project",
		"task_id": "task123",
		"task_name": "버그 수정",
		"plan_title": "계획 수립 완료"
	}`

	var payload PlanReadyPayload
	err := json.Unmarshal([]byte(jsonData), &payload)

	if err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	if payload.Cwd != "/Users/zime/project" {
		t.Errorf("Cwd 불일치: %s", payload.Cwd)
	}
	if payload.TaskID != "task123" {
		t.Errorf("TaskID 불일치: %s", payload.TaskID)
	}
	if payload.TaskName != "버그 수정" {
		t.Errorf("TaskName 불일치: %s", payload.TaskName)
	}
	if payload.PlanTitle != "계획 수립 완료" {
		t.Errorf("PlanTitle 불일치: %s", payload.PlanTitle)
	}
}
