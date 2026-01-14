# Phase 2: Migrate Sessions Module

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 sessions 相关代码从分层架构（models/repository/service/handler）合并到单一的 `internal/sessions/` 模块中。

**Architecture:** 每个功能模块包含完整的垂直切片（handler → service → repository → models），消除跨层查找代码的问题。

**Tech Stack:** Go 1.21+, SQLite, net/http

---

## Task 1: Create sessions module directory

**Files:**
- Create: `internal/sessions/`

**Step 1: Create sessions directory**

```bash
mkdir -p internal/sessions
```

**Step 2: Verify directory created**

Run: `ls -la internal/sessions/`
Expected: Empty directory

**Step 3: Commit**

```bash
git add internal/sessions/
git commit -m "phase2: create sessions module directory"
```

---

## Task 2: Move models to sessions module

**Files:**
- Move: `internal/models/models.go` → `internal/sessions/models.go`
- Move: `internal/models/models_test.go` → `internal/sessions/models_test.go`
- Modify: `internal/sessions/models.go` (update package name and imports)

**Step 1: Move models files**

```bash
git mv internal/models/models.go internal/sessions/models.go
git mv internal/models/models_test.go internal/sessions/models_test.go 2>/dev/null || echo "models_test.go not found"
```

**Step 2: Update package declaration**

In `internal/sessions/models.go`, change line 2:
```go
package sessions
```

**Step 3: Update imports in models.go**

Replace line 8-9:
```go
"time-tracker/internal/shared/config"
"time-tracker/internal/shared/validation"
```

**Step 4: Update package in models_test.go**

In `internal/sessions/models_test.go`, change:
```go
package sessions
```

And update imports:
```go
"time-tracker/internal/sessions"
```

**Step 5: Run models tests**

Run: `go test ./internal/sessions/models_test.go -v`
Expected: Tests pass

**Step 6: Update all imports across codebase**

```bash
find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/models|time-tracker/internal/sessions|g' {} \;
```

**Step 7: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 8: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 9: Commit**

```bash
git add -A
git commit -m "phase2: move models to sessions module"
```

---

## Task 3: Move repository to sessions module

**Files:**
- Move: `internal/repository/session_repository.go` → `internal/sessions/repository.go`
- Move: `internal/repository/interfaces.go` → `internal/sessions/interfaces.go`
- Move: `internal/repository/*_test.go` → `internal/sessions/*_test.go`

**Step 1: Move repository files**

```bash
# Create sessions directory if not exists
mkdir -p internal/sessions

# Move session repository
git mv internal/repository/session_repository.go internal/sessions/repository.go
git mv internal/repository/interfaces.go internal/sessions/repository_interfaces.go 2>/dev/null || true

# Move tests
git mv internal/repository/*_test.go internal/sessions/ 2>/dev/null || echo "No tests to move"
```

**Step 2: Update package declaration**

In `internal/sessions/repository.go`, change line 1:
```go
package sessions
```

**Step 3: Update imports in repository.go**

Replace lines 10-12:
```go
"time-tracker/internal/shared/database"
"time-tracker/internal/sessions"
"time-tracker/internal/shared/utils"
```

**Step 4: Update database.DB type reference**

In `internal/sessions/repository.go`, line 20, change:
```go
type SessionRepository struct {
    db *database.DB
}
```

**Step 5: Update NewSessionRepository signature**

In `internal/sessions/repository.go`, line 24, change:
```go
func NewSessionRepository(db *database.DB) *SessionRepository {
    return &SessionRepository{db: db}
}
```

**Step 6: Update repository tests**

In `internal/sessions/*_test.go`, update package:
```go
package sessions
```

Update imports in tests:
```go
"time-tracker/internal/sessions"
"time-tracker/internal/shared/database"
```

**Step 7: Run repository tests**

Run: `go test ./internal/sessions/... -v`
Expected: Tests pass

**Step 8: Update all imports across codebase**

```bash
find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/repository|time-tracker/internal/sessions|g' {} \;
```

**Step 9: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 10: Commit**

```bash
git add -A
git commit -m "phase2: move repository to sessions module"
```

---

## Task 4: Move service to sessions module

**Files:**
- Move: `internal/service/session_service.go` → `internal/sessions/service.go`
- Move: `internal/service/interfaces.go` → `internal/sessions/service_interfaces.go`
- Move: `internal/service/*_test.go` → `internal/sessions/*_test.go`

**Step 1: Move service files**

```bash
git mv internal/service/session_service.go internal/sessions/service.go
git mv internal/service/interfaces.go internal/sessions/service_interfaces.go 2>/dev/null || true
git mv internal/service/*_test.go internal/sessions/ 2>/dev/null || true
```

**Step 2: Update package declaration**

In `internal/sessions/service.go`, change line 2:
```go
package sessions
```

**Step 3: Update imports in service.go**

Replace lines 10-13:
```go
"time-tracker/internal/shared/config"
"time-tracker/internal/sessions"
"time-tracker/internal/sessions"
"time-tracker/internal/shared/utils"
```

**Step 4: Update SessionService struct**

In `internal/sessions/service.go`, line 30-32, change:
```go
type SessionService struct {
    repo *SessionRepository
}
```

**Step 5: Update NewSessionService**

In `internal/sessions/service.go`, line 35-39, change:
```go
func NewSessionService(repo *SessionRepository) *SessionService {
    return &SessionService{
        repo: repo,
    }
}
```

**Step 6: Update repository references**

Throughout `internal/sessions/service.go`, replace `s.repo.` calls - no change needed, just verify all references are correct.

**Step 7: Update service tests**

In `internal/sessions/*_test.go`, update:
```go
package sessions
```

Update imports:
```go
"time-tracker/internal/sessions"
```

**Step 8: Run service tests**

Run: `go test ./internal/sessions/... -v`
Expected: Tests pass

**Step 9: Update all imports across codebase**

```bash
find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/service|time-tracker/internal/sessions|g' {} \;
```

**Step 10: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 11: Commit**

```bash
git add -A
git commit -m "phase2: move service to sessions module"
```

---

## Task 5: Move handler to sessions module

**Files:**
- Move: `internal/handler/sessions.go` → `internal/sessions/handler.go`
- Move: `internal/handler/handler_test.go` → `internal/sessions/handler_test.go`

**Step 1: Move handler files**

```bash
git mv internal/handler/sessions.go internal/sessions/handler.go
git mv internal/handler/handler_test.go internal/sessions/handler_test.go 2>/dev/null || true
```

**Step 2: Update package declaration**

In `internal/sessions/handler.go`, change line 1:
```go
package sessions
```

**Step 3: Update imports in handler.go**

Replace lines 10-15:
```go
"time-tracker/internal/shared/config"
"time-tracker/internal/shared/errors"
"time-tracker/internal/sessions"
"time-tracker/internal/sessions"
"time-tracker/internal/shared/utils"
"time-tracker/internal/shared/validation"
```

**Step 4: Update SessionsHandler struct**

In `internal/sessions/handler.go`, line 19-21:
```go
type SessionsHandler struct {
    service *SessionService
}
```

**Step 5: Update NewSessionsHandler**

In `internal/sessions/handler.go`, line 24-26:
```go
func NewSessionsHandler(svc *SessionService) *SessionsHandler {
    return &SessionsHandler{service: svc}
}
```

**Step 6: Update handler tests**

In `internal/sessions/handler_test.go`, update:
```go
package sessions
```

**Step 7: Run handler tests**

Run: `go test ./internal/sessions/... -v`
Expected: Tests pass

**Step 8: Update all imports across codebase**

```bash
find . -name '*.go' -type f -exec sed -i '' 's|time-tracker/internal/handler|time-tracker/internal/sessions|g' {} \;
```

**Step 9: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 10: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

**Step 11: Commit**

```bash
git add -A
git commit -m "phase2: move handler to sessions module"
```

---

## Task 6: Update main.go imports

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Update imports in main.go**

Replace the imports section (lines 19-24) with:
```go
"time-tracker/internal/sessions"
"time-tracker/internal/shared/auth"
"time-tracker/internal/shared/config"
"time-tracker/internal/shared/database"
"time-tracker/internal/shared/middleware"
```

**Step 2: Update repository initialization**

In main.go around line 130, change:
```go
sessionRepo := sessions.NewSessionRepository(db)
```

**Step 3: Update service initialization**

In main.go around line 133, change:
```go
sessionService := sessions.NewSessionService(sessionRepo)
```

**Step 4: Update handler initialization**

In main.go around line 136, change:
```go
sessionsHandler := sessions.NewSessionsHandler(sessionService)
```

**Step 5: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 6: Run server locally**

Run: `TIMELOG_API_KEY=test123456789012345678901234567890 ./server`
Expected: Server starts successfully

Stop server with Ctrl+C.

**Step 7: Commit**

```bash
git add cmd/server/main.go
git commit -m "phase2: update main.go imports for sessions module"
```

---

## Task 7: Clean up old directories

**Files:**
- Remove: `internal/models/` (empty)
- Remove: `internal/repository/` (empty)
- Remove: `internal/service/` (empty)
- Remove: `internal/handler/` (may still have web.go, health.go)

**Step 1: Check what's left in old directories**

Run: `ls -la internal/models/ internal/repository/ internal/service/ internal/handler/`
Expected: These should be empty or only contain non-session files

**Step 2: Remove empty directories**

```bash
# Check if directories are empty
if [ -z "$(ls -A internal/models)" ]; then git rm -r internal/models; fi
if [ -z "$(ls -A internal/repository)" ]; then git rm -r internal/repository; fi
if [ -z "$(ls -A internal/service)" ]; then git rm -r internal/service; fi
```

Note: Don't remove `internal/handler/` yet - it still has web.go and health.go

**Step 3: Verify build**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Run tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "phase2: remove empty old directories"
```

---

## Task 8: Verify sessions module structure

**Step 1: List sessions module files**

Run: `ls -la internal/sessions/`
Expected Output:
```
internal/sessions/
├── models.go
├── repository.go
├── service.go
├── handler.go
└── *_test.go files
```

**Step 2: Verify module tests**

Run: `go test ./internal/sessions/... -v -cover`
Expected: All PASS with coverage

**Step 3: Integration test - start a session**

Run:
```bash
TIMELOG_API_KEY=test123456789012345678901234567890 ./server &
SERVER_PID=$!
sleep 2

# Test starting a session
curl -X POST http://localhost:7070/api/v1/sessions/start \
  -H "X-API-Key: test123456789012345678901234567890" \
  -H "Content-Type: application/json" \
  -d '{"category":"测试","task":"测试任务"}'

# Stop server
kill $SERVER_PID
```

Expected: Session created successfully response

**Step 4: Commit phase 2 completion marker**

```bash
echo "# Phase 2 Complete

Sessions module migrated:
- All session code in internal/sessions/
- Contains: models, repository, service, handler
- All tests passing
- Integration test successful" > docs/phases/phase2-complete.md

git add docs/phases/phase2-complete.md
git commit -m "phase2: complete - sessions module migrated"
```

---

## Phase 2 Completion Checklist

- [ ] `internal/sessions/` created with all code
- [ ] `models.go` moved and package updated
- [ ] `repository.go` moved and package updated
- [ ] `service.go` moved and package updated
- [ ] `handler.go` moved and package updated
- [ ] All imports updated across codebase
- [ ] `main.go` updated
- [ ] Old directories removed (except handler with web/health)
- [ ] All tests passing (`go test ./...`)
- [ ] Build successful (`go build ./cmd/server`)
- [ ] Integration test successful
