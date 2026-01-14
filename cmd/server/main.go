// Package main provides the entry point for the time tracker server.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"time-tracker/internal/handler"

	"time-tracker/internal/shared/auth"
	"time-tracker/internal/shared/database"
	"time-tracker/internal/shared/middleware"
	"time-tracker/internal/sessions"
	"time-tracker/internal/shared/health"
	"time-tracker/internal/tags"
	"time-tracker/internal/web"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	APIKey     string
	DBPath     string
	Timezone   string
	BasicUser  string
	BasicPass  string
	RateLimit  int
	Port       string
}

// LoadConfig loads configuration from environment variables.
// Returns an error if required configuration is missing or invalid.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		APIKey:    os.Getenv("TIMELOG_API_KEY"),
		DBPath:    os.Getenv("TIMELOG_DB_PATH"),
		Timezone:  os.Getenv("TIMELOG_TZ"),
		BasicUser: os.Getenv("TIMELOG_BASIC_USER"),
		BasicPass: os.Getenv("TIMELOG_BASIC_PASS"),
		Port:      os.Getenv("TIMELOG_PORT"),
	}

	// Validate API key (required, minimum 32 characters)
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("TIMELOG_API_KEY is required")
	}
	if len(cfg.APIKey) < 32 {
		return nil, fmt.Errorf("TIMELOG_API_KEY must be at least 32 characters long")
	}

	// Set defaults
	if cfg.DBPath == "" {
		cfg.DBPath = "./timelog.db"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}
	if cfg.Port == "" {
		cfg.Port = "7070"
	}

	// Parse rate limit
	rateLimitStr := os.Getenv("TIMELOG_RATE_LIMIT")
	if rateLimitStr == "" {
		cfg.RateLimit = 100
	} else {
		rateLimit, err := strconv.Atoi(rateLimitStr)
		if err != nil || rateLimit <= 0 {
			return nil, fmt.Errorf("TIMELOG_RATE_LIMIT must be a positive integer")
		}
		cfg.RateLimit = rateLimit
	}

	return cfg, nil
}

// logStartup logs startup information without exposing sensitive values.
func logStartup(cfg *Config) {
	log.Println("Starting Time Tracker server...")
	log.Printf("Database path: %s", cfg.DBPath)
	log.Printf("Timezone: %s", cfg.Timezone)
	log.Printf("Rate limit: %d requests/minute", cfg.RateLimit)
	log.Printf("Port: %s", cfg.Port)

	// Log API key prefix only (first 4 characters for debugging)
	if len(cfg.APIKey) >= 4 {
		log.Printf("API Key: %s...", cfg.APIKey[:4])
	}

	// Log Basic Auth status without exposing credentials
	if cfg.BasicUser != "" && cfg.BasicPass != "" {
		log.Println("Basic Auth: enabled")
	} else {
		log.Println("Basic Auth: disabled (web interface unprotected)")
	}
}

func main() {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Log startup info (without sensitive values)
	logStartup(cfg)

	// Parse timezone
	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("Invalid timezone %s: %v", cfg.Timezone, err)
	}

	// Initialize database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized successfully")

	// Initialize repositories
	sessionRepo := sessions.NewSessionRepository(db)

	// Initialize services
	sessionService := sessions.NewSessionService(sessionRepo)

	// Initialize handlers
	sessionsHandler := handler.NewSessionsHandler(sessionService)
	tagsRepo := tags.NewTagRepository(db)
	tagsService := tags.NewTagService(tagsRepo)
	tagsHandler := tags.NewTagsHandler(tagsService)
	healthHandler := health.NewHealthHandler()
	absTemplates, err := filepath.Abs("templates")
	if err != nil {
		log.Fatalf("Failed to resolve templates path: %v", err)
	}
	webHandler, err := web.NewWebHandler(sessionService, absTemplates, tz, cfg.APIKey)
	if err != nil {
		log.Fatalf("Failed to initialize web handler: %v", err)
	}

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit)

	// Create main router
	mux := http.NewServeMux()

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

	// Health endpoint (no authentication required)
	mux.Handle("/healthz", healthHandler)

	// API endpoints (require API key authentication)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case strings.HasPrefix(path, "/api/v1/sessions"):
			sessionsHandler.ServeHTTP(w, r)
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

	staticPath := filepath.Join(absTemplates, "static")
	if _, err := os.Stat(staticPath); err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))
	}

	// Apply global middleware chain
	var finalHandler http.Handler = mux

	// Apply rate limiting
	finalHandler = middleware.RateLimitMiddleware(rateLimiter)(finalHandler)

	// Apply nonce middleware before security headers
	finalHandler = nonceMiddleware(finalHandler)

	// Apply security headers
	finalHandler = middleware.SecurityHeadersMiddleware(finalHandler)

	// Start server
	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: finalHandler,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Stop rate limiter cleanup goroutine
	rateLimiter.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
