package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"time"

	"time-tracker/internal/sessions/models"
	"time-tracker/internal/sessions/repository"

	"time-tracker/internal/shared/config"
	"time-tracker/internal/shared/utils"
)

// Session service errors
var (
	ErrSessionAlreadyRunning = errors.New("a session is already running")
	ErrNoRunningSession      = errors.New("no running session found")
)

// CurrentSessionResponse represents the response for current session status.
type CurrentSessionResponse struct {
	Running    bool                    `json:"running"`
	Session    *models.SessionResponse `json:"session,omitempty"`
	ElapsedSec *int64                  `json:"elapsed_sec,omitempty"`
}

// SessionService handles business logic for session operations.
type SessionService struct {
	repo *repository.SessionRepository
}

// NewSessionService creates a new SessionService.
func NewSessionService(repo *repository.SessionRepository) *SessionService {
	return &SessionService{
		repo: repo,
	}
}

// StartSession starts a new session after checking for conflicts.
// Returns ErrSessionAlreadyRunning if a session is already running.
func (s *SessionService) StartSession(data *models.SessionStart) (*models.SessionResponse, error) {
	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Check for existing running session
	running, err := s.repo.GetRunning()
	if err != nil {
		return nil, err
	}
	if running != nil {
		return running, ErrSessionAlreadyRunning
	}

	return s.repo.Create(data)
}

// DeleteSession deletes a session entry.
func (s *SessionService) DeleteSession(id int64) error {
	return s.repo.Delete(id)
}

// UpdateSession updates a session entry after validation.
func (s *SessionService) UpdateSession(id int64, data *models.SessionUpdate) error {
	if err := data.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// If timestamps are modified, we might need to recalculate duration
	if data.StartedAt != nil || data.EndedAt != nil {
		session, err := s.repo.GetByID(id)
		if err != nil {
			return err
		}
		if session == nil {
			return errors.New("session not found")
		}

		// Only recalculate if session is stopped
		if session.Status == string(models.SessionStatusStopped) {
			startTimeStr := session.StartedAt
			if data.StartedAt != nil {
				startTimeStr = *data.StartedAt
			}
			endTimeStr := ""
			if session.EndedAt != nil {
				endTimeStr = *session.EndedAt
			}
			if data.EndedAt != nil {
				endTimeStr = *data.EndedAt
			}

			if endTimeStr != "" {
				start, err1 := time.Parse(time.RFC3339, startTimeStr)
				end, err2 := time.Parse(time.RFC3339, endTimeStr)
				if err1 == nil && err2 == nil {
					duration := int64(end.Sub(start).Seconds())
					data.DurationSec = &duration
				}
			}
		}
	}

	return s.repo.Update(id, data)
}

// StopSession stops the currently running session.
// Returns ErrNoRunningSession if no session is running.
func (s *SessionService) StopSession(data *models.SessionStop) (*models.SessionResponse, error) {
	if data != nil {
		if err := data.Validate(); err != nil {
			return nil, fmt.Errorf("validation error: %w", err)
		}
	} else {
		data = &models.SessionStop{}
	}

	session, err := s.repo.StopRunning(data)
	if errors.Is(err, repository.ErrNoRunningSession) {
		return nil, ErrNoRunningSession
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetCurrent returns the current session status.
func (s *SessionService) GetCurrent() (*CurrentSessionResponse, error) {
	running, err := s.repo.GetRunning()
	if err != nil {
		return nil, err
	}

	if running == nil {
		return &CurrentSessionResponse{
			Running: false,
		}, nil
	}

	// Calculate elapsed time
	startTime, err := time.Parse(time.RFC3339, running.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse started_at: %w", err)
	}
	elapsed := int64(time.Since(startTime).Seconds())

	return &CurrentSessionResponse{
		Running:    true,
		Session:    running,
		ElapsedSec: &elapsed,
	}, nil
}

// GetSessions retrieves a paginated list of sessions with optional filters.
func (s *SessionService) GetSessions(limit, offset int, status, category *string) (*models.PaginatedResponse[models.SessionResponse], error) {
	// Apply default and max limits
	if limit <= 0 {
		limit = config.DefaultPageSize
	}
	if limit > config.MaxPageSize {
		limit = config.MaxPageSize
	}
	if offset < 0 {
		offset = 0
	}

	sessions, err := s.repo.List(limit, offset, status, category)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.Count(status, category)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResponse[models.SessionResponse]{
		Items:  sessions,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ExportCSV exports sessions as CSV with UTF-8 BOM for Excel compatibility.
// Includes duration in human-readable format (H:MM:SS).
func (s *SessionService) ExportCSV(status, category *string) ([]byte, error) {
	// Get all matching sessions (no pagination for export)
	sessions, err := s.repo.List(config.MaxExportLimit, 0, status, category)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	// Write UTF-8 BOM
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"id", "category", "task", "note", "location", "mood", "started_at", "ended_at", "duration", "status"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, session := range sessions {
		row := []string{
			fmt.Sprintf("%d", session.ID),
			session.Category,
			session.Task,
			utils.PtrToString(session.Note),
			utils.PtrToString(session.Location),
			utils.PtrToString(session.Mood),
			session.StartedAt,
			utils.PtrToString(session.EndedAt),
			utils.FormatDuration(session.DurationSec),
			session.Status,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}
