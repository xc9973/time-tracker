package service

import "time-tracker/internal/models"

// SessionServiceInterface defines the interface for session service operations.
type SessionServiceInterface interface {
	StartSession(data *models.SessionStart) (*models.SessionResponse, error)
	DeleteSession(id int64) error
	UpdateSession(id int64, data *models.SessionUpdate) error
	StopSession(data *models.SessionStop) (*models.SessionResponse, error)
	GetCurrent() (*CurrentSessionResponse, error)
	GetSessions(limit, offset int, status, category *string) (*models.PaginatedResponse[models.SessionResponse], error)
	ExportCSV(status, category *string) ([]byte, error)
}
