package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_CreatesTablesAndIndexes(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "timetracker-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Verify sessions table exists
	var sessionsTableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&sessionsTableExists)
	if err != nil {
		t.Fatalf("failed to check sessions table: %v", err)
	}
	if sessionsTableExists != 1 {
		t.Error("sessions table was not created")
	}

	// Verify sessions indexes exist
	sessionsIndexes := []string{"idx_sessions_started_at", "idx_sessions_status", "idx_sessions_category"}
	for _, idx := range sessionsIndexes {
		var indexExists int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&indexExists)
		if err != nil {
			t.Fatalf("failed to check index %s: %v", idx, err)
		}
		if indexExists != 1 {
			t.Errorf("index %s was not created", idx)
		}
	}
}

func TestNew_IdempotentTableCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "timetracker-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database twice - should not error
	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}
	db1.Close()

	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}
	db2.Close()
}

func TestDB_Path(t *testing.T) {
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

	if db.Path() != dbPath {
		t.Errorf("expected path %s, got %s", dbPath, db.Path())
	}
}
