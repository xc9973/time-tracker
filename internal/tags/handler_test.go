package tags

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"time-tracker/internal/sessions"
	"time-tracker/internal/shared/database"
)

func TestTagsHandler_CreateAndList(t *testing.T) {
	tmp, err := os.CreateTemp("", "tags_handler_*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()
	defer os.Remove(tmp.Name())

	db, err := database.New(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewTagRepository(db)
	svc := NewTagService(repo)
	h := NewTagsHandler(svc)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(`{"name":"工作","color":"#3B82F6"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createW.Code)
	}

	var created Tag
	if err := json.NewDecoder(createW.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected id")
	}
	if created.Name != "工作" {
		t.Fatalf("expected name %q, got %q", "工作", created.Name)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listW.Code)
	}

	var items []Tag
	if err := json.NewDecoder(listW.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1, got %d", len(items))
	}
}

func TestTagsHandler_SessionTagsAssociations(t *testing.T) {
	tmp, err := os.CreateTemp("", "tags_session_*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()
	defer os.Remove(tmp.Name())

	db, err := database.New(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Setup sessions and tags
	sessionRepo := sessions.NewSessionRepository(db)
	sessionSvc := sessions.NewSessionService(sessionRepo)
	tagRepo := NewTagRepository(db)
	tagSvc := NewTagService(tagRepo)
	h := NewTagsHandler(tagSvc)

	// Create a session
	start := &sessions.SessionStart{
		Category: "测试",
		Task:     "测试任务",
	}
	started, err := sessionSvc.StartSession(start)
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	// Create two tags
	tag1, err := tagSvc.Create(&TagCreate{Name: "工作", Color: "#3B82F6"})
	if err != nil {
		t.Fatalf("failed to create tag1: %v", err)
	}
	tag2, err := tagSvc.Create(&TagCreate{Name: "重要", Color: "#EF4444"})
	if err != nil {
		t.Fatalf("failed to create tag2: %v", err)
	}

	// Test POST /api/v1/sessions/:id/tags - assign tags
	sessionID := strconv.FormatInt(started.ID, 10)
	tag1ID := strconv.FormatInt(tag1.ID, 10)
	tag2ID := strconv.FormatInt(tag2.ID, 10)
	assignPath := "/api/v1/sessions/" + sessionID + "/tags"
	t.Logf("Assign path: %q", assignPath)
	assignReq := httptest.NewRequest(http.MethodPost, assignPath,
		strings.NewReader(`{"tag_ids":[`+tag1ID+`,`+tag2ID+`]}`))
	assignReq.Header.Set("Content-Type", "application/json")
	assignW := httptest.NewRecorder()
	h.ServeHTTP(assignW, assignReq)
	t.Logf("Assign response status: %d, body: %s", assignW.Code, assignW.Body.String())

	if assignW.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", assignW.Code, assignW.Body.String())
	}

	// Test GET /api/v1/sessions/:id/tags - list session tags
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/tags", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listW.Code)
	}

	var sessionTags []Tag
	if err := json.NewDecoder(listW.Body).Decode(&sessionTags); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(sessionTags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(sessionTags))
	}

	// Test DELETE /api/v1/sessions/:id/tags/:tag_id - remove tag
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+sessionID+"/tags/"+tag1ID, nil)
	deleteW := httptest.NewRecorder()
	h.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", deleteW.Code)
	}

	// Verify only one tag remains
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/tags", nil)
	listW2 := httptest.NewRecorder()
	h.ServeHTTP(listW2, listReq2)

	var remainingTags []Tag
	if err := json.NewDecoder(listW2.Body).Decode(&remainingTags); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(remainingTags) != 1 {
		t.Fatalf("expected 1 tag after deletion, got %d", len(remainingTags))
	}
}
