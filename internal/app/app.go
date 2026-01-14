package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"time-tracker/internal/handler"

	"time-tracker/internal/shared/database"
	"time-tracker/internal/shared/middleware"
	"time-tracker/internal/sessions"
	"time-tracker/internal/shared/health"
	"time-tracker/internal/tags"
	"time-tracker/internal/web"
)

// App holds the application dependencies and HTTP server.
type App struct {
	cfg         *Config
	db          *database.DB
	tz          *time.Location
	server      *http.Server
	rateLimiter *middleware.RateLimiter
}

// New creates and wires all application dependencies.
func New(cfg *Config) (*App, error) {
	// Parse timezone
	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", cfg.Timezone, err)
	}

	// Initialize database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize repositories
	sessionRepo := sessions.NewSessionRepository(db)
	tagsRepo := tags.NewTagRepository(db)

	// Initialize services
	sessionService := sessions.NewSessionService(sessionRepo)
	tagsService := tags.NewTagService(tagsRepo)

	// Initialize handlers
	sessionsHandler := handler.NewSessionsHandler(sessionService)
	tagsHandler := tags.NewTagsHandler(tagsService)
	healthHandler := health.NewHealthHandler()

	absTemplates, err := filepath.Abs("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve templates path: %w", err)
	}
	webHandler, err := web.NewWebHandler(sessionService, absTemplates, tz, cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web handler: %w", err)
	}

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit)

	// Create router with all routes
	mux := NewRouter(cfg, sessionsHandler, tagsHandler, healthHandler, webHandler)

	// Apply global middleware chain
	finalHandler := setupMiddlewareChain(mux, rateLimiter)

	return &App{
		cfg:         cfg,
		db:          db,
		tz:          tz,
		server: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: finalHandler,
		},
		rateLimiter: rateLimiter,
	}, nil
}

// setupMiddlewareChain creates the middleware chain in the correct order.
func setupMiddlewareChain(mux *http.ServeMux, rateLimiter *middleware.RateLimiter) http.Handler {
	var finalHandler http.Handler = mux

	// Apply rate limiting
	finalHandler = middleware.RateLimitMiddleware(rateLimiter)(finalHandler)

	// Apply nonce middleware (CSP)
	nonceMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonceBytes := make([]byte, 16)
			if _, err := rand.Read(nonceBytes); err != nil {
				http.Error(w, "failed to generate nonce", http.StatusInternalServerError)
				return
			}
			nonce := base64.StdEncoding.EncodeToString(nonceBytes)
			ctx := context.WithValue(r.Context(), middleware.CSPNonceKey{}, nonce)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	finalHandler = nonceMiddleware(finalHandler)

	// Apply security headers
	finalHandler = middleware.SecurityHeadersMiddleware(finalHandler)

	return finalHandler
}

// Run starts the HTTP server and blocks until shutdown.
func (a *App) Run() error {
	log.Printf("Server listening on %s", a.server.Addr)
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (a *App) Shutdown() error {
	log.Println("Shutting down server...")

	// Stop rate limiter cleanup goroutine
	a.rateLimiter.Stop()

	// Close database
	a.db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited properly")
	return nil
}
