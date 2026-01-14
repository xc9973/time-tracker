# 应用组装与清理（Phase 5）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 引入 `internal/app` 作为唯一组装入口：统一创建 DB、repositories、services、handlers、middlewares，并在完成 Phase 1-4 后清理旧目录（handler/service/repository/models 等）。

**Architecture:** `cmd/server/main.go` 只做：加载配置 → new(app) → run/shutdown。路由注册在 `internal/app/router.go`，依赖注入在 `internal/app/app.go`。保持最小 diff：先做到“main.go 简化但功能不变”，再删除旧目录。

**Tech Stack:** Go, net/http, context

---

## 前置检查

**Step 1: 基线测试通过**

Run: `go test ./...`
Expected: PASS

**Step 2: 确认 Phase 1-4 已实现对应模块**

需要存在：
- shared: `internal/shared/*`
- sessions: `internal/sessions/*`
- web: `internal/web/*`
- tags: `internal/tags/*`
- statistics: `internal/statistics/*`

---

## Task 1: 创建 internal/app 结构（若未创建）

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/router.go`
- Test: `internal/app/app_test.go`（可选，编译级）

**Step 1: 创建目录与骨架**

Run:
```bash
mkdir -p internal/app
```

**Step 2: 写编译级测试（确保包能被引用）**

Create `internal/app/app_test.go`:

```go
package app

import "testing"

func TestApp_Compile(t *testing.T) {}
```

Run: `go test ./internal/app -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/app/
git commit -m "phase5: add app wiring skeleton"
```

---

## Task 2: 迁移配置加载到 internal/app（保持行为不变）

**Files:**
- Modify: `cmd/server/main.go`（或拆到 `internal/app/config.go`）

**Step 1: 抽出 Config struct 与 LoadConfig()**

将 `cmd/server/main.go` 里的 `Config` + `LoadConfig()` 搬到 `internal/app/config.go`（或保留在 main.go，但 main.go 只调用）

**Step 2: 运行全量测试**

Run: `go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add -A
git commit -m "phase5: move config loading into app"
```

---

## Task 3: 在 internal/app/app.go 组装依赖

**Files:**
- Modify: `internal/app/app.go`

**Step 1: 组装 DB + repos + services + handlers**

`New(cfg)` 内创建：
- db: `database.New(cfg.DBPath)`
- sessionRepo/service/handler
- tagRepo/service/handler
- statsRepo/service/handler
- webHandler（需要 templates path + tz）
- healthHandler

**Step 2: 组装 middleware chain**

保持与当前 main.go 行为一致：
- rate limit
- nonce (CSP)
- security headers
- auth middleware 对 `/api/` 生效、basic auth 对 `/web/` 与 `/sessions.csv` 生效

**Step 3: 测试与构建**

Run: `go test ./...`
Expected: PASS

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Commit**

```bash
git add internal/app/app.go internal/app/router.go
git commit -m "phase5: wire app dependencies and router"
```

---

## Task 4: 简化 cmd/server/main.go

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: main.go 仅保留启动/优雅退出**

保留：
- signal 监听
- server 启动
- shutdown

逻辑改为：
- `cfg, err := app.LoadConfig()`
- `a, err := app.New(cfg)`
- `a.Run()`

**Step 2: 测试与构建**

Run: `go test ./...`
Expected: PASS

Run: `go build ./cmd/server`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "phase5: simplify main entrypoint"
```

---

## Task 5: 清理旧目录（仅在无引用后）

**Files:**
- Delete: `internal/handler/`（若已迁移完）
- Delete: `internal/service/`
- Delete: `internal/repository/`
- Delete: `internal/models/`
- Delete: `internal/auth/` 等（由 Phase 1 迁移后决定）

**Step 1: 确认无引用**

Run: `go test ./...`
Expected: PASS

Run: `go list ./...`（可选）

**Step 2: 删除目录**

```bash
# only if empty/unreferenced
```

**Step 3: 全量验证**

Run: `go test ./... -v`
Expected: PASS

Run: `go build ./cmd/server`
Expected: Success

**Step 4: Commit**

```bash
git add -A
git commit -m "phase5: remove old layered directories"
```

---

## Done 标准

- `cmd/server/main.go` 显著简化但功能行为保持
- 路由与中间件链行为与当前一致
- 所有旧目录清理完成
- `go test ./...` + `go build ./cmd/server` 全通过
