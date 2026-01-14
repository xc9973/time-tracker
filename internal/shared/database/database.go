// Package database provides SQLite connection management and table initialization.
package database

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection with initialization logic.
type DB struct {
	*sql.DB
	path string
	mu   sync.Mutex
}

// New creates a new database connection and initializes tables.
func New(dbPath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	// Configure connection pool for SQLite
	// SQLite supports only one writer at a time. Setting MaxOpenConns to 1
	// ensures that we don't run into "database is locked" errors during concurrent writes.
	// WAL mode allows concurrent readers, but keeping it simple with 1 connection
	// is the safest approach for SQLite unless we have high read throughput requirements.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0) // Reuse connections forever

	db := &DB{
		DB:   sqlDB,
		path: dbPath,
	}

	if err := db.initTables(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

// initTables creates the logs and sessions tables with indexes.
func (db *DB) initTables() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Create sessions table
	sessionsTableSQL := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT NOT NULL,
		task TEXT NOT NULL,
		note TEXT,
		location TEXT,
		mood TEXT,
		started_at TEXT NOT NULL,
		ended_at TEXT,
		duration_sec INTEGER,
		status TEXT NOT NULL
	);`

	if _, err := db.Exec(sessionsTableSQL); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create indexes for sessions table
	sessionsIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_category ON sessions(category);",
	}

	for _, idx := range sessionsIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create sessions index: %w", err)
		}
	}

	tagsTableSQL := `
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#6B7280',
		created_at TEXT NOT NULL
	);`

	if _, err := db.Exec(tagsTableSQL); err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);"); err != nil {
		return fmt.Errorf("failed to create tags index: %w", err)
	}

	sessionTagsTableSQL := `
	CREATE TABLE IF NOT EXISTS session_tags (
		session_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (session_id, tag_id),
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(sessionTagsTableSQL); err != nil {
		return fmt.Errorf("failed to create session_tags table: %w", err)
	}

	sessionTagsIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_session_tags_session ON session_tags(session_id);",
		"CREATE INDEX IF NOT EXISTS idx_session_tags_tag ON session_tags(tag_id);",
	}

	for _, idx := range sessionTagsIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create session_tags index: %w", err)
		}
	}

	return nil
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}
