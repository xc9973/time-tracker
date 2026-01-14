package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"time-tracker/internal/handler"
	"time-tracker/internal/shared/auth"
	"time-tracker/internal/tags"
	"time-tracker/internal/shared/health"
	"time-tracker/internal/web"
)

// NewRouter creates and configures the HTTP router with all routes.
func NewRouter(
	cfg *Config,
	sessionsHandler *handler.SessionsHandler,
	tagsHandler *tags.TagsHandler,
	healthHandler *health.HealthHandler,
	webHandler *web.WebHandler,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoint (no authentication required)
	mux.Handle("/healthz", healthHandler)

	// API endpoints (require API key authentication)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// Session-tags association endpoints go to tags handler
		case strings.HasPrefix(path, "/api/v1/sessions/") && (strings.HasSuffix(path, "/tags") || strings.Contains(path, "/tags/")):
			tagsHandler.ServeHTTP(w, r)
		// Other sessions endpoints
		case strings.HasPrefix(path, "/api/v1/sessions"):
			sessionsHandler.ServeHTTP(w, r)
		// Tags endpoints
		case strings.HasPrefix(path, "/api/v1/tags"):
			tagsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	// Apply API key middleware to API routes (also allow Basic Auth for web interface)
	mux.Handle("/api/", auth.APIKeyMiddleware(cfg.APIKey, cfg.BasicUser, cfg.BasicPass)(apiHandler))

	// Web endpoints (require Basic Auth if configured)
	webMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webHandler.ServeHTTP(w, r)
	})

	// CSV export endpoints (also require Basic Auth if configured)
	csvHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/sessions.csv":
			sessionsHandler.ExportCSV(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	// Apply Basic Auth middleware if credentials are configured
	if cfg.BasicUser != "" && cfg.BasicPass != "" {
		mux.Handle("/web/", auth.BasicAuthMiddleware(cfg.BasicUser, cfg.BasicPass)(webMux))
		mux.Handle("/sessions.csv", auth.BasicAuthMiddleware(cfg.BasicUser, cfg.BasicPass)(csvHandler))
	} else {
		mux.Handle("/web/", webMux)
		mux.Handle("/sessions.csv", csvHandler)
	}

	// Redirect root path to /web/sessions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/web/sessions", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Static files from templates/static
	absTemplates, err := filepath.Abs("templates")
	if err == nil {
		staticPath := filepath.Join(absTemplates, "static")
		if _, err := os.Stat(staticPath); err == nil {
			mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))
		}
	}

	return mux
}
