# Phase 1: Establish Shared Structure

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 创建 `internal/shared/` 目录，移动共享组件（database, auth, middleware, errors, validation）到新位置，建立 `internal/app/` 框架。

**Architecture:** 将横向关注点（cross-cutting concerns）集中到 shared 目录，为模块化重构做准备。不影响现有功能，只是移动和更新 import 路径。

**Tech Stack:** Go 1.21+, SQLite, net/http

---

## Task 1: Create shared directory structure

**Files:**
- Create: `internal/shared/database/`
- Create: `internal/shared/auth/`
- Create: `internal/shared/middleware/`
- Create: `internal/shared/errors/`
- Create: `internal/shared/validation/`
- Create: `internal/shared/config/`
- Create: `internal/shared/utils/`

**Step 1: Create all shared directories**

```bash
mkdir -p internal/shared/{database,auth,middleware,errors,validation,config,utils}
```

**Step 2: Verify directories created**

Run: `ls -la internal/shared/`
Expected: List of 7 directories

**Step 3: Commit**

```bash
git add internal/shared/
git commit -m "phase1: create shared directory structure"
```

---

## Task 2: Move database module to shared

**Files:**
- Move: `internal/database/database.go` → `internal/shared/database/database.go`
- Move: `internal/database/database_test.go` → `internal/shared/database/database_test.go`
- Modify: `internal/shared/database/database.go` (update package name)

**Step 1: Move database files**

```bash
git mv internal/database internal/shared/database
```

**Step 2: Update package declaration**

In `internal/shared/database/database.go`, change:
```go
package database
```

**Step 3: Update imports in database_test.go**

In `internal/shared/database/database_test.go`, change:
```go
package database

import (
    "testing"
    "time-tracker/internal/shared/database"
)
```

**Step 4: Run database tests**

Run: `go test ./internal/shared/database/ -v`
Expected: PASS

**Step 5: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/database|time-tracker/internal/shared/database|g' {} \;`

**Step 6: Verify build**

Run: `go build ./cmd/server`
Expected: Success, no errors

**Step 7: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 8: Commit**

```bash
git add -A
git commit -m "phase1: move database to shared directory"
```

---

## Task 3: Move auth module to shared

**Files:**
- Move: `internal/auth/auth.go` → `internal/shared/auth/auth.go`
- Move: `internal/auth/auth_test.go` → `internal/shared/auth/auth_test.go`

**Step 1: Move auth files**

```bash
git mv internal/auth internal/shared/auth
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/auth|time-tracker/internal/shared/auth|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Run auth tests**

Run: `go test ./internal/shared/auth/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "phase1: move auth to shared directory"
```

---

## Task 4: Move middleware module to shared

**Files:**
- Move: `internal/middleware/` → `internal/shared/middleware/`

**Step 1: Move middleware directory**

```bash
git mv internal/middleware internal/shared/middleware
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/middleware|time-tracker/internal/shared/middleware|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Run middleware tests**

Run: `go test ./internal/shared/middleware/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "phase1: move middleware to shared directory"
```

---

## Task 5: Move errors module to shared

**Files:**
- Move: `internal/errors/errors.go` → `internal/shared/errors/errors.go`
- Move: `internal/errors/errors_test.go` → `internal/shared/errors/errors_test.go`
- Move: `internal/errors/errors_property_test.go` → `internal/shared/errors/errors_property_test.go`

**Step 1: Move error files**

```bash
git mv internal/errors internal/shared/errors
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/errors|time-tracker/internal/shared/errors|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Run error tests**

Run: `go test ./internal/shared/errors/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "phase1: move errors to shared directory"
```

---

## Task 6: Move validation module to shared

**Files:**
- Move: `internal/validation/validation.go` → `internal/shared/validation/validation.go`
- Move: `internal/validation/validation_test.go` → `internal/shared/validation/validation_test.go`

**Step 1: Move validation files**

```bash
git mv internal/validation internal/shared/validation
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/validation|time-tracker/internal/shared/validation|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Run validation tests**

Run: `go test ./internal/shared/validation/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "phase1: move validation to shared directory"
```

---

## Task 7: Move utils module to shared

**Files:**
- Move: `internal/utils/utils.go` → `internal/shared/utils/utils.go`
- Move: `internal/utils/db_utils.go` → `internal/shared/utils/db_utils.go`

**Step 1: Move utils files**

```bash
git mv internal/utils internal/shared/utils
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/utils|time-tracker/internal/shared/utils|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Commit**

```bash
git add -A
git commit -m "phase1: move utils to shared directory"
```

---

## Task 8: Move config module to shared

**Files:**
- Move: `internal/config/constants.go` → `internal/shared/config/constants.go`

**Step 1: Move config files**

```bash
git mv internal/config internal/shared/config
```

**Step 2: Update all imports**

Run: `find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/config|time-tracker/internal/shared/config|g' {} \;`

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Commit**

```bash
git add -A
git commit -m "phase1: move config to shared directory"
```

---

## Task 9: Create app framework skeleton

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/router.go`

**Step 1: Write the basic app structure**

Create `internal/app/app.go`:

```go
package app

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "time-tracker/internal/shared/config"
    "time-tracker/internal/shared/middleware"
)

// Config holds application configuration
type Config struct {
    APIKey    string
    DBPath    string
    Timezone  string
    BasicUser string
    BasicPass string
    RateLimit int
    Port      string
}

// App represents the application
type App struct {
    db     *sql.DB
    cfg    *Config
    router http.Handler
}

// New creates a new application instance
func New(cfg *Config) (*App, error) {
    // TODO: Initialize database, services, handlers
    return &App{
        cfg: cfg,
    }, nil
}

// Run starts the application
func (a *App) Run() error {
    addr := ":" + a.cfg.Port
    log.Printf("Server listening on %s", addr)
    return http.ListenAndServe(addr, a.router)
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
    log.Println("Shutting down...")
    if a.db != nil {
        return a.db.Close()
    }
    return nil
}

// LoadConfig loads configuration from environment
func LoadConfig() (*Config, error) {
    cfg := &Config{
        APIKey:    getEnv("TIMELOG_API_KEY", ""),
        DBPath:    getEnv("TIMELOG_DB_PATH", "./timelog.db"),
        Timezone:  getEnv("TIMELOG_TZ", "UTC"),
        BasicUser: getEnv("TIMELOG_BASIC_USER", ""),
        BasicPass: getEnv("TIMELOG_BASIC_PASS", ""),
        Port:      getEnv("TIMELOG_PORT", "7070"),
    }

    // Validate API key
    if cfg.APIKey == "" || len(cfg.APIKey) < 32 {
        return nil, config.ErrInvalidConfig{Message: "TIMELOG_API_KEY required (min 32 chars)"}
    }

    // Parse rate limit
    cfg.RateLimit = config.DefaultRateLimit
    if rateLimitStr := getEnv("TIMELOG_RATE_LIMIT", ""); rateLimitStr != "" {
        if rateLimit, err := parseRateLimit(rateLimitStr); err == nil {
            cfg.RateLimit = rateLimit
        }
    }

    return cfg, nil
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}

func parseRateLimit(s string) (int, error) {
    return strconv.Atoi(s)
}
```

**Step 2: Write router skeleton**

Create `internal/app/router.go`:

```go
package app

import (
    "net/http"
)

// NewRouter creates the main HTTP router
// TODO: Add services as parameters when implemented
func NewRouter(cfg *Config) http.Handler {
    mux := http.NewServeMux()

    // Health check
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // TODO: Register API handlers
    // TODO: Apply middleware

    return mux
}
```

**Step 3: Verify build**

Run: `go build ./internal/app/`
Expected: Success

**Step 4: Commit**

```bash
git add internal/app/
git commit -m "phase1: create app framework skeleton"
```

---

## Task 10: Verify Phase 1 completion

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 3: Verify directory structure**

Run: `tree internal/ -L 2`
Expected Output:
```
internal/
├── app/
├── handler/
├── models/
├── repository/
├── service/
└── shared/
    ├── auth/
    ├── config/
    ├── database/
    ├── errors/
    ├── middleware/
    ├── utils/
    └── validation/
```

**Step 4: Commit phase 1 completion marker**

```bash
echo "# Phase 1 Complete

Shared structure established:
- All cross-cutting concerns moved to internal/shared/
- internal/app/ framework created
- All imports updated
- All tests passing" > docs/phases/phase1-complete.md

git add docs/phases/phase1-complete.md
git commit -m "phase1: complete - shared structure established"
```

---

## Phase 1 Completion Checklist

- [ ] All shared modules moved to `internal/shared/`
- [ ] All imports updated
- [ ] `internal/app/` framework created
- [ ] All tests passing (`go test ./...`)
- [ ] Build successful (`go build ./cmd/server`)
- [ ] Git worktree ready for Phase 2
