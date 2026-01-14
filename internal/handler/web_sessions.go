package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"time-tracker/internal/sessions"
	"time-tracker/internal/sessions/models"

	"time-tracker/internal/shared/utils"
	"time-tracker/internal/shared/validation"
)

// Sessions handles GET /web/sessions - displays the sessions list page.
func (h *WebHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	// Parse pagination
	page := 1
	if p := query.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 10
	offset := (page - 1) * limit

	// Parse and sanitize filters
	var category *string
	categoryStr := validation.SanitizeString(query.Get("category"))
	if categoryStr != "" {
		category = &categoryStr
	}

	var status *string
	statusStr := validation.SanitizeString(query.Get("status"))
	if statusStr != "" {
		status = &statusStr
	}

	// Get sessions from service
	result, err := h.sessionService.GetSessions(limit, offset, status, category)
	if err != nil {
		http.Error(w, "Failed to fetch sessions", http.StatusInternalServerError)
		return
	}

	// Convert to view data
	sessions := make([]SessionViewData, len(result.Items))
	for i, session := range result.Items {
		sessions[i] = SessionViewData{
			ID:               session.ID,
			Category:         session.Category,
			Task:             session.Task,
			Note:             utils.PtrToString(session.Note),
			Location:         utils.PtrToString(session.Location),
			Mood:             utils.PtrToString(session.Mood),
			DisplayStartTime: h.formatTime(session.StartedAt),
			DisplayEndTime:   h.formatTimePtr(session.EndedAt),
			Duration:         utils.FormatDuration(session.DurationSec),
			Status:           session.Status,
			StartedAt:        session.StartedAt,
			EndedAt:          session.EndedAt,
		}
	}

	// Calculate pagination
	totalPages := int((result.Total + int64(limit) - 1) / int64(limit))
	if totalPages < 1 {
		totalPages = 1
	}

	// Get current running session
	var runningSessionView *SessionViewData
	currentResp, err := h.sessionService.GetCurrent()
	if err == nil && currentResp.Running && currentResp.Session != nil {
		running := currentResp.Session
		runningSessionView = &SessionViewData{
			ID:               running.ID,
			Category:         running.Category,
			Task:             running.Task,
			Note:             utils.PtrToString(running.Note),
			Location:         utils.PtrToString(running.Location),
			Mood:             utils.PtrToString(running.Mood),
			DisplayStartTime: h.formatTime(running.StartedAt),
			Status:           running.Status,
			StartedAt:        running.StartedAt,
		}
	}

	data := map[string]interface{}{
		"Title":          "计时",
		"ActivePage":     "sessions",
		"Sessions":       sessions,
		"Category":       categoryStr,
		"Status":         statusStr,
		"CurrentPage":    page,
		"TotalPages":     totalPages,
		"PrevPage":       page - 1,
		"NextPage":       page + 1,
		"RunningSession": runningSessionView,
		"APIKey":         h.apiKey,
	}

	h.renderTemplate(w, r, h.sessionsTemplate, "base", data)
}

// WebStartSession handles POST /web/sessions/actions/start - starts a new session via web interface.
func (h *WebHandler) WebStartSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Category string  `json:"category"`
		Task     string  `json:"task"`
		Note     *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	startInput := models.SessionStart{
		Category: input.Category,
		Task:     input.Task,
		Note:     input.Note,
	}

	_, err := h.sessionService.StartSession(&startInput)
	if err != nil {
		if err == sessions.ErrSessionAlreadyRunning {
			http.Error(w, "Session already running", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// WebStopSession handles POST /web/sessions/actions/stop - stops the current session via web interface.
func (h *WebHandler) WebStopSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Body is empty for stop from web
	stopInput := &models.SessionStop{}

	_, err := h.sessionService.StopSession(stopInput)
	if err != nil {
		if err == sessions.ErrNoRunningSession {
			http.Error(w, "No running session found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// WebDeleteSession handles POST /web/sessions/actions/delete - deletes a session.
func (h *WebHandler) WebDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := h.sessionService.DeleteSession(input.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// WebUpdateSession handles POST /web/sessions/actions/update - updates a session.
func (h *WebHandler) WebUpdateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		ID int64 `json:"id"`
		models.SessionUpdate
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := h.sessionService.UpdateSession(input.ID, &input.SessionUpdate); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
