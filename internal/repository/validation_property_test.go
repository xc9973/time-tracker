package repository

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
	"time-tracker/internal/models"

	"time-tracker/internal/shared/database"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "repository_test_*.db")
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

// TestValidation_Property13_RoundTrip_Session tests that sessions also
// handle special characters correctly in round-trip.
func TestValidation_Property13_RoundTrip_Session(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSessionRepository(db)

	maliciousInputs := []string{
		"'; DROP TABLE sessions; --",
		"<script>alert('XSS')</script>",
		"${7*7}",
		"../../../etc/passwd",
	}

	rapid.Check(t, func(t *rapid.T) {
		malicious := rapid.SampledFrom(maliciousInputs).Draw(t, "malicious")

		session := &models.SessionStart{
			Category: "test",
			Task:     malicious,
		}

		// Validate and sanitize
		if err := session.Validate(); err != nil {
			t.Fatalf("validation failed: %v", err)
		}

		// Store in database
		created, err := repo.Create(session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Retrieve from database
		sessions, err := repo.List(10, 0, nil, nil)
		if err != nil {
			t.Fatalf("failed to list sessions: %v", err)
		}

		// Find our session
		var found *models.SessionResponse
		for i := range sessions {
			if sessions[i].ID == created.ID {
				found = &sessions[i]
				break
			}
		}

		if found == nil {
			t.Fatal("created session not found in list")
		}

		// Verify malicious input was stored and retrieved correctly
		expected := strings.TrimSpace(malicious)
		if found.Task != expected {
			t.Fatalf("malicious input not preserved in round-trip: expected %q, got %q", expected, found.Task)
		}

		// Clean up - stop the session
		_, _ = repo.StopRunning(&models.SessionStop{})
	})
}