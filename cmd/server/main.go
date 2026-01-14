// Package main provides the entry point for the time tracker server.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"time-tracker/internal/app"
)

// logStartup logs startup information without exposing sensitive values.
func logStartup(cfg *app.Config) {
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
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Log startup info (without sensitive values)
	logStartup(cfg)

	// Create and wire application
	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := a.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Shutdown the server
	if err := a.Shutdown(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
}
