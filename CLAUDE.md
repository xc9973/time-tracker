# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Time Tracker - a personal time logging system with REST API and web interface. Designed to work with iOS Shortcuts for quick start/stop timing.

## Commands

### Build & Run
```bash
# Build
go build ./cmd/server

# Run locally (requires .env with TIMELOG_API_KEY set)
export $(cat .env | xargs) && go run ./cmd/server

# Run with inline env (for testing)
TIMELOG_API_KEY=12345678901234567890123456789012 go run ./cmd/server
```

### Test
```bash
# All tests
go test ./...

# Single package tests
go test ./internal/errors

# Verbose
go test -v ./...
```

### Docker
```bash
docker build -t time-tracker .
docker run -p 8000:8000 -v $(pwd)/data:/data time-tracker
```

## Architecture

### Request Flow
```
HTTP Request → Middleware (RateLimit/Security/Nonce) → Auth (APIKey/BasicAuth)
→ Handler → Service → Repository → SQLite
```

### Key Components

**Entry Point** (`cmd/server/main.go`):
- Loads config from environment (TIMELOG_API_KEY, TIMELOG_DB_PATH, TIMELOG_TZ, etc.)
- Initializes SQLite DB with WAL mode and single-writer connection pool
- Sets up dependency chain: DB → Repository → Service → Handler
- Configures middleware chain: RateLimit → Nonce → SecurityHeaders
- Routes: `/api/v1/sessions/*` (API), `/web/*` (web UI), `/healthz`, `/sessions.csv`

**Service Layer** (`internal/service/`):
- Enforces business rules: only one running session at a time (returns `ErrSessionAlreadyRunning`)
- Calculates duration when stopping sessions
- Handles CSV export with UTF-8 BOM for Excel compatibility

**Repository Layer** (`internal/repository/`):
- Uses parameterized queries for SQL injection prevention
- `GetRunning()` returns nil (no error) when no session is running
- Sessions ordered by `started_at DESC` with indexes on status, category, started_at

**Authentication** (`internal/auth/`):
- API Key via `X-API-Key` header for API endpoints
- Basic Auth for web interface (`/web/*`) and CSV export
- Constant-time comparison (`subtle.ConstantTimeCompare`) for timing attack prevention
- API middleware allows either API Key OR Basic Auth (for web UI calling API)

**Database** (`internal/database/`):
- SQLite with foreign keys and WAL mode enabled
- Single-writer connection pool (`MaxOpenConns=1`) to avoid "database is locked" errors
- Tables: `sessions` with indexes on started_at, status, category

**Input Validation** (`internal/validation/`, `internal/models/`):
- Sanitization: trims whitespace, encodes HTML entities (`&<>` → `&amp;&lt;&gt;`)
- Length limits: Category (50), Task (200), Note (1000), Location (100), Mood (20)
- Defaults: empty category → "未分类", empty task → "未命名任务"

### Important Constraints

- **API Key**: Required, minimum 32 characters
- **SQLite Single Writer**: Only one concurrent write allowed due to SQLite constraints
- **Session State**: Only one session can be "running" at any time
- **Time Format**: All timestamps stored as RFC3339 UTC strings
- **Rate Limiting**: Sliding window per IP address, default 100 requests/minute

### Special Files

- `env.example`: Template for environment variables
- `templates/`: HTML templates for web interface with static assets in `templates/static/`
- `.kiro/specs/`: Project specs and design documents

### Error Handling

Custom error types in `internal/errors/`:
- `ValidationError` (400)
- `NotFoundError` (404)
- `ConflictError` (409) - includes current session info
- `UnauthorizedError` (401)
- `RateLimitError` (429) - includes Retry-After header
- `InternalError` (500) - generic, no details exposed