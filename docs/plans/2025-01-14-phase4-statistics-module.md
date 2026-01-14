# Statistics 模块（Phase 4B）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增 `internal/statistics` 模块，提供 sessions 的统计报表 API（按日汇总、日期范围汇总、按分类统计、按标签统计）。

**Architecture:** statistics 只读查询 sessions 数据（以及 session_tags/tags），不修改 session 状态。保持最小 diff：优先在 repository 层用 SQL 聚合（SUM/COUNT/GROUP BY），service 负责参数解析与业务校验（日期范围合法性、默认值），handler 负责 HTTP 路由和错误映射。

**Tech Stack:** Go, SQLite, net/http, time, encoding/json

---

## 前置检查（必做）

**Step 1: 基线测试通过**

Run: `go test ./...`
Expected: PASS

**Step 2: 确认 sessions 表与索引存在**

由 `internal/database/database.go:initTables` 负责（或迁移后 shared/database）。

**Step 3: 如需 tag 统计，确认 Phase 4A 已完成**

需要存在 `tags` 与 `session_tags` 表。

---

## Task 1: 定义统计响应模型（最小）

**Files:**
- Create: `internal/statistics/models.go`

**Step 1: 写失败测试（编译级即可）**

Create `internal/statistics/models_test.go`:

```go
package statistics

import "testing"

func TestModels_Compile(t *testing.T) {
    _ = DailySummary{}
    _ = CategoryStat{}
}
```

Expected: FAIL（类型未定义）

**Step 2: 运行测试确认失败**

Run: `go test ./internal/statistics -run TestModels_Compile -v`
Expected: FAIL

**Step 3: 写最小实现**

Create `internal/statistics/models.go`:

```go
package statistics

type DailySummary struct {
    Date         string         `json:"date"`
    TotalSeconds int64          `json:"total_seconds"`
    SessionCount int            `json:"session_count"`
    Categories   []CategoryStat `json:"categories"`
}

type CategoryStat struct {
    Category     string  `json:"category"`
    TotalSeconds int64   `json:"total_seconds"`
    Percentage   float64 `json:"percentage"`
}

type TagStat struct {
    TagID        int64  `json:"tag_id"`
    TagName      string `json:"tag_name"`
    UsageCount   int    `json:"usage_count"`
    TotalSeconds int64  `json:"total_seconds"`
}

type DateRangeSummary struct {
    Start           string         `json:"start"`
    End             string         `json:"end"`
    TotalSeconds    int64          `json:"total_seconds"`
    AvgDailySeconds int64          `json:"avg_daily_seconds"`
    Categories      []CategoryStat `json:"categories"`
}
```

**Step 4: 重新跑测试**

Run: `go test ./internal/statistics -run TestModels_Compile -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/statistics/models.go internal/statistics/models_test.go
git commit -m "phase4(stats): add statistics models"
```

---

## Task 2: 实现 statistics repository（SQL 聚合查询）

**Files:**
- Create: `internal/statistics/repository.go`
- Test: `internal/statistics/repository_test.go`

**Step 1: 写失败测试（按分类聚合）**

Create `internal/statistics/repository_test.go`:

```go
package statistics

import (
    "os"
    "testing"

    "time-tracker/internal/database"
    "time-tracker/internal/models"
    "time-tracker/internal/repository"
)

func TestStatsRepository_CategoryBreakdown(t *testing.T) {
    tmp, _ := os.CreateTemp("", "stats_*.db")
    tmp.Close()
    defer os.Remove(tmp.Name())

    db, err := database.New(tmp.Name())
    if err != nil { t.Fatal(err) }
    defer db.Close()

    // create a few sessions via existing SessionRepository
    sessRepo := repository.NewSessionRepository(db)

    _, _ = sessRepo.Create(&models.SessionStart{Category: "工作", Task: "A"})
    _, _ = sessRepo.StopRunning(&models.SessionStop{})

    _, _ = sessRepo.Create(&models.SessionStart{Category: "学习", Task: "B"})
    _, _ = sessRepo.StopRunning(&models.SessionStop{})

    statsRepo := NewStatsRepository(db)
    items, err := statsRepo.CategoryBreakdown("2020-01-01", "2030-01-01")
    if err != nil { t.Fatal(err) }
    if len(items) < 2 { t.Fatalf("expected >=2") }
}
```

Expected: FAIL

**Step 2: 写最小实现**

Create `internal/statistics/repository.go`:

```go
package statistics

import (
    "database/sql"
    "fmt"

    "time-tracker/internal/database"
)

type StatsRepository struct {
    db *database.DB
}

func NewStatsRepository(db *database.DB) *StatsRepository {
    return &StatsRepository{db: db}
}

// CategoryBreakdown aggregates stopped sessions within [start, end] (date strings).
func (r *StatsRepository) CategoryBreakdown(startDate, endDate string) ([]CategoryStat, error) {
    rows, err := r.db.Query(
        `SELECT category, COALESCE(SUM(duration_sec),0) AS total
         FROM sessions
         WHERE status = 'stopped'
           AND date(started_at) >= date(?)
           AND date(started_at) <= date(?)
         GROUP BY category
         ORDER BY total DESC`,
        startDate, endDate,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to query category breakdown: %w", err)
    }
    defer rows.Close()

    out := []CategoryStat{}
    var grand int64
    tmp := []struct {
        cat string
        sec int64
    }{}

    for rows.Next() {
        var cat sql.NullString
        var sec sql.NullInt64
        if err := rows.Scan(&cat, &sec); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }
        c := ""
        if cat.Valid { c = cat.String }
        s := int64(0)
        if sec.Valid { s = sec.Int64 }
        tmp = append(tmp, struct{cat string; sec int64}{c, s})
        grand += s
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows error: %w", err)
    }

    for _, it := range tmp {
        pct := 0.0
        if grand > 0 {
            pct = float64(it.sec) / float64(grand)
        }
        out = append(out, CategoryStat{Category: it.cat, TotalSeconds: it.sec, Percentage: pct})
    }
    return out, nil
}
```

**Step 3: 重新跑测试**

Run: `go test ./internal/statistics -run TestStatsRepository_CategoryBreakdown -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/statistics/repository.go internal/statistics/repository_test.go
git commit -m "phase4(stats): add stats repository category breakdown"
```

---

## Task 3: 实现 statistics service（参数校验 + 组合）

**Files:**
- Create: `internal/statistics/service.go`
- Test: `internal/statistics/service_test.go`

**Step 1: 写失败测试（日期参数非法）**

Create `internal/statistics/service_test.go`:

```go
package statistics

import "testing"

func TestStatsService_InvalidDate(t *testing.T) {
    svc := &StatsService{}
    if _, err := svc.CategoryBreakdown("bad", "2025-01-01"); err == nil {
        t.Fatalf("expected error")
    }
}
```

Expected: FAIL

**Step 2: 最小实现**

Create `internal/statistics/service.go`:

```go
package statistics

import (
    "errors"
    "time"
)

var ErrInvalidDate = errors.New("invalid date")

type StatsService struct {
    repo *StatsRepository
}

func NewStatsService(repo *StatsRepository) *StatsService {
    return &StatsService{repo: repo}
}

func parseDate(s string) error {
    _, err := time.Parse("2006-01-02", s)
    return err
}

func (s *StatsService) CategoryBreakdown(start, end string) ([]CategoryStat, error) {
    if err := parseDate(start); err != nil {
        return nil, ErrInvalidDate
    }
    if err := parseDate(end); err != nil {
        return nil, ErrInvalidDate
    }
    return s.repo.CategoryBreakdown(start, end)
}
```

**Step 3: 重新跑测试**

Run: `go test ./internal/statistics -run TestStatsService_InvalidDate -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/statistics/service.go internal/statistics/service_test.go
git commit -m "phase4(stats): add stats service"
```

---

## Task 4: 实现 statistics handler（HTTP API）

**Files:**
- Create: `internal/statistics/handler.go`
- Test: `internal/statistics/handler_test.go`

**Endpoints（最小集先做 categories）**
- GET `/api/v1/statistics/categories?start=YYYY-MM-DD&end=YYYY-MM-DD`

**Step 1: 写失败测试**

Create `internal/statistics/handler_test.go`:

```go
package statistics

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestStatsHandler_Categories_Validation(t *testing.T) {
    h := NewStatsHandler(nil)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/statistics/categories?start=bad&end=2025-01-01", nil)
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", w.Code)
    }
}
```

Expected: FAIL

**Step 2: 最小实现 handler**

Create `internal/statistics/handler.go`:

```go
package statistics

import (
    "encoding/json"
    "net/http"

    "time-tracker/internal/errors"
)

type StatsHandler struct {
    svc *StatsService
}

func NewStatsHandler(svc *StatsService) *StatsHandler {
    return &StatsHandler{svc: svc}
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch {
    case r.URL.Path == "/api/v1/statistics/categories" && r.Method == http.MethodGet:
        h.Categories(w, r)
    default:
        errors.WriteError(w, errors.NotFoundError("Endpoint not found"))
    }
}

func (h *StatsHandler) Categories(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    start := q.Get("start")
    end := q.Get("end")

    items, err := h.svc.CategoryBreakdown(start, end)
    if err != nil {
        if err == ErrInvalidDate {
            errors.WriteError(w, errors.ValidationError("Invalid date"))
            return
        }
        errors.WriteError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(items)
}
```

**Step 3: 重新跑 handler 测试**

Run: `go test ./internal/statistics -run TestStatsHandler_Categories_Validation -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/statistics/handler.go internal/statistics/handler_test.go
git commit -m "phase4(stats): add statistics http handler"
```

---

## Task 5: 把 statistics 路由挂到 main/app 路由

**Files:**
- Modify: `cmd/server/main.go`（现有 router 组装处）

**Step 1: 写失败集成测试（可选，若已有路由测试则扩展）**

如已有 handler 路由测试，可在对应测试文件加一个请求到 `/api/v1/statistics/categories`。

**Step 2: 最小实现路由注册**

在 main.go 的 apiHandler switch 中，加入：
- `/api/v1/statistics/*` → `statsHandler.ServeHTTP`

**Step 3: 运行全量测试**

Run: `go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "phase4(stats): register statistics routes"
```

---

## 扩展任务（细化：daily / range / tags）

> 下面三个任务建议按顺序做：先把日期解析/范围校验完善，再加 SQL 聚合，再挂到 handler。

---

## Task 6: Daily Summary（/api/v1/statistics/daily/:date）

**Endpoint:**
- GET `/api/v1/statistics/daily/:date`

**Files:**
- Modify: `internal/statistics/repository.go`
- Modify: `internal/statistics/service.go`
- Modify: `internal/statistics/handler.go`
- Test: `internal/statistics/*_test.go`

**Step 1: 写失败测试（handler 路由 + 400 校验）**

在 `internal/statistics/handler_test.go` 增加：

```go
func TestStatsHandler_Daily_Validation(t *testing.T) {
    h := NewStatsHandler(nil)

    req := httptest.NewRequest(http.MethodGet, "/api/v1/statistics/daily/bad", nil)
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", w.Code)
    }
}
```

Run: `go test ./internal/statistics -run TestStatsHandler_Daily_Validation -v`
Expected: FAIL

**Step 2: 扩展 handler 路由匹配**

在 `ServeHTTP` switch 中增加：
- `strings.HasPrefix(r.URL.Path, "/api/v1/statistics/daily/") && GET` → `h.Daily(w,r)`

并新增 `Daily(w,r)`：
- 从 path 解析 date
- 调用 `svc.DailySummary(date)`
- `ErrInvalidDate` → 400

Run: `go test ./internal/statistics -run TestStatsHandler_Daily_Validation -v`
Expected: PASS

**Step 3: 写失败测试（service 日期解析）**

在 `internal/statistics/service_test.go` 增加：

```go
func TestStatsService_Daily_InvalidDate(t *testing.T) {
    svc := &StatsService{}
    if _, err := svc.DailySummary("bad"); err == nil {
        t.Fatalf("expected error")
    }
}
```

Run: `go test ./internal/statistics -run TestStatsService_Daily_InvalidDate -v`
Expected: FAIL

**Step 4: 实现 service + repo（最小 SQL）**

- `StatsRepository.DailySummary(date string) (totalSeconds int64, count int, err error)`：

```sql
SELECT COALESCE(SUM(duration_sec),0) AS total, COUNT(*) AS cnt
FROM sessions
WHERE status='stopped'
  AND date(started_at)=date(?)
```

- `StatsService.DailySummary(date string) (*DailySummary, error)`：
  - 校验 `YYYY-MM-DD`
  - 调 repo 拿 total + count
  - 调 `CategoryBreakdown(date,date)` 生成 categories

**Step 5: 运行测试**

Run: `go test ./internal/statistics -v`
Expected: PASS

**Step 6: Commit**

```bash
git add -A
git commit -m "phase4(stats): add daily summary endpoint"
```

---

## Task 7: Date Range Summary（/api/v1/statistics/range）

**Endpoint:**
- GET `/api/v1/statistics/range?start=YYYY-MM-DD&end=YYYY-MM-DD`

**Files:**
- Modify: `internal/statistics/repository.go`
- Modify: `internal/statistics/service.go`
- Modify: `internal/statistics/handler.go`
- Test: `internal/statistics/*_test.go`

**Step 1: 写失败测试（缺参/非法参返回 400）**

在 `internal/statistics/handler_test.go` 增加：

```go
func TestStatsHandler_Range_Validation(t *testing.T) {
    h := NewStatsHandler(nil)

    req := httptest.NewRequest(http.MethodGet, "/api/v1/statistics/range?start=bad&end=2025-01-01", nil)
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", w.Code)
    }
}
```

Run: `go test ./internal/statistics -run TestStatsHandler_Range_Validation -v`
Expected: FAIL

**Step 2: handler 增加 range 路由与方法**

在 `ServeHTTP` switch 中增加：
- `r.URL.Path == "/api/v1/statistics/range" && GET` → `h.Range(w,r)`

`Range(w,r)`：
- 读取 query start/end
- 调 `svc.DateRangeSummary(start,end)`
- `ErrInvalidDate` 或 `ErrInvalidRange` → 400

**Step 3: service 增加范围校验**

新增错误：
- `ErrInvalidRange = errors.New("invalid range")`

校验规则：
- start/end 必须是 `YYYY-MM-DD`
- start <= end
- 可选：跨度上限（例如 366 天）防止慢查询

**Step 4: repo 实现范围聚合**

```sql
SELECT COALESCE(SUM(duration_sec),0) AS total, COUNT(*) AS cnt
FROM sessions
WHERE status='stopped'
  AND date(started_at) >= date(?)
  AND date(started_at) <= date(?)
```

service 组合输出：
- `TotalSeconds`
- `AvgDailySeconds = total / daysInclusive`
- `Categories = CategoryBreakdown(start,end)`

**Step 5: 测试与提交**

Run: `go test ./internal/statistics -v`
Expected: PASS

Commit:
```bash
git add -A
git commit -m "phase4(stats): add date range summary endpoint"
```

---

## Task 8: Tag Usage（/api/v1/statistics/tags）

**Endpoint:**
- GET `/api/v1/statistics/tags?start=YYYY-MM-DD&end=YYYY-MM-DD`

**Files:**
- Modify: `internal/statistics/repository.go`
- Modify: `internal/statistics/service.go`
- Modify: `internal/statistics/handler.go`
- Test: `internal/statistics/*_test.go`

**Precondition:** Phase 4A 已完成（存在 `tags` + `session_tags` 表）

**Step 1: 写失败测试（当表不存在时给出清晰错误/或跳过）**

在 repo 测试中：
- 如果 tags 表不存在，可选择 `t.Skip("tags not enabled")`

**Step 2: repo 实现 tags 聚合 SQL**

```sql
SELECT t.id, t.name,
       COUNT(DISTINCT st.session_id) AS usage_count,
       COALESCE(SUM(s.duration_sec),0) AS total
FROM tags t
JOIN session_tags st ON st.tag_id = t.id
JOIN sessions s ON s.id = st.session_id
WHERE s.status='stopped'
  AND date(s.started_at) >= date(?)
  AND date(s.started_at) <= date(?)
GROUP BY t.id, t.name
ORDER BY total DESC
```

**Step 3: handler 增加 tags 路由**

在 `ServeHTTP` switch 中增加：
- `r.URL.Path == "/api/v1/statistics/tags" && GET` → `h.Tags(w,r)`

**Step 4: service 复用 DateRange 校验**

调用 repo 获取 `[]TagStat`

**Step 5: 测试与提交**

Run: `go test ./internal/statistics -v`
Expected: PASS

Commit:
```bash
git add -A
git commit -m "phase4(stats): add tag usage endpoint"
```

---

## Done 标准

- `internal/statistics/` 模块存在并可独立测试
- endpoints 至少包含：
  - `/api/v1/statistics/categories`
  - `/api/v1/statistics/daily/:date`
  - `/api/v1/statistics/range`
  - `/api/v1/statistics/tags`（若 tags 可用）
- 所有新增路径都有 400 校验测试
- `go test ./...` 全通过
