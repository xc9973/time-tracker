# Web 模块迁移（Phase 3）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 Web UI 相关 handler 从 `internal/handler/` 迁移到独立的 `internal/web/` 包；同时把健康检查 handler 迁移到 `internal/shared/health/`，并更新 `cmd/server/main.go` 的组装引用，保持现有行为不变。

**Architecture:** 仅做“移动文件 + 调整 package/import + 更新组装入口(main.go)”的重构，不改业务逻辑与路由行为。依赖现有 `pgregory.net/rapid` property tests 与 `go test ./...` 做回归验证。

**Tech Stack:** Go, net/http, html/template, pgregory.net/rapid

---

## 前置检查（必做）

> 这些检查的目的是避免“迁移一半发现依赖不满足”。

**Step 1: 确认基线测试通过**

Run: `go test ./...`
Expected: PASS

**Step 2: 确认当前阶段依赖满足**

本计划默认你已经完成：
- Phase 1（shared 目录）：`internal/shared/middleware`、`internal/shared/utils`、`internal/shared/validation`、`internal/shared/auth`、`internal/shared/database`
- Phase 2（sessions 模块）：`internal/sessions`（包含 `SessionService`、`SessionStart/Stop/Update` 等）

Run:
```bash
ls -la internal/shared/middleware internal/shared/utils internal/shared/validation
ls -la internal/sessions
```
Expected: 两个命令都能列出目录内容

> 如果以上目录不存在：
> - 先执行 Phase 1/2，或者
> - 在本阶段临时保持旧 import（`internal/middleware` / `internal/utils` / `internal/validation` / `internal/service` / `internal/models`），待 Phase 1/2 完成后再统一替换。

---

### Task 1: 创建 `internal/web/` 并迁移 Web 源码文件

**Files:**
- Move: `internal/handler/web.go` → `internal/web/handler.go`
- Move: `internal/handler/web_sessions.go` → `internal/web/sessions.go`

**Step 1: 创建目录并移动文件（同一组操作，避免空目录无法提交）**

Run:
```bash
mkdir -p internal/web

git mv internal/handler/web.go internal/web/handler.go
git mv internal/handler/web_sessions.go internal/web/sessions.go
```
Expected: `git status` 显示两处 rename/move

**Step 2: 修改 `internal/web/handler.go` 的包名与 imports**

在 `internal/web/handler.go`：
- 把 `package handler` 改为：
```go
package web
```
- 把 imports 调整为使用 shared + sessions：
```go
import (
    "fmt"
    "html/template"
    "net/http"
    "time"

    "time-tracker/internal/shared/middleware"
    "time-tracker/internal/sessions"
)
```
- 把结构体字段类型从 `*service.SessionService` 改为 `*sessions.SessionService`：
```go
type WebHandler struct {
    sessionService   *sessions.SessionService
    sessionsTemplate *template.Template
    timezone         *time.Location
    apiKey           string
}
```
- 把构造函数签名从 `*service.SessionService` 改为 `*sessions.SessionService`：
```go
func NewWebHandler(sessionSvc *sessions.SessionService, templatesPath string, tz *time.Location, apiKey string) (*WebHandler, error) {
```

**Step 3: 修改 `internal/web/sessions.go` 的包名与 imports**

在 `internal/web/sessions.go`：
- 把 `package handler` 改为：
```go
package web
```
- 把 imports 调整为使用 shared + sessions：
```go
import (
    "encoding/json"
    "net/http"
    "strconv"

    "time-tracker/internal/sessions"
    "time-tracker/internal/shared/utils"
    "time-tracker/internal/shared/validation"
)
```

**Step 4: 替换类型引用（models/service → sessions）**

在 `internal/web/sessions.go`：
- `models.SessionStart` → `sessions.SessionStart`
- `models.SessionStop` → `sessions.SessionStop`
- `models.SessionUpdate` → `sessions.SessionUpdate`
- `service.ErrSessionAlreadyRunning` → `sessions.ErrSessionAlreadyRunning`
- `service.ErrNoRunningSession` → `sessions.ErrNoRunningSession`

**Step 5: 运行局部编译验证（预期先失败也正常）**

Run: `go test ./...`
Expected: 可能 FAIL（例如 `cmd/server/main.go` 仍引用旧的 `handler.NewWebHandler`）

**Step 6: Commit（仅在修复 main.go 后再提交）**

暂不提交，等待 Task 3 更新 `cmd/server/main.go` 之后再一起提交，避免中间提交破坏构建。

---

### Task 2: 迁移 Web property tests 到 `internal/web/`

**Files:**
- Move: `internal/handler/web_property_test.go` → `internal/web/web_property_test.go`

**Step 1: 移动测试文件**

Run:
```bash
git mv internal/handler/web_property_test.go internal/web/web_property_test.go
```
Expected: `git status` 显示 rename/move

**Step 2: 修改测试文件包名与 imports**

在 `internal/web/web_property_test.go`：
- 把 `package handler` 改为：
```go
package web
```
- 把 imports 中的旧包替换为 shared + sessions：
  - `time-tracker/internal/auth` → `time-tracker/internal/shared/auth`
  - `time-tracker/internal/database` → `time-tracker/internal/shared/database`
  - `time-tracker/internal/repository` / `time-tracker/internal/service` / `time-tracker/internal/models` → `time-tracker/internal/sessions`

并将 `setupWebTestEnv` 内的初始化替换为：
```go
sessionRepo := sessions.NewSessionRepository(db)
sessionSvc := sessions.NewSessionService(sessionRepo)

handler, err := NewWebHandler(sessionSvc, tmpDir, tz, apiKey)
```

**Step 3: 先跑一个 property test（快速验证 wiring）**

Run: `go test ./internal/web -run TestWebBasicAuth_Property15_ValidAuth -v`
Expected: PASS

**Step 4: Commit（仍暂缓）**

暂不提交，等待 main.go 与 health 迁移完成后统一提交，减少回滚成本。

---

### Task 3: 更新 `cmd/server/main.go` 使用 `internal/web`

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: 更新 imports**

在 `cmd/server/main.go` imports 中：
- 增加：
```go
"time-tracker/internal/web"
```
- 删除/替换：`"time-tracker/internal/handler"` 仅用于 web 的引用（如果 handler 仍用于 sessions/health，则保留 handler import）

> 如果 Phase 2 已完成且 sessions handler 也已迁移到 `internal/sessions`，则 main.go 里 `handler` import 可以进一步减少；否则保持最小改动。

**Step 2: 更新 NewWebHandler 调用点**

把：
```go
webHandler, err := handler.NewWebHandler(sessionService, absTemplates, tz, cfg.APIKey)
```
改为：
```go
webHandler, err := web.NewWebHandler(sessionService, absTemplates, tz, cfg.APIKey)
```

**Step 3: 运行构建验证**

Run: `go build ./cmd/server`
Expected: Success

**Step 4: 运行全量测试**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit（本阶段第一个可用提交点）**

```bash
git add -A
git commit -m "phase3: move web handlers into internal/web"
```

---

### Task 4: 迁移 Health handler 到 `internal/shared/health/`

**Files:**
- Move: `internal/handler/health.go` → `internal/shared/health/handler.go`
- Modify: `cmd/server/main.go`

**Step 1: 创建目录并移动文件**

Run:
```bash
mkdir -p internal/shared/health

git mv internal/handler/health.go internal/shared/health/handler.go
```

**Step 2: 修改包名**

在 `internal/shared/health/handler.go`：
- 把 `package handler` 改为：
```go
package health
```

**Step 3: 更新 main.go import 与初始化**

在 `cmd/server/main.go`：
- imports 增加：
```go
"time-tracker/internal/shared/health"
```
- 把：
```go
healthHandler := handler.NewHealthHandler()
```
改为：
```go
healthHandler := health.NewHealthHandler()
```

**Step 4: 测试与构建**

Run: `go test ./...`
Expected: PASS

Run: `go build ./cmd/server`
Expected: Success

**Step 5: Commit**

```bash
git add -A
git commit -m "phase3: move health handler into shared"
```

---

### Task 5: 清理 `internal/handler/`（仅在可删时）

**Files:**
- Potentially delete: `internal/handler/`（仅在目录已空、且 sessions 已迁移）

**Step 1: 检查 handler 目录是否仍有文件**

Run: `ls -la internal/handler/`
Expected:
- 如果仍有 `sessions.go` 等文件：不要删除（等待 Phase 2/5）。
- 如果目录为空：可以删除。

**Step 2: 删除空目录**

Run:
```bash
if [ -d internal/handler ] && [ -z "$(ls -A internal/handler)" ]; then
  git rm -r internal/handler
fi
```

**Step 3: 回归验证**

Run: `go test ./...`
Expected: PASS

---

### Task 6: 最小烟测（可选，但建议）

**Step 1: 启动服务**

Run:
```bash
TIMELOG_API_KEY=12345678901234567890123456789012 go run ./cmd/server
```
Expected: 监听在 `:7070`（或你配置的端口）

**Step 2: 验证 Web 页面可访问**

Run:
```bash
curl -i http://localhost:7070/web/sessions
```
Expected:
- 如果未配置 Basic Auth：HTTP 200，返回 HTML
- 如果配置了 Basic Auth：HTTP 401，并带 `WWW-Authenticate`

**Step 3: 结束服务**

Ctrl+C

---

## Done 标准

- `internal/web/handler.go` 与 `internal/web/sessions.go` 存在且 `package web`
- `internal/web/web_property_test.go` 能跑通（至少 1 个 property test）
- `internal/shared/health/handler.go` 存在且 `package health`
- `cmd/server/main.go` 已更新引用
- `go test ./...` 与 `go build ./cmd/server` 均通过
