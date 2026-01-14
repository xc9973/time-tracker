package service

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
	"time-tracker/internal/models"
	"time-tracker/internal/repository"

	"time-tracker/internal/shared/database"
	"time-tracker/internal/shared/utils"
)

// Feature: time-tracker, Property 4: Session 生命周期
// **Validates: Requirements 2.1, 2.3**
//
// For any Session:
// - When created, status is "running", has started_at timestamp, ended_at and duration_sec are null
// - After stopped, status is "stopped", has ended_at timestamp, duration_sec = ended_at - started_at (seconds)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "service_test_*.db")
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

func TestSessionService_Property4_Lifecycle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	rapid.Check(t, func(t *rapid.T) {
		category := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category")
		task := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task")

		// Start a session
		session, err := svc.StartSession(&models.SessionStart{
			Category: category,
			Task:     task,
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}

		// Verify initial state
		if session.Status != "running" {
			t.Fatalf("expected status 'running', got %q", session.Status)
		}
		if session.StartedAt == "" {
			t.Fatal("expected started_at to be set")
		}
		if session.EndedAt != nil {
			t.Fatal("expected ended_at to be nil for running session")
		}
		if session.DurationSec != nil {
			t.Fatal("expected duration_sec to be nil for running session")
		}

		// Stop the session
		stopped, err := svc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}

		// Verify stopped state
		if stopped.Status != "stopped" {
			t.Fatalf("expected status 'stopped', got %q", stopped.Status)
		}
		if stopped.EndedAt == nil {
			t.Fatal("expected ended_at to be set after stop")
		}
		if stopped.DurationSec == nil {
			t.Fatal("expected duration_sec to be set after stop")
		}
		if *stopped.DurationSec < 0 {
			t.Fatalf("expected non-negative duration, got %d", *stopped.DurationSec)
		}
	})
}


// Feature: time-tracker, Property 5: Session 并发控制
// **Validates: Requirements 2.2**
//
// For any existing running Session, attempting to create a new Session should return
// 409 Conflict with the current running Session's information.

func TestSessionService_Property5_ConcurrencyControl(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	rapid.Check(t, func(t *rapid.T) {
		category1 := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category1")
		task1 := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task1")
		category2 := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category2")
		task2 := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task2")

		// Start first session
		first, err := svc.StartSession(&models.SessionStart{
			Category: category1,
			Task:     task1,
		})
		if err != nil {
			t.Fatalf("failed to start first session: %v", err)
		}

		// Try to start second session - should fail with conflict
		running, err := svc.StartSession(&models.SessionStart{
			Category: category2,
			Task:     task2,
		})

		if err != ErrSessionAlreadyRunning {
			t.Fatalf("expected ErrSessionAlreadyRunning, got %v", err)
		}

		// Verify the returned session is the first one
		if running == nil {
			t.Fatal("expected running session to be returned on conflict")
		}
		if running.ID != first.ID {
			t.Fatalf("expected running session ID %d, got %d", first.ID, running.ID)
		}

		// Clean up - stop the session for next iteration
		_, err = svc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}
	})
}

// Feature: time-tracker, Property 6: Session 停止时更新
// **Validates: Requirements 2.4**
//
// For any optional fields (note, mood, location) provided in the stop request,
// the stopped Session should contain these updated field values.

func TestSessionService_Property6_StopUpdates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	rapid.Check(t, func(t *rapid.T) {
		category := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category")
		task := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task")

		// Start a session without optional fields
		_, err := svc.StartSession(&models.SessionStart{
			Category: category,
			Task:     task,
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}

		// Generate optional fields for stop
		// Use non-whitespace patterns to ensure values are preserved after sanitization
		var note, location, mood *string
		if rapid.Bool().Draw(t, "hasNote") {
			n := rapid.StringMatching(`[a-zA-Z0-9]{1,100}`).Draw(t, "note")
			note = &n
		}
		if rapid.Bool().Draw(t, "hasLocation") {
			l := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "location")
			location = &l
		}
		if rapid.Bool().Draw(t, "hasMood") {
			m := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "mood")
			mood = &m
		}

		// Stop with updates
		stopped, err := svc.StopSession(&models.SessionStop{
			Note:     note,
			Location: location,
			Mood:     mood,
		})
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}

		// Verify updates were applied
		if note != nil {
			if stopped.Note == nil || *stopped.Note != *note {
				t.Fatalf("note not updated: expected %v, got %v", note, stopped.Note)
			}
		}
		if location != nil {
			if stopped.Location == nil || *stopped.Location != *location {
				t.Fatalf("location not updated: expected %v, got %v", location, stopped.Location)
			}
		}
		if mood != nil {
			if stopped.Mood == nil || *stopped.Mood != *mood {
				t.Fatalf("mood not updated: expected %v, got %v", mood, stopped.Mood)
			}
		}
	})
}


// Feature: time-tracker, Property 7: Session 查询正确性
// **Validates: Requirements 2.6, 2.7**
//
// For any Session query request:
// - Querying current returns the running Session if exists, otherwise running=false
// - Using status filter, all returned Sessions have matching status
// - Using category filter, all returned Sessions have matching category

func TestSessionService_Property7_QueryCorrectness_Current(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	// Test when no session is running
	current, err := svc.GetCurrent()
	if err != nil {
		t.Fatalf("failed to get current: %v", err)
	}
	if current.Running {
		t.Fatal("expected running=false when no session")
	}
	if current.Session != nil {
		t.Fatal("expected session to be nil when not running")
	}

	// Start a session
	started, err := svc.StartSession(&models.SessionStart{
		Category: "test",
		Task:     "task",
	})
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	// Test when session is running
	current, err = svc.GetCurrent()
	if err != nil {
		t.Fatalf("failed to get current: %v", err)
	}
	if !current.Running {
		t.Fatal("expected running=true when session exists")
	}
	if current.Session == nil {
		t.Fatal("expected session to be set when running")
	}
	if current.Session.ID != started.ID {
		t.Fatalf("expected session ID %d, got %d", started.ID, current.Session.ID)
	}
	if current.ElapsedSec == nil {
		t.Fatal("expected elapsed_sec to be set")
	}
}

func TestSessionService_Property7_StatusFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	// Create some stopped sessions
	for i := 0; i < 3; i++ {
		_, err := svc.StartSession(&models.SessionStart{
			Category: "test",
			Task:     "task",
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}
		_, err = svc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}
	}

	// Create one running session
	_, err := svc.StartSession(&models.SessionStart{
		Category: "test",
		Task:     "running_task",
	})
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom([]string{"running", "stopped"}).Draw(t, "status")

		result, err := svc.GetSessions(50, 0, &status, nil)
		if err != nil {
			t.Fatalf("failed to get sessions: %v", err)
		}

		// Verify all returned sessions match the status
		for _, session := range result.Items {
			if session.Status != status {
				t.Fatalf("expected status %q, got %q", status, session.Status)
			}
		}
	})
}

func TestSessionService_Property7_CategoryFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	// Create sessions with different categories
	categories := []string{"work", "personal", "study"}
	for _, cat := range categories {
		_, err := svc.StartSession(&models.SessionStart{
			Category: cat,
			Task:     "task",
		})
		if err != nil {
			t.Fatalf("failed to start session: %v", err)
		}
		_, err = svc.StopSession(nil)
		if err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}
	}

	rapid.Check(t, func(t *rapid.T) {
		category := rapid.SampledFrom(categories).Draw(t, "category")

		result, err := svc.GetSessions(50, 0, nil, &category)
		if err != nil {
			t.Fatalf("failed to get sessions: %v", err)
		}

		// Verify all returned sessions match the category
		for _, session := range result.Items {
			if session.Category != category {
				t.Fatalf("expected category %q, got %q", category, session.Category)
			}
		}
	})
}

// TestSessionService_StopNoRunning tests stopping when no session is running.
func TestSessionService_StopNoRunning(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	_, err := svc.StopSession(nil)
	if err != ErrNoRunningSession {
		t.Fatalf("expected ErrNoRunningSession, got %v", err)
	}
}

// TestSessionService_ExportCSV tests CSV export functionality.
func TestSessionService_ExportCSV(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sessionRepo := repository.NewSessionRepository(db)
	svc := NewSessionService(sessionRepo)

	// Create and stop a session
	_, err := svc.StartSession(&models.SessionStart{
		Category: "work",
		Task:     "coding",
	})
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}
	_, err = svc.StopSession(nil)
	if err != nil {
		t.Fatalf("failed to stop session: %v", err)
	}

	// Export CSV
	csvData, err := svc.ExportCSV(nil, nil)
	if err != nil {
		t.Fatalf("failed to export CSV: %v", err)
	}

	// Verify UTF-8 BOM
	if len(csvData) < 3 || csvData[0] != 0xEF || csvData[1] != 0xBB || csvData[2] != 0xBF {
		t.Fatal("CSV does not start with UTF-8 BOM")
	}

	// Verify content contains header and data
	content := string(csvData[3:])
	if !strings.Contains(content, "id,category,task,note,location,mood,started_at,ended_at,duration,status") {
		t.Fatal("CSV missing header")
	}
	if !strings.Contains(content, "work") || !strings.Contains(content, "coding") {
		t.Fatal("CSV missing data")
	}
}

// TestSessionService_FormatDuration tests duration formatting.
func TestSessionService_FormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "0:00:00"},
		{59, "0:00:59"},
		{60, "0:01:00"},
		{3599, "0:59:59"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{7325, "2:02:05"},
	}

	for _, tt := range tests {
		result := utils.FormatDuration(&tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatDuration(%d) = %q, expected %q", tt.seconds, result, tt.expected)
		}
	}

	// Test nil
	if utils.FormatDuration(nil) != "" {
		t.Error("FormatDuration(nil) should return empty string")
	}
}
