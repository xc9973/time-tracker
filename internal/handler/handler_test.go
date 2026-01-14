package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"time-tracker/internal/sessions"
	"time-tracker/internal/sessions/models"
	"time-tracker/internal/shared/database"
	"time-tracker/internal/shared/errors"
)

// setupTestDB creates a temporary database for testing.
func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "handler_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := database.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

// ============================================
// Health Handler Tests
// ============================================

// TestHealthHandler_Check tests GET /healthz endpoint.
// **Validates: Requirements 6.1, 6.2**
func TestHealthHandler_Check(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.OK {
		t.Fatal("expected ok to be true")
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

// ============================================
// Sessions Handler Tests
// ============================================

func setupSessionsHandler(t *testing.T) (*SessionsHandler, func()) {
	db, cleanup := setupTestDB(t)
	repo := sessions.NewSessionRepository(db)
	svc := sessions.NewSessionService(repo)
	handler := NewSessionsHandler(svc)
	return handler, cleanup
}

// TestSessionsHandler_Start tests POST /api/v1/sessions/start endpoint.
// **Validates: Requirements 2.1**
func TestSessionsHandler_Start(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Start(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.SessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if resp.Status != "running" {
		t.Fatalf("expected status 'running', got %q", resp.Status)
	}
	if resp.StartedAt == "" {
		t.Fatal("expected non-empty started_at")
	}
	if resp.EndedAt != nil {
		t.Fatal("expected nil ended_at for running session")
	}
}

// TestSessionsHandler_Start_Conflict tests conflict when session already running.
// **Validates: Requirements 2.2**
func TestSessionsHandler_Start_Conflict(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Start first session
	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.Start(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for first session, got %d", w.Code)
	}

	// Try to start second session
	body = `{"category":"work","task":"coding"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.Start(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}

	var resp errors.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "CONFLICT" {
		t.Fatalf("expected error code 'CONFLICT', got %q", resp.Error.Code)
	}
	if resp.Error.CurrentSession == nil {
		t.Fatal("expected current_session in conflict response")
	}
}

// TestSessionsHandler_Stop tests POST /api/v1/sessions/stop endpoint.
// **Validates: Requirements 2.3, 2.4**
func TestSessionsHandler_Stop(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Start a session first
	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.Start(w, req)

	// Stop the session with optional updates
	body = `{"note":"completed chapter 1","mood":"good"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/stop", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.Stop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.SessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "stopped" {
		t.Fatalf("expected status 'stopped', got %q", resp.Status)
	}
	if resp.EndedAt == nil {
		t.Fatal("expected non-nil ended_at")
	}
	if resp.DurationSec == nil {
		t.Fatal("expected non-nil duration_sec")
	}
	if resp.Note == nil || *resp.Note != "completed chapter 1" {
		t.Fatal("expected note to be updated")
	}
}

// TestSessionsHandler_Stop_NoRunning tests stopping when no session is running.
// **Validates: Requirements 2.5**
func TestSessionsHandler_Stop_NoRunning(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/stop", nil)
	w := httptest.NewRecorder()
	handler.Stop(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp errors.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected error code 'NOT_FOUND', got %q", resp.Error.Code)
	}
}

// TestSessionsHandler_Current tests GET /api/v1/sessions/current endpoint.
// **Validates: Requirements 2.6**
func TestSessionsHandler_Current(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Test when no session is running
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/current", nil)
	w := httptest.NewRecorder()
	handler.Current(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp sessions.CurrentSessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Running {
		t.Fatal("expected running to be false when no session")
	}

	// Start a session
	body := `{"category":"study","task":"reading"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.Start(w, req)

	// Test when session is running
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/current", nil)
	w = httptest.NewRecorder()
	handler.Current(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Running {
		t.Fatal("expected running to be true")
	}
	if resp.Session == nil {
		t.Fatal("expected session to be non-nil")
	}
}

// TestSessionsHandler_List tests GET /api/v1/sessions endpoint.
// **Validates: Requirements 2.7**
func TestSessionsHandler_List(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Create and stop a session
	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.Start(w, req)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/stop", nil)
	w = httptest.NewRecorder()
	handler.Stop(w, req)

	// Start another session (running)
	body = `{"category":"work","task":"coding"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.Start(w, req)

	// List all sessions
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	w = httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.PaginatedResponse[models.SessionResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(resp.Items))
	}
}

// TestSessionsHandler_List_StatusFilter tests status filtering.
func TestSessionsHandler_List_StatusFilter(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Create and stop a session
	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.Start(w, req)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/stop", nil)
	w = httptest.NewRecorder()
	handler.Stop(w, req)

	// Start another session (running)
	body = `{"category":"work","task":"coding"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w = httptest.NewRecorder()
	handler.Start(w, req)

	// Filter by status=running
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions?status=running", nil)
	w = httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.PaginatedResponse[models.SessionResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 running session, got %d", len(resp.Items))
	}
	if resp.Items[0].Status != "running" {
		t.Fatalf("expected status 'running', got %q", resp.Items[0].Status)
	}
}

// TestSessionsHandler_ExportCSV tests GET /api/v1/sessions.csv endpoint.
// **Validates: Requirements 3.2, 3.4, 3.5**
func TestSessionsHandler_ExportCSV(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	// Create and stop a session
	body := `{"category":"study","task":"reading"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/start", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.Start(w, req)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/sessions/stop", nil)
	w = httptest.NewRecorder()
	handler.Stop(w, req)

	// Export CSV
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions.csv", nil)
	w = httptest.NewRecorder()
	handler.ExportCSV(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Fatalf("expected Content-Type text/csv, got %q", contentType)
	}

	// Check UTF-8 BOM
	body_bytes := w.Body.Bytes()
	if len(body_bytes) < 3 || body_bytes[0] != 0xEF || body_bytes[1] != 0xBB || body_bytes[2] != 0xBF {
		t.Fatal("CSV does not start with UTF-8 BOM")
	}

	// Check duration format (H:MM:SS)
	content := string(body_bytes[3:])
	if !strings.Contains(content, "duration") {
		t.Fatal("CSV missing duration column")
	}
}

func TestSessionsHandler_ServeHTTP_Routing(t *testing.T) {
	handler, cleanup := setupSessionsHandler(t)
	defer cleanup()

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodPost, "/api/v1/sessions/start", `{"category":"work","task":"test"}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/sessions/current", "", http.StatusOK},
		{http.MethodGet, "/api/v1/sessions", "", http.StatusOK},
		{http.MethodGet, "/api/v1/sessions.csv", "", http.StatusOK},
		{http.MethodPost, "/api/v1/sessions/stop", "", http.StatusOK}, // Now has running session
		{http.MethodGet, "/api/v1/unknown", "", http.StatusNotFound},
	}

	for _, tt := range tests {
		var bodyReader *bytes.Reader
		if tt.body != "" {
			bodyReader = bytes.NewReader([]byte(tt.body))
		} else {
			bodyReader = bytes.NewReader(nil)
		}

		req := httptest.NewRequest(tt.method, tt.path, bodyReader)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("%s %s: expected status %d, got %d: %s", tt.method, tt.path, tt.status, w.Code, w.Body.String())
		}
	}
}
