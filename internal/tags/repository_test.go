package tags

import (
	"os"
	"testing"

	"time-tracker/internal/shared/database"
)

func TestTagRepository_CreateAndList(t *testing.T) {
	tmp, err := os.CreateTemp("", "tags_repo_*.db")
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

	created, err := repo.Create(&TagCreate{Name: "工作", Color: "#3B82F6"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatalf("expected id")
	}

	items, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1, got %d", len(items))
	}
}
