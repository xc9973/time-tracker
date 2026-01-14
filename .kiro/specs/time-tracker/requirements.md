# Requirements Document

## Introduction

个人时间记录系统（Time Tracker）是一个轻量级的时间管理工具，支持通过 iOS 快捷指令快速开始/停止计时，提供 Web 界面查看记录，并支持 CSV 导出用于周复盘分析。系统采用 Go + SQLite 架构，支持云服务器（Caddy）和家庭设备（Cloudflared）两种部署方式。

## Glossary

- **Time_Tracker**: 时间记录系统的核心服务
- **Log**: 打点日志，记录某一时刻正在做的事情
- **Session**: 计时段，包含开始和结束时间，用于计算时长
- **API_Server**: FastAPI 后端服务，提供 REST API
- **Web_Interface**: 网页端界面，用于查看和导出记录
- **Shortcut_Client**: iOS 快捷指令客户端

## Requirements

### Requirement 1: 打点日志记录

**User Story:** As a user, I want to quickly log what I'm doing at any moment, so that I can track my activities throughout the day.

#### Acceptance Criteria

1. WHEN a user submits a log with category and task, THE API_Server SHALL create a new Log record with a UTC timestamp
2. WHEN a user submits a log with optional fields (note, location, mood), THE API_Server SHALL store all provided fields
3. WHEN a log is successfully created, THE API_Server SHALL return the complete Log object including id and created_at
4. IF a log request is missing category or task, THEN THE API_Server SHALL return a 400 error with a descriptive message
5. WHEN a user queries logs with limit and offset, THE API_Server SHALL return paginated results ordered by created_at descending
6. WHEN a user queries logs with category filter, THE API_Server SHALL return only logs matching that category
7. WHEN a user queries logs with search term (q), THE API_Server SHALL return logs where task or note contains the search term

### Requirement 2: 计时功能

**User Story:** As a user, I want to start and stop timing sessions, so that I can track how long I spend on specific tasks.

#### Acceptance Criteria

1. WHEN a user starts a session with category and task, THE API_Server SHALL create a new Session with status "running" and started_at timestamp
2. WHILE a session is running, IF a user attempts to start another session, THEN THE API_Server SHALL return a 409 Conflict with the current running session info
3. WHEN a user stops the current session, THE API_Server SHALL set ended_at, calculate duration_sec, and update status to "stopped"
4. WHEN a user stops a session with optional fields (note, mood, location), THE API_Server SHALL update those fields on the session
5. IF no session is currently running, WHEN a user attempts to stop, THEN THE API_Server SHALL return a 404 error
6. WHEN a user queries current session, THE API_Server SHALL return the running session or indicate none is active
7. WHEN a user queries sessions with filters (status, category, limit, offset), THE API_Server SHALL return matching paginated results

### Requirement 3: CSV 导出

**User Story:** As a user, I want to export my logs and sessions as CSV files, so that I can analyze my time usage in Excel or Numbers.

#### Acceptance Criteria

1. WHEN a user requests logs CSV export, THE API_Server SHALL return a text/csv response with proper headers
2. WHEN a user requests sessions CSV export, THE API_Server SHALL return a text/csv response with proper headers
3. WHEN exporting CSV, THE API_Server SHALL use UTF-8 encoding with BOM for Excel compatibility
4. WHEN exporting CSV, THE API_Server SHALL apply the same query filters as the list endpoints
5. WHEN exporting sessions CSV, THE API_Server SHALL include duration in human-readable format (hours:minutes:seconds)

### Requirement 4: API 认证与安全

**User Story:** As a user, I want my data protected by robust authentication and security measures, so that only I can access my time records and my data is safe from attacks.

#### Acceptance Criteria

1. WHEN an API request to /api/* lacks X-API-Key header, THE API_Server SHALL return 401 Unauthorized
2. WHEN an API request has invalid X-API-Key, THE API_Server SHALL return 401 Unauthorized
3. WHEN an API request has valid X-API-Key, THE API_Server SHALL process the request normally
4. THE API_Server SHALL read the API key from TIMELOG_API_KEY environment variable
5. THE API_Server SHALL use constant-time comparison for API key validation to prevent timing attacks
6. WHEN an API key is configured, THE API_Server SHALL require it to be at least 32 characters long
7. THE API_Server SHALL implement rate limiting of 100 requests per minute per IP address
8. IF rate limit is exceeded, THEN THE API_Server SHALL return 429 Too Many Requests with Retry-After header
9. THE API_Server SHALL add security headers (X-Content-Type-Options, X-Frame-Options, Content-Security-Policy) to all responses
10. WHEN logging requests, THE API_Server SHALL NOT log the full API key value (only first 4 characters for debugging)
11. THE Web_Interface SHALL support HTTP Basic Auth for browser access (username/password from TIMELOG_BASIC_USER and TIMELOG_BASIC_PASS)
12. WHEN Basic Auth credentials are configured, THE Web_Interface SHALL require authentication for /web/* and /*.csv endpoints
13. THE API_Server SHALL validate and sanitize all input data to prevent injection attacks
14. WHEN returning error responses, THE API_Server SHALL NOT expose internal system details or stack traces

### Requirement 5: Web 界面

**User Story:** As a user, I want to view my logs and sessions in a web browser, so that I can review my time records visually.

#### Acceptance Criteria

1. WHEN a user visits /web/logs, THE Web_Interface SHALL display a paginated list of logs with category, task, note, location, mood, and timestamp
2. WHEN a user visits /web/sessions, THE Web_Interface SHALL display a paginated list of sessions with task, category, start time, end time, and duration
3. WHEN viewing logs or sessions, THE Web_Interface SHALL provide a search/filter input for category
4. WHEN viewing logs or sessions, THE Web_Interface SHALL provide a "导出 CSV" button linking to the CSV endpoint
5. THE Web_Interface SHALL display timestamps in the configured timezone (TIMELOG_TZ)

### Requirement 6: 健康检查

**User Story:** As a system operator, I want a health check endpoint, so that I can monitor if the service is running.

#### Acceptance Criteria

1. WHEN a request is made to /healthz, THE API_Server SHALL return {"ok": true} with status 200
2. THE /healthz endpoint SHALL NOT require authentication

### Requirement 7: 数据持久化

**User Story:** As a user, I want my data stored reliably, so that I don't lose my time records.

#### Acceptance Criteria

1. THE API_Server SHALL store all data in SQLite database at the path specified by TIMELOG_DB_PATH
2. WHEN the application starts, THE API_Server SHALL create necessary tables if they don't exist
3. THE API_Server SHALL create indexes on created_at and category for logs table
4. THE API_Server SHALL create indexes on started_at, status, and category for sessions table
5. WHEN storing timestamps, THE API_Server SHALL use RFC3339 format in UTC

### Requirement 8: 配置管理

**User Story:** As a system operator, I want to configure the service via environment variables, so that I can deploy it in different environments.

#### Acceptance Criteria

1. THE API_Server SHALL read TIMELOG_API_KEY for API authentication
2. THE API_Server SHALL read TIMELOG_DB_PATH for database location (default: ./timelog.db)
3. THE API_Server SHALL read TIMELOG_TZ for display timezone (default: UTC)
4. THE API_Server SHALL read TIMELOG_BASIC_USER and TIMELOG_BASIC_PASS for Web Basic Auth (optional)
5. THE API_Server SHALL read TIMELOG_RATE_LIMIT for rate limiting threshold (default: 100 requests/minute)
6. IF TIMELOG_API_KEY is missing or too short, WHEN the application starts, THEN THE API_Server SHALL fail with a clear error message
7. THE API_Server SHALL NOT log sensitive configuration values (API keys, passwords) at startup
