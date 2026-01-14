// Package handler provides HTTP handlers for the time tracker API.
package handler
import (
	"fmt"
	"html/template"
	"net/http"
	"time"
	"time-tracker/internal/sessions"

	"time-tracker/internal/shared/middleware"
)
// WebHandler handles HTTP requests for web interface.
type WebHandler struct {
	sessionService   *sessions.SessionService
	sessionsTemplate *template.Template
	timezone         *time.Location
	apiKey           string
}
// SessionViewData represents a session for display in templates.
type SessionViewData struct {
	ID               int64
	Category         string
	Task             string
	Note             string
	Location         string
	Mood             string
	DisplayStartTime string
	DisplayEndTime   string
	Duration         string
	Status           string
	StartedAt        string
	EndedAt          *string
}
// SessionsPageData represents the data for the sessions page template.
type SessionsPageData struct {
	Title          string
	ActivePage     string
	Sessions       []SessionViewData
	Category       string
	Status         string
	CurrentPage    int
	TotalPages     int
	PrevPage       int
	NextPage       int
	RunningSession *SessionViewData
	Categories     []string
	APIKey         string
}
// NewWebHandler creates a new WebHandler.
func NewWebHandler(sessionSvc *sessions.SessionService, templatesPath string, tz *time.Location, apiKey string) (*WebHandler, error) {
	sessionsTmpl, err := template.ParseFiles(templatesPath+"/base.html", templatesPath+"/sessions.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse sessions template: %w", err)
	}
	if tz == nil {
		tz = time.UTC
	}
	return &WebHandler{
		sessionService:   sessionSvc,
		sessionsTemplate: sessionsTmpl,
		timezone:         tz,
		apiKey:           apiKey,
	}, nil
}
// renderTemplate renders a template with the given data.
func (h *WebHandler) renderTemplate(w http.ResponseWriter, r *http.Request, tmpl *template.Template, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pageData, ok := data.(map[string]interface{})
	if !ok {
		pageData = map[string]interface{}{}
	}
	if nonce, ok := r.Context().Value(middleware.CSPNonceKey{}).(string); ok {
		pageData["ScriptNonce"] = nonce
	}
	if err := tmpl.ExecuteTemplate(w, templateName, pageData); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}
// formatTime converts an RFC3339 UTC timestamp to the configured timezone.
func (h *WebHandler) formatTime(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.In(h.timezone).Format("2006-01-02 15:04")
}
// formatTimePtr formats a time pointer, returning empty string for nil.
func (h *WebHandler) formatTimePtr(rfc3339 *string) string {
	if rfc3339 == nil {
		return ""
	}
	return h.formatTime(*rfc3339)
}
// ServeHTTP implements http.Handler for routing web requests.
func (h *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch path {
	case "/web/sessions":
		h.Sessions(w, r)
	case "/web/sessions/actions/start":
		h.WebStartSession(w, r)
	case "/web/sessions/actions/stop":
		h.WebStopSession(w, r)
	case "/web/sessions/actions/delete":
		h.WebDeleteSession(w, r)
	case "/web/sessions/actions/update":
		h.WebUpdateSession(w, r)
	default:
		http.NotFound(w, r)
	}
}
