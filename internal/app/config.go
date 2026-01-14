package app

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	APIKey    string
	DBPath    string
	Timezone  string
	BasicUser string
	BasicPass string
	RateLimit int
	Port      string
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
