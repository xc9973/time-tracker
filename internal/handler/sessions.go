package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"time-tracker/internal/sessions"
	"time-tracker/internal/sessions/models"

	"time-tracker/internal/shared/config"
	"time-tracker/internal/shared/errors"
	"time-tracker/internal/shared/utils"
	"time-tracker/internal/shared/validation"
)

// SessionsHandler handles HTTP requests for session operations.
type SessionsHandler struct {
	service *sessions.SessionService
}

// NewSessionsHandler creates a new SessionsHandler.
func NewSessionsHandler(svc *sessions.SessionService) *SessionsHandler {
	return &SessionsHandler{service: svc}
}

// Start handles POST /api/v1/sessions/start - starts a new session.
func (h *SessionsHandler) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.WriteError(w, errors.ValidationError("Method not allowed"))
		return
	}

	var input models.SessionStart
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.WriteError(w, errors.ValidationError("Invalid JSON body"))
		return
	}

	session, err := h.service.StartSession(&input)
	if err != nil {
		// Check for conflict error (session already running)
		if err == sessions.ErrSessionAlreadyRunning && session != nil {
			conflictErr := errors.NewConflictError("A session is already running", map[string]interface{}{
				"id":         session.ID,
				"task":       session.Task,
				"started_at": session.StartedAt,
			})
			errors.WriteError(w, conflictErr)
			return
		}
		// Check if it's a validation error
		if strings.Contains(err.Error(), "validation error") {
			errors.WriteError(w, errors.ValidationError(strings.TrimPrefix(err.Error(), "validation error: ")))
			return
		}
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// Stop handles POST /api/v1/sessions/stop - stops the current session.
func (h *SessionsHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.WriteError(w, errors.ValidationError("Method not allowed"))
		return
	}

	var input *models.SessionStop
	// Body is optional for stop
	if r.ContentLength > 0 {
		input = &models.SessionStop{}
		if err := json.NewDecoder(r.Body).Decode(input); err != nil {
			errors.WriteError(w, errors.ValidationError("Invalid JSON body"))
			return
		}
	}

	session, err := h.service.StopSession(input)
	if err != nil {
		if err == sessions.ErrNoRunningSession {
			errors.WriteError(w, errors.NotFoundError("No running session found"))
			return
		}
		// Check if it's a validation error
		if strings.Contains(err.Error(), "validation error") {
			errors.WriteError(w, errors.ValidationError(strings.TrimPrefix(err.Error(), "validation error: ")))
			return
		}
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// Current handles GET /api/v1/sessions/current - gets the current session status.
func (h *SessionsHandler) Current(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.WriteError(w, errors.ValidationError("Method not allowed"))
		return
	}

	result, err := h.service.GetCurrent()
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// List handles GET /api/v1/sessions - retrieves paginated sessions.
func (h *SessionsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.WriteError(w, errors.ValidationError("Method not allowed"))
		return
	}

	// Parse and sanitize query parameters
	query := r.URL.Query()

	limit, offset := utils.ParsePaginationParams(query, 10, config.MaxPageSize)

	// Sanitize status filter
	var status *string
	if s := query.Get("status"); s != "" {
		sanitized := validation.SanitizeString(s)
		if sanitized != "" {
			status = &sanitized
		}
	}

	// Sanitize category filter
	var category *string
	if c := query.Get("category"); c != "" {
		sanitized := validation.SanitizeString(c)
		if sanitized != "" {
			category = &sanitized
		}
	}

	result, err := h.service.GetSessions(limit, offset, status, category)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExportCSV handles GET /api/v1/sessions.csv - exports sessions as CSV.
func (h *SessionsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.WriteError(w, errors.ValidationError("Method not allowed"))
		return
	}

	// Parse and sanitize query parameters
	query := r.URL.Query()

	// Sanitize status filter
	var status *string
	if s := query.Get("status"); s != "" {
		sanitized := validation.SanitizeString(s)
		if sanitized != "" {
			status = &sanitized
		}
	}

	// Sanitize category filter
	var category *string
	if c := query.Get("category"); c != "" {
		sanitized := validation.SanitizeString(c)
		if sanitized != "" {
			category = &sanitized
		}
	}

	csvData, err := h.service.ExportCSV(status, category)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	// Set headers for CSV download
	filename := fmt.Sprintf("sessions_%s.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(csvData)
}

// ServeHTTP implements http.Handler for routing session requests.
func (h *SessionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/api/v1/sessions/start" && r.Method == http.MethodPost:
		h.Start(w, r)
	case path == "/api/v1/sessions/stop" && r.Method == http.MethodPost:
		h.Stop(w, r)
	case path == "/api/v1/sessions/current" && r.Method == http.MethodGet:
		h.Current(w, r)
	case path == "/api/v1/sessions" && r.Method == http.MethodGet:
		h.List(w, r)
	case path == "/api/v1/sessions.csv" && r.Method == http.MethodGet:
		h.ExportCSV(w, r)
	default:
		errors.WriteError(w, errors.NotFoundError("Endpoint not found"))
	}
}
