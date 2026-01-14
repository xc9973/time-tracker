package repository

import "time-tracker/internal/models"

// SessionRepositoryInterface defines the interface for session repository operations.
type SessionRepositoryInterface interface {
	Create(session *models.SessionStart) (*models.SessionResponse, error)
	Delete(id int64) error
	GetRunning() (*models.SessionResponse, error)
	StopRunning(updates *models.SessionStop) (*models.SessionResponse, error)
	List(limit, offset int, status, category *string) ([]models.SessionResponse, error)
	Count(status, category *string) (int64, error)
	GetByID(id int64) (*models.SessionResponse, error)
	Update(id int64, data *models.SessionUpdate) error
}
