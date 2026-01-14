# Design Document: Time Tracker

## Overview

Time Tracker 是一个个人时间记录系统，采用 FastAPI + SQLite 的轻量级架构。系统提供 REST API 供 iOS 快捷指令调用，同时提供简洁的 Web 界面用于查看和导出数据。

### 核心设计原则

1. **简单优先**: 单用户场景，避免过度设计
2. **快捷指令友好**: API 设计考虑 iOS Shortcuts 的调用特点
3. **安全可靠**: 完善的认证、速率限制和输入验证
4. **可部署性**: 支持云服务器和家庭设备两种部署方式

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client Layer                           │
├─────────────────┬─────────────────┬─────────────────────────┤
│  iOS Shortcuts  │   Web Browser   │      curl/httpie        │
│  (X-API-Key)    │  (Basic Auth)   │      (X-API-Key)        │
└────────┬────────┴────────┬────────┴────────────┬────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Entry Layer (选一)                        │
├─────────────────────────────────────────────────────────────┤
│  Option A: Caddy (云服务器)                                  │
│    - 自动 HTTPS                                              │
│    - 反向代理到 127.0.0.1:8000                               │
│                                                              │
│  Option B: Cloudflared (家庭设备)                            │
│    - Tunnel 到 127.0.0.1:8000                                │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    FastAPI Application                       │
├─────────────────────────────────────────────────────────────┤
│  Middleware Layer:                                           │
│    - RateLimitMiddleware (速率限制)                          │
│    - SecurityHeadersMiddleware (安全头)                      │
│    - RequestLoggingMiddleware (请求日志)                     │
├─────────────────────────────────────────────────────────────┤
│  Auth Layer:                                                 │
│    - APIKeyAuth (X-API-Key for /api/*)                      │
│    - BasicAuth (for /web/*, /*.csv)                         │
├─────────────────────────────────────────────────────────────┤
│  Router Layer:                                               │
│    - /api/v1/logs (打点日志 CRUD)                            │
│    - /api/v1/sessions (计时 CRUD)                            │
│    - /web/* (HTML 页面)                                      │
│    - /healthz (健康检查)                                     │
├─────────────────────────────────────────────────────────────┤
│  Service Layer:                                              │
│    - LogService (日志业务逻辑)                               │
│    - SessionService (计时业务逻辑)                           │
│    - ExportService (CSV 导出)                                │
├─────────────────────────────────────────────────────────────┤
│  Repository Layer:                                           │
│    - LogRepository (日志数据访问)                            │
│    - SessionRepository (计时数据访问)                        │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                      SQLite Database                         │
│  Tables: logs, sessions                                      │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Configuration Module (`config.py`)

负责从环境变量加载配置，并进行验证。

```python
class Settings:
    api_key: str           # TIMELOG_API_KEY (必填, >=32字符)
    db_path: str           # TIMELOG_DB_PATH (默认: ./timelog.db)
    timezone: str          # TIMELOG_TZ (默认: UTC)
    basic_user: str | None # TIMELOG_BASIC_USER (可选)
    basic_pass: str | None # TIMELOG_BASIC_PASS (可选)
    rate_limit: int        # TIMELOG_RATE_LIMIT (默认: 100)
```

### 2. Database Module (`database.py`)

管理 SQLite 连接和表初始化。

```python
class Database:
    def __init__(self, db_path: str): ...
    def init_tables(self) -> None: ...
    def get_connection(self) -> sqlite3.Connection: ...
```

### 3. Models (`models.py`)

Pydantic 模型用于请求/响应验证。

```python
# 请求模型
class LogCreate:
    category: str          # 必填
    task: str              # 必填
    note: str | None       # 可选
    location: str | None   # 可选
    mood: str | None       # 可选

class SessionStart:
    category: str
    task: str
    note: str | None
    location: str | None
    mood: str | None

class SessionStop:
    note: str | None
    location: str | None
    mood: str | None

# 响应模型
class LogResponse:
    id: int
    category: str
    task: str
    note: str | None
    location: str | None
    mood: str | None
    created_at: str        # RFC3339 UTC

class SessionResponse:
    id: int
    category: str
    task: str
    note: str | None
    location: str | None
    mood: str | None
    started_at: str        # RFC3339 UTC
    ended_at: str | None
    duration_sec: int | None
    status: str            # "running" | "stopped"
```

### 4. Repository Layer

#### LogRepository (`repositories/log_repository.py`)

```python
class LogRepository:
    def create(self, log: LogCreate) -> LogResponse: ...
    def list(self, limit: int, offset: int, category: str | None, q: str | None) -> list[LogResponse]: ...
    def count(self, category: str | None, q: str | None) -> int: ...
```

#### SessionRepository (`repositories/session_repository.py`)

```python
class SessionRepository:
    def create(self, session: SessionStart) -> SessionResponse: ...
    def get_running(self) -> SessionResponse | None: ...
    def stop_running(self, updates: SessionStop) -> SessionResponse: ...
    def list(self, limit: int, offset: int, status: str | None, category: str | None) -> list[SessionResponse]: ...
    def count(self, status: str | None, category: str | None) -> int: ...
```

### 5. Service Layer

#### LogService (`services/log_service.py`)

```python
class LogService:
    def create_log(self, data: LogCreate) -> LogResponse: ...
    def get_logs(self, limit: int, offset: int, category: str | None, q: str | None) -> PaginatedResponse[LogResponse]: ...
    def export_csv(self, category: str | None, q: str | None) -> str: ...
```

#### SessionService (`services/session_service.py`)

```python
class SessionService:
    def start_session(self, data: SessionStart) -> SessionResponse: ...
    def stop_session(self, data: SessionStop) -> SessionResponse: ...
    def get_current(self) -> SessionResponse | None: ...
    def get_sessions(self, limit: int, offset: int, status: str | None, category: str | None) -> PaginatedResponse[SessionResponse]: ...
    def export_csv(self, status: str | None, category: str | None) -> str: ...
```

### 6. Auth Module (`auth.py`)

```python
def verify_api_key(api_key: str, expected: str) -> bool:
    """常量时间比较 API Key"""
    ...

def verify_basic_auth(credentials: str, user: str, password: str) -> bool:
    """验证 Basic Auth 凭据"""
    ...

class APIKeyDependency:
    """FastAPI 依赖，验证 X-API-Key"""
    ...

class BasicAuthDependency:
    """FastAPI 依赖，验证 Basic Auth"""
    ...
```

### 7. Middleware

#### RateLimitMiddleware (`middleware/rate_limit.py`)

```python
class RateLimitMiddleware:
    """
    基于 IP 的滑动窗口速率限制
    - 使用内存存储（单实例足够）
    - 窗口大小: 1 分钟
    - 默认限制: 100 请求/分钟
    """
    ...
```

#### SecurityHeadersMiddleware (`middleware/security.py`)

```python
class SecurityHeadersMiddleware:
    """
    添加安全响应头:
    - X-Content-Type-Options: nosniff
    - X-Frame-Options: DENY
    - Content-Security-Policy: default-src 'self'
    - X-XSS-Protection: 1; mode=block
    """
    ...
```

### 8. Routers

#### API Router (`routers/api.py`)

```
POST   /api/v1/logs              创建打点日志
GET    /api/v1/logs              查询日志列表
GET    /api/v1/logs.csv          导出日志 CSV

POST   /api/v1/sessions/start    开始计时
POST   /api/v1/sessions/stop     结束计时
GET    /api/v1/sessions/current  查询当前计时
GET    /api/v1/sessions          查询计时列表
GET    /api/v1/sessions.csv      导出计时 CSV
```

#### Web Router (`routers/web.py`)

```
GET    /web/logs                 日志列表页面
GET    /web/sessions             计时列表页面
```

#### Health Router (`routers/health.py`)

```
GET    /healthz                  健康检查（无需认证）
```

## Data Models

### SQLite Schema

```sql
-- 打点日志表
CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    task TEXT NOT NULL,
    note TEXT,
    location TEXT,
    mood TEXT,
    created_at TEXT NOT NULL  -- RFC3339 UTC
);

CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at);
CREATE INDEX IF NOT EXISTS idx_logs_category ON logs(category);

-- 计时段表
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    task TEXT NOT NULL,
    note TEXT,
    location TEXT,
    mood TEXT,
    started_at TEXT NOT NULL,  -- RFC3339 UTC
    ended_at TEXT,             -- RFC3339 UTC, NULL if running
    duration_sec INTEGER,      -- calculated on stop
    status TEXT NOT NULL       -- 'running' or 'stopped'
);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_category ON sessions(category);
```

### 字段约束

| 字段 | 类型 | 约束 |
|------|------|------|
| category | string | 必填, 1-50 字符 |
| task | string | 必填, 1-200 字符 |
| note | string | 可选, 最大 1000 字符 |
| location | string | 可选, 最大 100 字符 |
| mood | string | 可选, 最大 20 字符 |



## API Specifications

### 认证

#### API 认证 (X-API-Key)

```http
POST /api/v1/logs HTTP/1.1
Host: time.example.com
X-API-Key: your-secret-key-at-least-32-chars
Content-Type: application/json
```

#### Web 认证 (Basic Auth)

```http
GET /web/logs HTTP/1.1
Host: time.example.com
Authorization: Basic base64(username:password)
```

### 错误响应格式

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "category is required"
  }
}
```

错误码:
- `VALIDATION_ERROR` (400): 请求参数验证失败
- `UNAUTHORIZED` (401): 认证失败
- `NOT_FOUND` (404): 资源不存在
- `CONFLICT` (409): 资源冲突（如已有运行中的 session）
- `RATE_LIMITED` (429): 超出速率限制
- `INTERNAL_ERROR` (500): 服务器内部错误

### Logs API

#### POST /api/v1/logs

创建打点日志。

Request:
```json
{
  "category": "工作",
  "task": "回邮件",
  "note": "处理客户问题",
  "location": "公司",
  "mood": "🙂一般"
}
```

Response (201):
```json
{
  "id": 1,
  "category": "工作",
  "task": "回邮件",
  "note": "处理客户问题",
  "location": "公司",
  "mood": "🙂一般",
  "created_at": "2024-01-15T08:30:00Z"
}
```

#### GET /api/v1/logs

查询日志列表。

Query Parameters:
- `limit` (int, default=50, max=200): 每页数量
- `offset` (int, default=0): 偏移量
- `category` (string, optional): 分类过滤
- `q` (string, optional): 搜索 task/note

Response (200):
```json
{
  "items": [...],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

#### GET /api/v1/logs.csv

导出日志为 CSV。参数同 GET /api/v1/logs。

Response Headers:
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="logs_20240115.csv"
```

### Sessions API

#### POST /api/v1/sessions/start

开始计时。

Request:
```json
{
  "category": "学习",
  "task": "英语听力",
  "note": "",
  "location": "家",
  "mood": "😀好"
}
```

Response (201):
```json
{
  "id": 1,
  "category": "学习",
  "task": "英语听力",
  "note": "",
  "location": "家",
  "mood": "😀好",
  "started_at": "2024-01-15T09:00:00Z",
  "ended_at": null,
  "duration_sec": null,
  "status": "running"
}
```

Error (409 - 已有运行中的 session):
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "A session is already running",
    "current_session": {
      "id": 1,
      "task": "英语听力",
      "started_at": "2024-01-15T09:00:00Z"
    }
  }
}
```

#### POST /api/v1/sessions/stop

结束当前计时。

Request (可选，用于补充信息):
```json
{
  "note": "完成25分钟",
  "mood": "🙂一般"
}
```

Response (200):
```json
{
  "id": 1,
  "category": "学习",
  "task": "英语听力",
  "note": "完成25分钟",
  "location": "家",
  "mood": "🙂一般",
  "started_at": "2024-01-15T09:00:00Z",
  "ended_at": "2024-01-15T09:25:00Z",
  "duration_sec": 1500,
  "status": "stopped"
}
```

Error (404 - 没有运行中的 session):
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No running session found"
  }
}
```

#### GET /api/v1/sessions/current

查询当前计时状态。

Response (200 - 有运行中的 session):
```json
{
  "running": true,
  "session": {
    "id": 1,
    "task": "英语听力",
    "started_at": "2024-01-15T09:00:00Z",
    "elapsed_sec": 300
  }
}
```

Response (200 - 没有运行中的 session):
```json
{
  "running": false,
  "session": null
}
```

#### GET /api/v1/sessions

查询计时列表。

Query Parameters:
- `limit` (int, default=50, max=200)
- `offset` (int, default=0)
- `status` (string, optional): "running" | "stopped"
- `category` (string, optional)

#### GET /api/v1/sessions.csv

导出计时为 CSV。

### Health API

#### GET /healthz

Response (200):
```json
{
  "ok": true
}
```

## Web Interface Design

### 页面结构

两个页面共用相同的布局：
- 顶部导航栏（切换 Logs/Sessions）
- 搜索/过滤区域
- 数据表格
- 分页控件
- 导出按钮

### /web/logs 页面

```
┌─────────────────────────────────────────────────────────────┐
│  Time Tracker    [日志]  [计时]                              │
├─────────────────────────────────────────────────────────────┤
│  分类: [全部 ▼]  搜索: [________]  [导出 CSV]               │
├─────────────────────────────────────────────────────────────┤
│  时间              分类    事项        备注    地点   心情   │
│  ─────────────────────────────────────────────────────────  │
│  2024-01-15 16:30  工作    回邮件      处理... 公司   🙂    │
│  2024-01-15 14:00  学习    看文档      -       家     😀    │
│  ...                                                         │
├─────────────────────────────────────────────────────────────┤
│  [上一页]  第 1 页 / 共 5 页  [下一页]                       │
└─────────────────────────────────────────────────────────────┘
```

### /web/sessions 页面

```
┌─────────────────────────────────────────────────────────────┐
│  Time Tracker    [日志]  [计时]                              │
├─────────────────────────────────────────────────────────────┤
│  分类: [全部 ▼]  状态: [全部 ▼]  [导出 CSV]                 │
├─────────────────────────────────────────────────────────────┤
│  开始时间          结束时间          分类   事项     时长    │
│  ─────────────────────────────────────────────────────────  │
│  2024-01-15 09:00  2024-01-15 09:25  学习   英语听力 0:25:00│
│  2024-01-15 10:00  (进行中)          工作   写代码   -      │
│  ...                                                         │
├─────────────────────────────────────────────────────────────┤
│  [上一页]  第 1 页 / 共 3 页  [下一页]                       │
└─────────────────────────────────────────────────────────────┘
```

### 技术实现

- 使用 Jinja2 模板渲染
- 简单的 CSS（可用 Pico.css 或手写）
- 无需 JavaScript 框架（纯服务端渲染）
- 时间显示使用配置的时区

## Correctness Properties

*正确性属性是系统在所有有效执行中都应保持为真的特征或行为。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### Property 1: 日志创建完整性

*For any* 有效的日志创建请求（包含 category 和 task），创建后返回的 Log 对象应包含：
- 自动生成的 id
- 所有提交的字段（category, task, 以及任何提供的可选字段）
- RFC3339 格式的 UTC 时间戳 created_at

**Validates: Requirements 1.1, 1.2, 1.3**

### Property 2: 日志输入验证

*For any* 缺少 category 或 task 的日志创建请求，API 应返回 400 错误，且原有日志列表不变。

**Validates: Requirements 1.4**

### Property 3: 日志查询正确性

*For any* 日志查询请求：
- 使用 limit 和 offset 时，返回的结果数量不超过 limit，且按 created_at 降序排列
- 使用 category 过滤时，返回的所有日志的 category 都匹配过滤值
- 使用搜索词 q 时，返回的所有日志的 task 或 note 包含该搜索词

**Validates: Requirements 1.5, 1.6, 1.7**

### Property 4: Session 生命周期

*For any* Session：
- 创建时状态为 "running"，有 started_at 时间戳，ended_at 和 duration_sec 为 null
- 停止后状态为 "stopped"，有 ended_at 时间戳，duration_sec = ended_at - started_at（秒）

**Validates: Requirements 2.1, 2.3**

### Property 5: Session 并发控制

*For any* 已存在 running 状态的 Session，尝试创建新 Session 时应返回 409 Conflict，且包含当前运行中 Session 的信息。

**Validates: Requirements 2.2**

### Property 6: Session 停止时更新

*For any* 停止 Session 请求中提供的可选字段（note, mood, location），停止后的 Session 应包含这些更新的字段值。

**Validates: Requirements 2.4**

### Property 7: Session 查询正确性

*For any* Session 查询请求：
- 查询 current 时，如有 running Session 则返回该 Session，否则返回 running=false
- 使用 status 过滤时，返回的所有 Session 的 status 都匹配过滤值
- 使用 category 过滤时，返回的所有 Session 的 category 都匹配过滤值

**Validates: Requirements 2.6, 2.7**

### Property 8: CSV 导出格式正确性

*For any* CSV 导出请求：
- 响应 Content-Type 为 text/csv
- 内容以 UTF-8 BOM (0xEF 0xBB 0xBF) 开头
- Sessions CSV 中的 duration 格式为 H:MM:SS

**Validates: Requirements 3.1, 3.2, 3.3, 3.5**

### Property 9: CSV 导出过滤一致性

*For any* 相同的过滤条件，CSV 导出的记录数量和内容应与列表 API 返回的结果一致。

**Validates: Requirements 3.4**

### Property 10: API Key 认证正确性

*For any* API 请求到 /api/* 端点：
- 无 X-API-Key 头时返回 401
- X-API-Key 值与配置不匹配时返回 401
- X-API-Key 值正确时正常处理请求
- 配置的 API Key 必须至少 32 字符，否则启动失败

**Validates: Requirements 4.1, 4.2, 4.3, 4.6**

### Property 11: 速率限制正确性

*For any* IP 地址，在 1 分钟内超过配置的请求限制后：
- 返回 429 Too Many Requests
- 响应包含 Retry-After 头

**Validates: Requirements 4.7, 4.8**

### Property 12: 安全头正确性

*For any* API 响应，都应包含以下安全头：
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- Content-Security-Policy

**Validates: Requirements 4.9**

### Property 13: 输入验证安全性

*For any* 包含特殊字符（SQL 注入尝试、XSS 脚本等）的输入，系统应正确存储原始内容而不执行，且查询时返回原始内容。

**Validates: Requirements 4.13**

### Property 14: 错误响应安全性

*For any* 错误响应，不应包含内部系统细节、堆栈跟踪或敏感配置信息。

**Validates: Requirements 4.14**

### Property 15: Web Basic Auth 正确性

*For any* 访问 /web/* 或 /*.csv 端点的请求（当配置了 Basic Auth 时）：
- 无 Authorization 头时返回 401
- 凭据不正确时返回 401
- 凭据正确时正常返回页面/文件

**Validates: Requirements 4.11, 4.12**

### Property 16: 时区显示正确性

*For any* Web 页面显示的时间戳，应按配置的 TIMELOG_TZ 时区显示，而非 UTC。

**Validates: Requirements 5.5**

### Property 17: 时间戳存储格式正确性

*For any* 存储在数据库中的时间戳，格式应为 RFC3339 UTC（如 2024-01-15T08:30:00Z）。

**Validates: Requirements 7.5**

## Error Handling

### 错误响应格式

所有错误响应使用统一的 JSON 格式：

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### 错误码映射

| HTTP Status | Error Code | 场景 |
|-------------|------------|------|
| 400 | VALIDATION_ERROR | 请求参数验证失败（缺少必填字段、格式错误等） |
| 401 | UNAUTHORIZED | API Key 或 Basic Auth 认证失败 |
| 404 | NOT_FOUND | 资源不存在（如尝试停止不存在的 Session） |
| 409 | CONFLICT | 资源冲突（如已有运行中的 Session） |
| 429 | RATE_LIMITED | 超出速率限制 |
| 500 | INTERNAL_ERROR | 服务器内部错误（不暴露细节） |

### 错误处理策略

1. **输入验证错误**: 在 Pydantic 模型层捕获，返回具体的字段错误信息
2. **业务逻辑错误**: 在 Service 层抛出自定义异常，由全局异常处理器转换为 HTTP 响应
3. **数据库错误**: 捕获 SQLite 异常，记录详细日志，返回通用 500 错误
4. **认证错误**: 在中间件/依赖层处理，返回 401 但不暴露具体原因（防止枚举攻击）

### 自定义异常类

```python
class TimeTrackerError(Exception):
    """基础异常类"""
    def __init__(self, code: str, message: str, status_code: int = 400):
        self.code = code
        self.message = message
        self.status_code = status_code

class ValidationError(TimeTrackerError):
    def __init__(self, message: str):
        super().__init__("VALIDATION_ERROR", message, 400)

class NotFoundError(TimeTrackerError):
    def __init__(self, message: str):
        super().__init__("NOT_FOUND", message, 404)

class ConflictError(TimeTrackerError):
    def __init__(self, message: str, current_session: dict = None):
        super().__init__("CONFLICT", message, 409)
        self.current_session = current_session

class RateLimitError(TimeTrackerError):
    def __init__(self, retry_after: int):
        super().__init__("RATE_LIMITED", "Too many requests", 429)
        self.retry_after = retry_after
```

## Testing Strategy

### 测试框架选择

- **单元测试**: pytest
- **属性测试**: hypothesis (Python PBT 库)
- **API 测试**: pytest + httpx (FastAPI TestClient)

### 双重测试方法

本项目采用单元测试和属性测试相结合的方式：

1. **单元测试**: 验证特定示例、边界情况和错误条件
2. **属性测试**: 验证所有输入上的通用属性

两者互补，共同提供全面的测试覆盖。

### 属性测试配置

- 每个属性测试至少运行 100 次迭代
- 每个测试用注释标注对应的设计文档属性
- 标注格式: **Feature: time-tracker, Property {number}: {property_text}**

### 测试文件结构

```
tests/
├── conftest.py              # pytest fixtures
├── test_models.py           # Pydantic 模型验证测试
├── test_repositories.py     # Repository 层测试
├── test_services.py         # Service 层测试
├── test_api_logs.py         # Logs API 端点测试
├── test_api_sessions.py     # Sessions API 端点测试
├── test_api_auth.py         # 认证相关测试
├── test_csv_export.py       # CSV 导出测试
├── test_rate_limit.py       # 速率限制测试
├── test_security.py         # 安全相关测试
└── properties/              # 属性测试
    ├── test_log_properties.py
    ├── test_session_properties.py
    ├── test_csv_properties.py
    └── test_auth_properties.py
```

### 属性测试示例

```python
from hypothesis import given, strategies as st, settings

# Feature: time-tracker, Property 1: 日志创建完整性
@settings(max_examples=100)
@given(
    category=st.text(min_size=1, max_size=50),
    task=st.text(min_size=1, max_size=200),
    note=st.one_of(st.none(), st.text(max_size=1000)),
    location=st.one_of(st.none(), st.text(max_size=100)),
    mood=st.one_of(st.none(), st.text(max_size=20))
)
def test_log_creation_completeness(client, category, task, note, location, mood):
    """
    **Validates: Requirements 1.1, 1.2, 1.3**
    """
    response = client.post("/api/v1/logs", json={
        "category": category,
        "task": task,
        "note": note,
        "location": location,
        "mood": mood
    })
    assert response.status_code == 201
    data = response.json()
    assert "id" in data
    assert "created_at" in data
    assert data["category"] == category
    assert data["task"] == task
    # 验证可选字段
    if note is not None:
        assert data["note"] == note
    if location is not None:
        assert data["location"] == location
    if mood is not None:
        assert data["mood"] == mood
```

### 单元测试重点

1. **边界情况**: 空字符串、最大长度、特殊字符
2. **错误条件**: 缺少必填字段、无效格式、认证失败
3. **集成点**: 数据库操作、中间件链
4. **特定示例**: 健康检查响应、CSV BOM 头

