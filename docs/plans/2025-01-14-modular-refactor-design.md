# Time Tracker 模块化重构设计

**日期**: 2025-01-14
**作者**: Claude & Duola
**状态**: 设计阶段

## 目标

1. **代码组织重构** - 从分层架构改为模块化单体（按功能域组织），解决"找不到代码在哪里"的问题
2. **添加标签系统** - 支持灵活的标签分类和管理
3. **添加数据统计/报表** - 提供时间统计和分类分析功能

## 目录结构

```
time-tracker/
├── cmd/
│   └── server/
│       └── main.go              # 入口点（简化）
├── internal/
│   ├── sessions/                # 会话管理模块
│   │   ├── handler.go           # HTTP 处理
│   │   ├── service.go           # 业务逻辑
│   │   ├── repository.go        # 数据访问
│   │   ├── models.go            # 领域模型
│   │   └── *_test.go            # 测试
│   ├── tags/                    # 标签模块（新）
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   └── *_test.go
│   ├── statistics/              # 统计报表模块（新）
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   └── *_test.go
│   ├── web/                     # Web 界面模块
│   │   ├── handler.go
│   │   └── *_test.go
│   ├── shared/                  # 共享组件
│   │   ├── database/
│   │   │   └── database.go
│   │   ├── auth/
│   │   │   └── auth.go
│   │   ├── middleware/
│   │   │   ├── rate_limit.go
│   │   │   ├── security.go
│   │   │   └── chain.go
│   │   ├── errors/
│   │   │   └── errors.go
│   │   └── validation/
│   │       └── validation.go
│   └── app/                     # 应用组装
│       ├── app.go               # 依赖注入
│       └── router.go            # 路由注册
├── templates/
├── data/
├── docs/
│   └── plans/
└── go.mod
```

## 模块接口

### Sessions 模块

```go
type SessionService interface {
    Start(data *SessionStart) (*Session, error)
    Stop(data *SessionStop) (*Session, error)
    GetCurrent() (*Session, error)
    List(filter *SessionFilter) (*PaginatedResult, error)
    Delete(id int64) error
    Update(id int64, data *SessionUpdate) error
    ExportCSV(filter *SessionFilter) ([]byte, error)
}
```

### Tags 模块（新）

```go
type TagService interface {
    Create(name string, color string) (*Tag, error)
    List() ([]Tag, error)
    Get(id int64) (*Tag, error)
    Delete(id int64) error
    Update(id int64, data *TagUpdate) error
}
```

### Statistics 模块（新）

```go
type StatisticsService interface {
    DailySummary(date string) (*DailySummary, error)
    DateRangeSummary(start, end string) (*DateRangeSummary, error)
    CategoryBreakdown(start, end string) ([]CategoryStat, error)
    TagUsage(start, end string) ([]TagStat, error)
}
```

## 数据模型

### Tag 模型

```go
type Tag struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Color     string    `json:"color"`
    CreatedAt time.Time `json:"created_at"`
}

type TagUpdate struct {
    Name  *string `json:"name,omitempty"`
    Color *string `json:"color,omitempty"`
}
```

### Statistics 模型

```go
type DailySummary struct {
    Date         string          `json:"date"`
    TotalSeconds int64           `json:"total_seconds"`
    SessionCount int             `json:"session_count"`
    Categories   []CategoryStat  `json:"categories"`
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

### Session 模型扩展

```go
type Session struct {
    ID          int64     `json:"id"`
    Category    string    `json:"category"`
    Task        string    `json:"task"`
    Note        *string   `json:"note,omitempty"`
    Location    *string   `json:"location,omitempty"`
    Mood        *string   `json:"mood,omitempty"`
    StartedAt   string    `json:"started_at"`
    EndedAt     *string   `json:"ended_at,omitempty"`
    DurationSec *int64    `json:"duration_sec,omitempty"`
    Status      string    `json:"status"`
    Tags        []Tag     `json:"tags,omitempty"`  // 新增
}
```

## 数据库变更

```sql
-- 标签表
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6B7280',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_tags_name ON tags(name);

-- 会话-标签关联表（多对多）
CREATE TABLE session_tags (
    session_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (session_id, tag_id),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_session_tags_session ON session_tags(session_id);
CREATE INDEX idx_session_tags_tag ON session_tags(tag_id);
```

## API 端点

### Tags API（新）

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/tags` | 创建标签 |
| GET | `/api/v1/tags` | 列出所有标签 |
| GET | `/api/v1/tags/:id` | 获取单个标签 |
| PUT | `/api/v1/tags/:id` | 更新标签 |
| DELETE | `/api/v1/tags/:id` | 删除标签 |

### Statistics API（新）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/statistics/daily/:date` | 单日统计 |
| GET | `/api/v1/statistics/range` | 日期范围统计 |
| GET | `/api/v1/statistics/categories` | 分类对比 |
| GET | `/api/v1/statistics/tags` | 标签使用统计 |

### Session API（扩展）

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/sessions/:id/tags` | 关联标签到会话 |
| DELETE | `/api/v1/sessions/:id/tags/:tag_id` | 移除会话标签 |

## 应用组装

### app/app.go

```go
type App struct {
    db          *sql.DB
    cfg         *Config
    sessionSvc  *sessions.Service
    tagSvc      *tags.Service
    statsSvc    *statistics.Service
    router      http.Handler
}

func New(cfg *Config) (*App, error) {
    db := database.New(cfg.DBPath)

    // Repositories
    sessionRepo := sessions.NewRepository(db)
    tagRepo := tags.NewRepository(db)

    // Services
    sessionSvc := sessions.NewService(sessionRepo)
    tagSvc := tags.NewService(tagRepo)
    statsSvc := statistics.NewService(sessionRepo)

    return &App{
        db:         db,
        cfg:        cfg,
        sessionSvc: sessionSvc,
        tagSvc:     tagSvc,
        statsSvc:   statsSvc,
        router:     NewRouter(sessionSvc, tagSvc, statsSvc, cfg),
    }, nil
}
```

### 简化后的 main.go

```go
func main() {
    cfg := config.Load()

    app, err := app.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer app.Shutdown(context.Background())

    app.Run(cfg.Port)
}
```

## 迁移策略

### 阶段 1：建立新结构
1. 创建 `internal/shared/` 目录
2. 移动 database/, auth/, middleware/, errors/, validation/
3. 创建 `internal/app/` 框架

### 阶段 2：迁移 sessions 模块
1. 创建 `internal/sessions/`
2. 移动并重构代码
3. 更新 imports
4. 运行测试验证

### 阶段 3：迁移 web 模块
1. 创建 `internal/web/`
2. 移动 web handler

### 阶段 4：添加新模块
1. 创建 `internal/tags/`（完整实现）
2. 创建 `internal/statistics/`（完整实现）
3. 数据库迁移

### 阶段 5：切换入口
1. `app/app.go` 接管
2. 简化 `main.go`
3. 清理旧目录

## 测试策略

### 测试结构
```
internal/
├── sessions/
│   ├── service_test.go
│   ├── handler_test.go
│   └── repository_test.go
├── tags/
│   ├── service_test.go
│   └── handler_test.go
└── shared/
    └── fixtures/
        └── db_test.go
```

### 覆盖目标
- 单元测试覆盖率 ≥ 80%
- 集成测试覆盖关键路径
- 使用内存 SQLite 进行集成测试

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 破坏现有功能 | 渐进式迁移，每阶段验证 |
| import 循环依赖 | 严格依赖方向：app → modules → shared |
| 数据库迁移失败 | 先备份，使用事务 |
| 测试不足 | 每阶段运行 `go test ./...` |

## 后续优化（可选）

- [ ] 添加用户/多账号支持
- [ ] 添加 WebSocket 实时更新
- [ ] 添加数据导出（JSON、Excel）
- [ ] 添加移动端优化
