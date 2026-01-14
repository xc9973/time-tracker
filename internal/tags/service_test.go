package tags

import (
	"os"
	"strings"
	"testing"

	"time-tracker/internal/shared/database"
)

func TestTagService_DuplicateName(t *testing.T) {
	tmp, err := os.CreateTemp("", "tags_svc_*.db")
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

	_, err = svc.Create(&TagCreate{Name: "work", Color: "#3B82F6"})
	if err != nil {
		t.Fatalf("expected first create ok, got %v", err)
	}

	_, err = svc.Create(&TagCreate{Name: "work", Color: "#3B82F6"})
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
	if !strings.Contains(err.Error(), "tags") && !strings.Contains(err.Error(), "UNIQUE") {
		// sqlite error message varies, just ensure we didn't silently succeed
		t.Fatalf("unexpected duplicate error: %v", err)
	}
}
