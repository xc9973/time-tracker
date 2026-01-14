package tags

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
