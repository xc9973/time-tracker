package sessions

import (
	"time-tracker/internal/sessions/models"
	"time-tracker/internal/sessions/repository"
	"time-tracker/internal/sessions/service"
	"time-tracker/internal/shared/database"
)

// NewSessionRepository keeps legacy wiring stable while sessions are being migrated.
func NewSessionRepository(db *database.DB) *repository.SessionRepository {
	return repository.NewSessionRepository(db)
}

// NewSessionService keeps legacy wiring stable while sessions are being migrated.
func NewSessionService(repo *repository.SessionRepository) *service.SessionService {
	return service.NewSessionService(repo)
}

// Re-export types commonly referenced by handlers.
//
// Note: these are type aliases, so there is no runtime overhead.
type SessionRepository = repository.SessionRepository
type SessionService = service.SessionService

type SessionStart = models.SessionStart
type SessionStop = models.SessionStop
type SessionUpdate = models.SessionUpdate

type CurrentSessionResponse = service.CurrentSessionResponse

// Re-export errors commonly referenced by handlers.
var (
	ErrSessionAlreadyRunning = service.ErrSessionAlreadyRunning
	ErrNoRunningSession      = service.ErrNoRunningSession
)
