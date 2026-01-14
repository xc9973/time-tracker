package models

import (
	"testing"

	"pgregory.net/rapid"
	"time-tracker/internal/shared/config"
)

// TestSessionStart_Validation tests SessionStart validation.
func TestSessionStart_Validation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		category := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "category")
		task := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "task")

		session := &SessionStart{
			Category: category,
			Task:     task,
		}

		err := session.Validate()
		if err != nil {
			t.Fatalf("expected no error for valid session start, got %v", err)
		}
	})
}

// TestSessionStart_MissingCategory ensures missing category falls back to default.
func TestSessionStart_MissingCategory(t *testing.T) {
	session := &SessionStart{
		Category: "",
		Task:     "valid task",
	}

	err := session.Validate()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Category != config.DefaultCategory {
		t.Fatalf("expected default category %q, got %q", config.DefaultCategory, session.Category)
	}
}

// TestSessionStart_MissingTask tests that sessions without task are rejected.
func TestSessionStart_MissingTask(t *testing.T) {
	session := &SessionStart{
		Category: "valid category",
		Task:     "",
	}

	err := session.Validate()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Task != config.DefaultTask {
		t.Fatalf("expected default task %q, got %q", config.DefaultTask, session.Task)
	}
}