# Tags 模块（Phase 4A）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增 `internal/tags` 模块，实现标签 CRUD，并提供 Session 与 Tag 的关联能力（多对多）。

**Architecture:** 以“模块化单体（feature-based）”组织：`tags` 模块包含 handler/service/repository/models（或最小必要文件），复用现有 shared（database/errors/validation/utils/auth/middleware）。通过新增表 `tags` 和关联表 `session_tags` 支撑功能。尽量保持最小 diff：先实现 API（tags CRUD + session_tags 绑定），再增量扩展 Web/统计。

**Tech Stack:** Go, SQLite, net/http, encoding/json, pgregory.net/rapid (property tests)

---

## 前置检查（必做）

**Step 1: 基线测试通过**

Run: `go test ./...`
Expected: PASS

**Step 2: 确认 DB 初始化入口位置**

当前表初始化在 `internal/database/database.go:initTables`（或迁移后 `internal/shared/database/database.go`）。本阶段需要在同一处追加 tags/session_tags 建表语句。

---

## Task 1: 定义 tags 数据模型（最小）

**Files:**
- Create: `internal/tags/models.go`

**Step 1: 写失败测试（模型/校验）**

Create `internal/tags/models_test.go`:

```go
package tags

import "testing"

func TestTag_Validate_NameRequired(t *testing.T) {
    tag := TagCreate{Name: "   "}
    if err := tag.Validate(); err == nil {
        t.Fatalf("expected validation error")
    }
}
```

Expected: FAIL（TagCreate/Validate 未定义）

**Step 2: 运行测试确认失败**

Run: `go test ./internal/tags -run TestTag_Validate_NameRequired -v`
Expected: FAIL

**Step 3: 写最小实现**

Create `internal/tags/models.go`:

```go
package tags

import (
    "errors"
    "strings"

    "time-tracker/internal/validation"
)

type Tag struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Color     string `json:"color"`
    CreatedAt string `json:"created_at"`
}

type TagCreate struct {
    Name  string `json:"name"`
    Color string `json:"color"`
}

var ErrNameRequired = errors.New("name is required")

func (t *TagCreate) Validate() error {
    t.Name = validation.SanitizeString(t.Name)
    t.Color = strings.TrimSpace(t.Color)
    if t.Name == "" {
        return ErrNameRequired
    }
    if t.Color == "" {
        t.Color = "#6B7280"
    }
    return nil
}
```

**Step 4: 重新跑测试**

Run: `go test ./internal/tags -run TestTag_Validate_NameRequired -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tags/models.go internal/tags/models_test.go
git commit -m "phase4(tags): add tag models and validation"
```

---

## Task 2: 扩展数据库初始化（tags + session_tags）

**Files:**
- Modify: `internal/database/database.go`（或迁移后 `internal/shared/database/database.go`）
- Test: `internal/database/database_test.go`

**Step 1: 写失败测试（检查新表存在）**

在 `internal/database/database_test.go` 增加：

```go
func TestNew_CreatesTagsTables(t *testing.T) {
    // create db as in existing tests
    // assert tags table exists
    // assert session_tags table exists
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/database -run TestNew_CreatesTagsTables -v`
Expected: FAIL（表不存在）

**Step 3: 写最小实现（追加建表语句）**

在 `initTables()` 中追加：

```sql
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6B7280',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

CREATE TABLE IF NOT EXISTS session_tags (
    session_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (session_id, tag_id),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_tags_session ON session_tags(session_id);
CREATE INDEX IF NOT EXISTS idx_session_tags_tag ON session_tags(tag_id);
```

**Step 4: 重新跑测试**

Run: `go test ./internal/database -run TestNew_CreatesTagsTables -v`
Expected: PASS

**Step 5: 全量测试**

Run: `go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/database/database.go internal/database/database_test.go
git commit -m "phase4(tags): add tags tables to sqlite init"
```

---

## Task 3: 实现 tags repository（SQLite）

**Files:**
- Create: `internal/tags/repository.go`
- Test: `internal/tags/repository_test.go`

**Step 1: 写失败测试（创建+查询）**

Create `internal/tags/repository_test.go`:

```go
package tags

import (
    "os"
    "testing"
    "time-tracker/internal/database"
)

func TestTagRepository_CreateAndList(t *testing.T) {
    tmp, err := os.CreateTemp("", "tags_repo_*.db")
    if err != nil { t.Fatal(err) }
    tmp.Close()
    defer os.Remove(tmp.Name())

    db, err := database.New(tmp.Name())
    if err != nil { t.Fatal(err) }
    defer db.Close()

    repo := NewTagRepository(db)

    created, err := repo.Create(&TagCreate{Name: "工作", Color: "#3B82F6"})
    if err != nil { t.Fatal(err) }
    if created.ID == 0 { t.Fatalf("expected id") }

    items, err := repo.List()
    if err != nil { t.Fatal(err) }
    if len(items) != 1 { t.Fatalf("expected 1, got %d", len(items)) }
}
```

Expected: FAIL

**Step 2: 运行测试确认失败**

Run: `go test ./internal/tags -run TestTagRepository_CreateAndList -v`
Expected: FAIL

**Step 3: 写最小实现**

Create `internal/tags/repository.go`:

```go
package tags

import (
    "database/sql"
    "fmt"

    "time-tracker/internal/database"
)

type TagRepository struct {
    db *database.DB
}

func NewTagRepository(db *database.DB) *TagRepository {
    return &TagRepository{db: db}
}

func (r *TagRepository) Create(input *TagCreate) (*Tag, error) {
    res, err := r.db.Exec(
        `INSERT INTO tags (name, color, created_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ','now'))`,
        input.Name, input.Color,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to insert tag: %w", err)
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("failed to get last insert id: %w", err)
    }
    return r.GetByID(id)
}

func (r *TagRepository) GetByID(id int64) (*Tag, error) {
    var t Tag
    err := r.db.QueryRow(`SELECT id, name, color, created_at FROM tags WHERE id = ?`, id).
        Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to query tag: %w", err)
    }
    return &t, nil
}

func (r *TagRepository) List() ([]Tag, error) {
    rows, err := r.db.Query(`SELECT id, name, color, created_at FROM tags ORDER BY name ASC`)
    if err != nil {
        return nil, fmt.Errorf("failed to query tags: %w", err)
    }
    defer rows.Close()

    out := []Tag{}
    for rows.Next() {
        var t Tag
        if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan tag: %w", err)
        }
        out = append(out, t)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("tags rows error: %w", err)
    }
    return out, nil
}
```

**Step 4: 重新跑测试**

Run: `go test ./internal/tags -run TestTagRepository_CreateAndList -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tags/repository.go internal/tags/repository_test.go
git commit -m "phase4(tags): add tag repository"
```

---

## Task 4: 实现 tags service（业务规则）

**Files:**
- Create: `internal/tags/service.go`
- Test: `internal/tags/service_test.go`

**Step 1: 写失败测试（重复名称返回冲突）**

Create `internal/tags/service_test.go`:

```go
package tags

import (
    "testing"
)

func TestTagService_DuplicateName(t *testing.T) {
    // use in-memory sqlite temp db like repo tests
    // create tag twice with same name
    // expect conflict-like error
}
```

Expected: FAIL

**Step 2: 最小实现**

Create `internal/tags/service.go`:

```go
package tags

import "fmt"

type TagService struct {
    repo *TagRepository
}

func NewTagService(repo *TagRepository) *TagService {
    return &TagService{repo: repo}
}

func (s *TagService) Create(input *TagCreate) (*Tag, error) {
    if err := input.Validate(); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    return s.repo.Create(input)
}

func (s *TagService) List() ([]Tag, error) {
    return s.repo.List()
}

func (s *TagService) Get(id int64) (*Tag, error) {
    return s.repo.GetByID(id)
}
```

**Step 3: 跑测试（补齐断言）**

Run: `go test ./internal/tags -run TestTagService_DuplicateName -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/tags/service.go internal/tags/service_test.go
git commit -m "phase4(tags): add tag service"
```

---

## Task 5: 实现 tags handler（HTTP API）

**Files:**
- Create: `internal/tags/handler.go`
- Test: `internal/tags/handler_test.go`

**Endpoints:**
- POST `/api/v1/tags`
- GET `/api/v1/tags`
- GET `/api/v1/tags/:id`

**Step 1: 写失败测试（POST+GET）**

Create `internal/tags/handler_test.go` (httptest + temp db):

```go
package tags

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestTagsHandler_CreateAndList(t *testing.T) {
    // setup db + repo + service + handler
    // POST /api/v1/tags
    // GET /api/v1/tags
}
```

Expected: FAIL

**Step 2: 最小实现 handler**

Create `internal/tags/handler.go`:

```go
package tags

import (
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    "time-tracker/internal/errors"
)

type TagsHandler struct {
    service *TagService
}

func NewTagsHandler(svc *TagService) *TagsHandler {
    return &TagsHandler{service: svc}
}

func (h *TagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    switch {
    case path == "/api/v1/tags" && r.Method == http.MethodPost:
        h.Create(w, r)
    case path == "/api/v1/tags" && r.Method == http.MethodGet:
        h.List(w, r)
    case strings.HasPrefix(path, "/api/v1/tags/") && r.Method == http.MethodGet:
        h.Get(w, r)
    default:
        errors.WriteError(w, errors.NotFoundError("Endpoint not found"))
    }
}

func (h *TagsHandler) Create(w http.ResponseWriter, r *http.Request) {
    var input TagCreate
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        errors.WriteError(w, errors.ValidationError("Invalid JSON body"))
        return
    }
    created, err := h.service.Create(&input)
    if err != nil {
        if strings.Contains(err.Error(), "validation error") {
            errors.WriteError(w, errors.ValidationError(strings.TrimPrefix(err.Error(), "validation error: ")))
            return
        }
        errors.WriteError(w, err)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(created)
}

func (h *TagsHandler) List(w http.ResponseWriter, r *http.Request) {
    items, err := h.service.List()
    if err != nil {
        errors.WriteError(w, err)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(items)
}

func (h *TagsHandler) Get(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tags/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || id <= 0 {
        errors.WriteError(w, errors.ValidationError("Invalid id"))
        return
    }
    t, err := h.service.Get(id)
    if err != nil {
        errors.WriteError(w, err)
        return
    }
    if t == nil {
        errors.WriteError(w, errors.NotFoundError("Tag not found"))
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(t)
}
```

**Step 3: 运行 handler 测试**

Run: `go test ./internal/tags -run TestTagsHandler_CreateAndList -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/tags/handler.go internal/tags/handler_test.go
git commit -m "phase4(tags): add tags http handler"
```

---

## Task 6: Session ↔ Tag 关联（session_tags）API

**Files:**
- Modify: `internal/tags/repository.go`
- Modify: `internal/tags/service.go`
- Modify: `internal/tags/handler.go` (或 `internal/sessions/handler.go`，按你偏好)
- Test: `internal/tags/*_test.go`

**Endpoints（推荐挂在 sessions 下）**
- POST `/api/v1/sessions/:id/tags` body: `{ "tag_ids": [1,2] }`
- DELETE `/api/v1/sessions/:id/tags/:tag_id`

**Step 1: 写失败测试（绑定后能查到关联）**

Create test skeleton in `internal/tags/repository_test.go`:

```go
func TestSessionTags_AssignAndList(t *testing.T) {
    // create session
    // create tags
    // assign
    // query join
}
```

**Step 2: 实现 repository 方法**

在 `internal/tags/repository.go` 增加：
- `AssignToSession(sessionID int64, tagIDs []int64) error`
- `RemoveFromSession(sessionID, tagID int64) error`

**Step 3: 实现 service 方法（输入校验 + 去重）**

在 `internal/tags/service.go` 增加：
- `AssignToSession(sessionID int64, tagIDs []int64) error`

**Step 4: 暴露 HTTP 路由**

最小做法：在 `internal/handler/sessions.go`（或 `internal/sessions/handler.go` 迁移后）增加两个 case 分支调用 tags service。

**Step 5: 跑全量测试**

Run: `go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add -A
git commit -m "phase4(tags): add session tag association endpoints"
```

---

## Done 标准

- `internal/tags/` 模块存在并可独立测试
- DB 初始化包含 `tags` 与 `session_tags` 表
- tags CRUD endpoints 工作正常
- 至少 1 个 session-tag 关联 endpoint 可用并有测试覆盖
- `go test ./...` 全通过
