// Package models defines data structures and validation for the time tracker.
package models

import (
	"errors"
	"time"

	"time-tracker/internal/shared/config"
	"time-tracker/internal/shared/validation"
)

// Field length constraints
const (
	CategoryMinLen = 1
	CategoryMaxLen = 50
	TaskMinLen     = 1
	TaskMaxLen     = 200
	NoteMaxLen     = 1000
	LocationMaxLen = 100
	MoodMaxLen     = 20
)

// Validation errors
var (
	ErrCategoryRequired = errors.New("category is required")
	ErrCategoryTooLong  = errors.New("category must be at most 50 characters")
	ErrTaskRequired     = errors.New("task is required")
	ErrTaskTooLong      = errors.New("task must be at most 200 characters")
	ErrNoteTooLong      = errors.New("note must be at most 1000 characters")
	ErrLocationTooLong  = errors.New("location must be at most 100 characters")
	ErrMoodTooLong      = errors.New("mood must be at most 20 characters")
)



// SessionStart represents the input for starting a new session.
type SessionStart struct {
	Category string  `json:"category"`
	Task     string  `json:"task"`
	Note     *string `json:"note,omitempty"`
	Location *string `json:"location,omitempty"`
	Mood     *string `json:"mood,omitempty"`
}

// Validate checks if the SessionStart fields meet the requirements and sanitizes inputs.
// Special characters are preserved (not escaped) as they are safely stored via parameterized queries.
func (s *SessionStart) Validate() error {
	// Sanitize inputs
	s.Category = validation.SanitizeString(s.Category)
	s.Task = validation.SanitizeString(s.Task)
	s.Note = validation.SanitizeStringPtr(s.Note)
	s.Location = validation.SanitizeStringPtr(s.Location)
	s.Mood = validation.SanitizeStringPtr(s.Mood)

	// Validate required fields
	if s.Category == "" {
		s.Category = config.DefaultCategory
	}
	if len(s.Category) > CategoryMaxLen {
		return ErrCategoryTooLong
	}

	if s.Task == "" {
		s.Task = config.DefaultTask
	}
	if len(s.Task) > TaskMaxLen {
		return ErrTaskTooLong
	}

	if s.Note != nil && len(*s.Note) > NoteMaxLen {
		return ErrNoteTooLong
	}

	if s.Location != nil && len(*s.Location) > LocationMaxLen {
		return ErrLocationTooLong
	}

	if s.Mood != nil && len(*s.Mood) > MoodMaxLen {
		return ErrMoodTooLong
	}

	return nil
}

// SessionStop represents the input for stopping a session.
type SessionStop struct {
	Note     *string `json:"note,omitempty"`
	Location *string `json:"location,omitempty"`
	Mood     *string `json:"mood,omitempty"`
}

// Validate checks if the SessionStop fields meet the requirements and sanitizes inputs.
// Special characters are preserved (not escaped) as they are safely stored via parameterized queries.
func (s *SessionStop) Validate() error {
	// Sanitize inputs
	s.Note = validation.SanitizeStringPtr(s.Note)
	s.Location = validation.SanitizeStringPtr(s.Location)
	s.Mood = validation.SanitizeStringPtr(s.Mood)

	if s.Note != nil && len(*s.Note) > NoteMaxLen {
		return ErrNoteTooLong
	}

	if s.Location != nil && len(*s.Location) > LocationMaxLen {
		return ErrLocationTooLong
	}

	if s.Mood != nil && len(*s.Mood) > MoodMaxLen {
		return ErrMoodTooLong
	}

	return nil
}

// SessionUpdate represents the input for updating a session.
type SessionUpdate struct {
	Category  *string `json:"category,omitempty"`
	Task      *string `json:"task,omitempty"`
	Note      *string `json:"note,omitempty"`
	Location  *string `json:"location,omitempty"`
	Mood      *string `json:"mood,omitempty"`
	StartedAt *string `json:"started_at,omitempty"`
	EndedAt   *string `json:"ended_at,omitempty"`
	DurationSec *int64 `json:"duration_sec,omitempty"`
}

// Validate checks if the SessionUpdate fields meet the requirements.
func (s *SessionUpdate) Validate() error {
	// Sanitize inputs
	s.Category = validation.SanitizeStringPtr(s.Category)
	s.Task = validation.SanitizeStringPtr(s.Task)
	s.Note = validation.SanitizeStringPtr(s.Note)
	s.Location = validation.SanitizeStringPtr(s.Location)
	s.Mood = validation.SanitizeStringPtr(s.Mood)

	if s.Category != nil {
		if *s.Category == "" {
			return ErrCategoryRequired
		}
		if len(*s.Category) > CategoryMaxLen {
			return ErrCategoryTooLong
		}
	}

	if s.Task != nil {
		if *s.Task == "" {
			return ErrTaskRequired
		}
		if len(*s.Task) > TaskMaxLen {
			return ErrTaskTooLong
		}
	}

	if s.Note != nil && len(*s.Note) > NoteMaxLen {
		return ErrNoteTooLong
	}

	if s.Location != nil && len(*s.Location) > LocationMaxLen {
		return ErrLocationTooLong
	}

	if s.Mood != nil && len(*s.Mood) > MoodMaxLen {
		return ErrMoodTooLong
	}

	return nil
}

// SessionStatus represents the status of a session.
type SessionStatus string

const (
	SessionStatusRunning SessionStatus = "running"
	SessionStatusStopped SessionStatus = "stopped"
)

// SessionResponse represents a session returned from the API.
type SessionResponse struct {
	ID          int64   `json:"id"`
	Category    string  `json:"category"`
	Task        string  `json:"task"`
	Note        *string `json:"note,omitempty"`
	Location    *string `json:"location,omitempty"`
	Mood        *string `json:"mood,omitempty"`
	StartedAt   string  `json:"started_at"`
	EndedAt     *string `json:"ended_at,omitempty"`
	DurationSec *int64  `json:"duration_sec,omitempty"`
	Status      string  `json:"status"`
}

// PaginatedResponse wraps a list of items with pagination metadata.
type PaginatedResponse[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// FormatRFC3339 formats a time.Time to RFC3339 UTC string.
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// NowRFC3339 returns the current time as RFC3339 UTC string.
func NowRFC3339() string {
	return FormatRFC3339(time.Now())
}