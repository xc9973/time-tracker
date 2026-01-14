package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_CreatesTagsTables(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "timetracker-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Verify tags table exists
	var tagsTableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tags'").Scan(&tagsTableExists)
	if err != nil {
		t.Fatalf("failed to check tags table: %v", err)
	}
	if tagsTableExists != 1 {
		t.Error("tags table was not created")
	}

	// Verify session_tags table exists
	var sessionTagsTableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='session_tags'").Scan(&sessionTagsTableExists)
	if err != nil {
		t.Fatalf("failed to check session_tags table: %v", err)
	}
	if sessionTagsTableExists != 1 {
		t.Error("session_tags table was not created")
	}
}
