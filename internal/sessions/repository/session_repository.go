package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"time-tracker/internal/sessions/models"

	"time-tracker/internal/shared/database"
	"time-tracker/internal/shared/utils"
)

// ErrNoRunningSession is returned when no running session exists.
var ErrNoRunningSession = errors.New("no running session found")

// SessionRepository handles database operations for sessions.
type SessionRepository struct {
	db *database.DB
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(db *database.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create inserts a new session with status "running" and returns the complete SessionResponse.
func (r *SessionRepository) Create(session *models.SessionStart) (*models.SessionResponse, error) {
	startedAt := models.NowRFC3339()
	status := string(models.SessionStatusRunning)

	result, err := r.db.Exec(
		`INSERT INTO sessions (category, task, note, location, mood, started_at, status) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.Category, session.Task, session.Note, session.Location, session.Mood, startedAt, status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.SessionResponse{
		ID:        id,
		Category:  session.Category,
		Task:      session.Task,
		Note:      session.Note,
		Location:  session.Location,
		Mood:      session.Mood,
		StartedAt: startedAt,
		Status:    status,
	}, nil
}

// Delete removes a session entry by ID.
func (r *SessionRepository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}


// GetRunning returns the currently running session, or nil if none exists.
func (r *SessionRepository) GetRunning() (*models.SessionResponse, error) {
	var session models.SessionResponse
	var note, location, mood, endedAt sql.NullString
	var durationSec sql.NullInt64

	err := r.db.QueryRow(
		`SELECT id, category, task, note, location, mood, started_at, ended_at, duration_sec, status 
		 FROM sessions WHERE status = ? LIMIT 1`,
		string(models.SessionStatusRunning),
	).Scan(&session.ID, &session.Category, &session.Task, &note, &location, &mood,
		&session.StartedAt, &endedAt, &durationSec, &session.Status)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query running session: %w", err)
	}

	if note.Valid {
		session.Note = &note.String
	}
	if location.Valid {
		session.Location = &location.String
	}
	if mood.Valid {
		session.Mood = &mood.String
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.String
	}
	if durationSec.Valid {
		session.DurationSec = &durationSec.Int64
	}

	return &session, nil
}

// StopRunning stops the currently running session and updates it with the provided data.
// Returns ErrNoRunningSession if no running session exists.
func (r *SessionRepository) StopRunning(updates *models.SessionStop) (*models.SessionResponse, error) {
	// First get the running session
	running, err := r.GetRunning()
	if err != nil {
		return nil, err
	}
	if running == nil {
		return nil, ErrNoRunningSession
	}

	endedAt := models.NowRFC3339()

	// Calculate duration
	startTime, err := time.Parse(time.RFC3339, running.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse started_at: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, endedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ended_at: %w", err)
	}
	durationSec := int64(endTime.Sub(startTime).Seconds())

	// Merge updates with existing values
	note := running.Note
	if updates.Note != nil {
		note = updates.Note
	}
	location := running.Location
	if updates.Location != nil {
		location = updates.Location
	}
	mood := running.Mood
	if updates.Mood != nil {
		mood = updates.Mood
	}

	_, err = r.db.Exec(
		`UPDATE sessions SET ended_at = ?, duration_sec = ?, status = ?, note = ?, location = ?, mood = ? 
		 WHERE id = ?`,
		endedAt, durationSec, string(models.SessionStatusStopped), note, location, mood, running.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &models.SessionResponse{
		ID:          running.ID,
		Category:    running.Category,
		Task:        running.Task,
		Note:        note,
		Location:    location,
		Mood:        mood,
		StartedAt:   running.StartedAt,
		EndedAt:     &endedAt,
		DurationSec: &durationSec,
		Status:      string(models.SessionStatusStopped),
	}, nil
}


// List retrieves sessions with pagination and optional filters.
// Results are ordered by started_at descending.
func (r *SessionRepository) List(limit, offset int, status, category *string) ([]models.SessionResponse, error) {
	query := "SELECT id, category, task, note, location, mood, started_at, ended_at, duration_sec, status FROM sessions"
	args := []interface{}{}
	conditions := []string{}

	if status != nil && *status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, *status)
	}

	if category != nil && *category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, *category)
	}

	if len(conditions) > 0 {
		query += utils.BuildWhereClause(conditions)
	}

	query += " ORDER BY started_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	sessions := []models.SessionResponse{}
	for rows.Next() {
		var session models.SessionResponse
		var note, location, mood, endedAt sql.NullString
		var durationSec sql.NullInt64

		if err := rows.Scan(&session.ID, &session.Category, &session.Task, &note, &location, &mood,
			&session.StartedAt, &endedAt, &durationSec, &session.Status); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}

		if note.Valid {
			session.Note = &note.String
		}
		if location.Valid {
			session.Location = &location.String
		}
		if mood.Valid {
			session.Mood = &mood.String
		}
		if endedAt.Valid {
			session.EndedAt = &endedAt.String
		}
		if durationSec.Valid {
			session.DurationSec = &durationSec.Int64
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// Count returns the total number of sessions matching the filters.
func (r *SessionRepository) Count(status, category *string) (int64, error) {
	query := "SELECT COUNT(*) FROM sessions"
	args := []interface{}{}
	conditions := []string{}

	if status != nil && *status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, *status)
	}

	if category != nil && *category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, *category)
	}

	if len(conditions) > 0 {
		query += utils.BuildWhereClause(conditions)
	}

	var count int64
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}

// GetByID retrieves a session by ID.
func (r *SessionRepository) GetByID(id int64) (*models.SessionResponse, error) {
	var session models.SessionResponse
	var note, location, mood, endedAt sql.NullString
	var durationSec sql.NullInt64

	err := r.db.QueryRow(
		`SELECT id, category, task, note, location, mood, started_at, ended_at, duration_sec, status
		 FROM sessions WHERE id = ?`,
		id,
	).Scan(&session.ID, &session.Category, &session.Task, &note, &location, &mood,
		&session.StartedAt, &endedAt, &durationSec, &session.Status)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	if note.Valid {
		session.Note = &note.String
	}
	if location.Valid {
		session.Location = &location.String
	}
	if mood.Valid {
		session.Mood = &mood.String
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.String
	}
	if durationSec.Valid {
		session.DurationSec = &durationSec.Int64
	}

	return &session, nil
}

// Update updates a session entry.
func (r *SessionRepository) Update(id int64, data *models.SessionUpdate) error {
	fieldToCol := map[string]string{
		"Category":    "category",
		"Task":        "task",
		"Note":        "note",
		"Location":    "location",
		"Mood":        "mood",
		"StartedAt":   "started_at",
		"EndedAt":     "ended_at",
		"DurationSec": "duration_sec",
	}

	updates, args := utils.BuildUpdateQueryFromStruct(data, fieldToCol)

	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE sessions SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, id)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}
