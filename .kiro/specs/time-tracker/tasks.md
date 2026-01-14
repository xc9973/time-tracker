# Implementation Plan: Time Tracker

## Overview

基于 Go + SQLite 的个人时间记录系统实现计划。采用分层架构，从底层数据访问逐步构建到 HTTP API 端点和 Web 界面。使用 Go 1.21+，go test + rapid (PBT) 进行测试。

## Tasks

- [x] 1. 项目初始化和基础配置
  - [x] 1.1 创建项目结构和依赖配置
    - 创建 `go.mod` 配置依赖
    - 创建目录结构: `cmd/`, `internal/config/`, `internal/database/`, `internal/models/`, `internal/repository/`, `internal/service/`, `internal/handler/`, `internal/middleware/`, `templates/`
    - _Requirements: 8.1-8.7_

  - [x] 1.2 实现配置模块 (`internal/config/config.go`)
    - 从环境变量读取配置 (TIMELOG_API_KEY, TIMELOG_DB_PATH, TIMELOG_TZ, TIMELOG_BASIC_USER, TIMELOG_BASIC_PASS, TIMELOG_RATE_LIMIT)
    - 验证 API Key 至少 32 字符
    - 设置默认值
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7_

  - [x] 1.3 编写配置模块单元测试
    - 测试环境变量读取
    - 测试 API Key 长度验证
    - 测试默认值
    - _Requirements: 8.6_

- [x] 2. 数据库层实现
  - [x] 2.1 实现数据库模块 (`internal/database/database.go`)
    - SQLite 连接管理
    - 表创建 (logs, sessions)
    - 索引创建
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x] 2.2 实现数据模型 (`internal/models/models.go`)
    - LogCreate, LogResponse
    - SessionStart, SessionStop, SessionResponse
    - PaginatedResponse
    - 字段验证 (长度限制)
    - _Requirements: 1.1, 1.2, 2.1_

  - [x] 2.3 编写模型验证属性测试
    - **Property 2: 日志输入验证**
    - **Validates: Requirements 1.4**

- [x] 3. Repository 层实现
  - [x] 3.1 实现 LogRepository (`internal/repository/log_repository.go`)
    - Create(): 创建日志，返回完整对象
    - List(): 分页查询，支持 category 和 q 过滤
    - Count(): 统计总数
    - _Requirements: 1.1, 1.2, 1.3, 1.5, 1.6, 1.7_

  - [x] 3.2 实现 SessionRepository (`internal/repository/session_repository.go`)
    - Create(): 创建 session
    - GetRunning(): 获取运行中的 session
    - StopRunning(): 停止并更新 session
    - List(): 分页查询，支持 status 和 category 过滤
    - Count(): 统计总数
    - _Requirements: 2.1, 2.3, 2.6, 2.7_

  - [x] 3.3 编写 Repository 层属性测试
    - **Property 1: 日志创建完整性**
    - **Property 17: 时间戳存储格式正确性**
    - **Validates: Requirements 1.1, 1.2, 1.3, 7.5**

- [x] 4. Service 层实现
  - [x] 4.1 实现 LogService (`internal/service/log_service.go`)
    - CreateLog(): 创建日志
    - GetLogs(): 获取分页日志列表
    - ExportCSV(): 导出 CSV (UTF-8 BOM)
    - _Requirements: 1.1, 1.2, 1.3, 1.5, 1.6, 1.7, 3.1, 3.3, 3.4_

  - [x] 4.2 实现 SessionService (`internal/service/session_service.go`)
    - StartSession(): 开始计时，检查并发冲突
    - StopSession(): 停止计时，计算 duration
    - GetCurrent(): 获取当前状态
    - GetSessions(): 获取分页列表
    - ExportCSV(): 导出 CSV (含 duration 格式化)
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.2, 3.4, 3.5_

  - [x] 4.3 编写 Service 层属性测试
    - **Property 3: 日志查询正确性**
    - **Property 4: Session 生命周期**
    - **Property 5: Session 并发控制**
    - **Property 6: Session 停止时更新**
    - **Property 7: Session 查询正确性**
    - **Validates: Requirements 1.5, 1.6, 1.7, 2.1, 2.2, 2.3, 2.4, 2.6, 2.7**

- [x] 5. Checkpoint - 核心业务逻辑验证
  - 确保所有测试通过，如有问题请询问用户

- [x] 6. 认证和中间件实现
  - [x] 6.1 实现认证模块 (`internal/auth/auth.go`)
    - VerifyAPIKey(): 常量时间比较
    - VerifyBasicAuth(): Basic Auth 验证
    - APIKeyMiddleware: HTTP 中间件
    - BasicAuthMiddleware: HTTP 中间件
    - _Requirements: 4.1, 4.2, 4.3, 4.5, 4.11_

  - [x] 6.2 实现速率限制中间件 (`internal/middleware/rate_limit.go`)
    - 基于 IP 的滑动窗口限制
    - 返回 429 和 Retry-After 头
    - _Requirements: 4.7, 4.8_

  - [x] 6.3 实现安全头中间件 (`internal/middleware/security.go`)
    - X-Content-Type-Options, X-Frame-Options, Content-Security-Policy
    - _Requirements: 4.9_

  - [x] 6.4 实现自定义错误和错误处理 (`internal/errors/errors.go`)
    - TimeTrackerError 基础类型
    - ValidationError, NotFoundError, ConflictError, RateLimitError
    - 全局错误处理器
    - _Requirements: 4.14_

  - [x] 6.5 编写认证和安全属性测试
    - **Property 10: API Key 认证正确性**
    - **Property 11: 速率限制正确性**
    - **Property 12: 安全头正确性**
    - **Property 14: 错误响应安全性**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.6, 4.7, 4.8, 4.9, 4.14**

- [x] 7. HTTP Handler 实现
  - [x] 7.1 实现 Logs Handler (`internal/handler/logs.go`)
    - POST /api/v1/logs
    - GET /api/v1/logs
    - GET /api/v1/logs.csv
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 3.1, 3.3, 3.4_

  - [x] 7.2 实现 Sessions Handler (`internal/handler/sessions.go`)
    - POST /api/v1/sessions/start
    - POST /api/v1/sessions/stop
    - GET /api/v1/sessions/current
    - GET /api/v1/sessions
    - GET /api/v1/sessions.csv
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.2, 3.4, 3.5_

  - [x] 7.3 实现 Health Handler (`internal/handler/health.go`)
    - GET /healthz (无需认证)
    - _Requirements: 6.1, 6.2_

  - [x] 7.4 编写 Handler 集成测试
    - 测试完整的请求/响应流程
    - 测试认证流程
    - 测试错误响应格式
    - _Requirements: 1.1-1.7, 2.1-2.7, 6.1, 6.2_

- [x] 8. CSV 导出功能完善
  - [x] 8.1 完善 CSV 导出实现
    - UTF-8 BOM 头
    - Sessions duration 格式化 (H:MM:SS)
    - Content-Disposition 头
    - _Requirements: 3.1, 3.2, 3.3, 3.5_

  - [x] 8.2 编写 CSV 导出属性测试
    - **Property 8: CSV 导出格式正确性**
    - **Property 9: CSV 导出过滤一致性**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5**

- [x] 9. Checkpoint - API 层验证
  - 确保所有测试通过，如有问题请询问用户

- [x] 10. Web 界面实现
  - [x] 10.1 创建 HTML 模板基础结构
    - base.html: 基础布局
    - 简单 CSS 样式
    - _Requirements: 5.1, 5.2_

  - [x] 10.2 实现 Web Handler (`internal/handler/web.go`)
    - GET /web/logs: 日志列表页面
    - GET /web/sessions: 计时列表页面
    - Basic Auth 保护
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 4.12_

  - [x] 10.3 实现时区转换显示
    - 使用 TIMELOG_TZ 配置转换时间显示
    - _Requirements: 5.5_

  - [x] 10.4 编写 Web 界面属性测试
    - **Property 15: Web Basic Auth 正确性**
    - **Property 16: 时区显示正确性**
    - **Validates: Requirements 4.11, 4.12, 5.5**

- [x] 11. 输入验证和安全加固
  - [x] 11.1 实现输入验证和清理
    - 结构体验证
    - 特殊字符处理
    - _Requirements: 4.13_

  - [x] 11.2 编写输入验证安全属性测试
    - **Property 13: 输入验证安全性**
    - **Validates: Requirements 4.13**

- [x] 12. 应用入口和集成
  - [x] 12.1 创建应用入口 (`cmd/server/main.go`)
    - 注册所有 Handler
    - 注册中间件
    - 注册错误处理器
    - 启动时初始化数据库
    - _Requirements: 7.2_

  - [x] 12.2 创建启动脚本和配置文件
    - Dockerfile (可选)
    - .env.example
    - README.md 使用说明
    - _Requirements: 8.1-8.7_

- [x] 13. Final Checkpoint - 完整系统验证
  - 确保所有测试通过
  - 验证所有 API 端点工作正常
  - 如有问题请询问用户

## Notes

- 所有任务都是必做的，包括测试任务
- 每个任务都引用了具体的需求条款以便追溯
- Checkpoint 任务用于阶段性验证
- 属性测试使用 pgregory.net/rapid 库验证通用正确性属性
- 单元测试验证特定示例和边界情况
