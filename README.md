# Time Tracker

个人时间记录系统 - 轻量级时间管理工具，支持通过 iOS 快捷指令快速开始/停止计时。

## 功能特性

- **计时功能**: 开始/停止计时，自动计算时长
- **标签系统**: 支持为记录打标签，方便分类和筛选
- **Sessions Web 界面**: 浏览器查看记录，支持分页和过滤
- **CSV 导出**: 导出数据用于周复盘分析
- **安全认证**: API Key 认证 + Basic Auth 保护

## 快速开始

### 推荐：服务器 Docker 部署

这是最简单的部署方式，适用于云服务器、NAS（群晖、威联通等）。

#### 1. 准备配置文件

```bash
# 克隆项目
git clone https://github.com/xc9973/time-tracker.git
cd time-tracker

# 复制环境变量模板
cp env.example .env

# 编辑 .env 文件，修改以下必填项：
# - TIMELOG_API_KEY: 设置你的 API 密钥（至少 32 字符）
# - TIMELOG_BASIC_USER: Web 界面用户名
# - TIMELOG_BASIC_PASS: Web 界面密码
```

#### 2. 一键部署

```bash
# 运行部署脚本
./deploy.sh

# 或手动部署
docker-compose up -d
```

#### 3. 访问服务

- **API 地址**: `http://your-server:7070`
- **Web 界面**: `http://your-server:7070/web/sessions`
- **健康检查**: `http://your-server:7070/healthz`

### 使用 Docker Hub 镜像

```bash
docker run -d \
  --name time-tracker \
  -p 7070:8000 \
  -v $(pwd)/data:/data \
  -e TIMELOG_API_KEY="your-secret-api-key-at-least-32-characters" \
  -e TIMELOG_TZ="Asia/Shanghai" \
  -e TIMELOG_BASIC_USER="admin" \
  -e TIMELOG_BASIC_PASS="your-password" \
  xc9973/time-tracker:latest
```

### 本地开发运行

#### 环境要求

- Go 1.21+
- SQLite3

#### 运行步骤

```bash
# 1. 安装依赖
go mod download

# 2. 配置环境变量
cp env.example .env
# 编辑 .env 文件

# 3. 运行服务
export $(cat .env | xargs) && go run ./cmd/server
```

服务将在 `http://localhost:7070` 启动。

## 配置说明

| 环境变量 | 必填 | 默认值 | 说明 |
|---------|------|--------|------|
| `TIMELOG_API_KEY` | ✅ | - | API 认证密钥（至少 32 字符） |
| `TIMELOG_DB_PATH` | ❌ | `./timelog.db` | SQLite 数据库路径 |
| `TIMELOG_TZ` | ❌ | `UTC` | 显示时区（如 `Asia/Shanghai`） |
| `TIMELOG_BASIC_USER` | ❌ | - | Web Basic Auth 用户名 |
| `TIMELOG_BASIC_PASS` | ❌ | - | Web Basic Auth 密码 |
| `TIMELOG_RATE_LIMIT` | ❌ | `100` | 每分钟请求限制 |
| `TIMELOG_PORT` | ❌ | `7070` | 服务端口 |

## API 文档

### 认证方式

API 端点支持两种认证方式：

1. **API Key**（推荐用于程序调用）
   ```bash
   curl -H "X-API-Key: your-api-key" http://localhost:7070/api/v1/sessions
   ```

2. **Basic Auth**（用于 Web 界面）
   ```bash
   curl -u admin:password http://localhost:7070/api/v1/sessions
   ```

### Sessions API

```
POST /api/v1/sessions/start    # 开始计时
POST /api/v1/sessions/stop     # 停止计时
GET  /api/v1/sessions/current  # 当前状态
GET  /api/v1/sessions          # 查询列表
GET  /sessions.csv             # 导出 CSV
```

**开始计时示例：**

```bash
curl -X POST http://localhost:7070/api/v1/sessions/start \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "category": "学习",
    "task": "英语听力"
  }'
```

**停止计时示例：**

```bash
curl -X POST http://localhost:7070/api/v1/sessions/stop \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "note": "完成25分钟",
    "mood": "😀好"
  }'
```

### Tags API

```
POST   /api/v1/tags              # 创建标签
GET    /api/v1/tags              # 获取标签列表
GET    /api/v1/tags/:id          # 获取单个标签
POST   /api/v1/sessions/:id/tags # 为记录分配标签
DELETE /api/v1/sessions/:id/tags/:tag_id # 移除记录标签
GET    /api/v1/sessions/:id/tags # 获取记录的标签
```

**创建标签示例：**

```bash
curl -X POST http://localhost:7070/api/v1/tags \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "工作",
    "color": "#3B82F6"
  }'
```

**为记录分配标签：**

```bash
curl -X POST http://localhost:7070/api/v1/sessions/1/tags \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "tag_ids": [1, 2, 3]
  }'
```

### Web 界面

访问 `/web/sessions` 查看记录（需要 Basic Auth 认证，如果已配置）。

## iOS 快捷指令集成

### 计时快捷指令

创建快捷指令用于开始/停止计时：

1. 添加"获取 URL 内容"操作
2. URL: `https://your-domain.com/api/v1/sessions/start` 或 `/stop`
3. 方法: POST
4. 请求头: `X-API-Key: your-api-key`
5. JSON: `{ "category": "默认分类", "task": "默认任务" }`

也可以创建两个快捷指令，分别对应开始与停止。

### 打标签快捷指令

停止计时时可以同时打标签：

```json
{
  "note": "完成项目开发",
  "mood": "😀好"
}
```

然后在服务器上为该记录分配标签，或在 Web 界面手动管理。

## 部署架构

### Docker Compose 配置

项目包含优化后的 `docker-compose.yml`，支持：

- ✅ 健康检查
- ✅ 资源限制（CPU 1核/512MB）
- ✅ 日志管理（单文件最大 10MB）
- ✅ 自动重启
- ✅ 数据持久化

### 反向代理配置

#### Caddy

```
time.example.com {
    reverse_proxy localhost:7070
}
```

#### Nginx

```nginx
location / {
    proxy_pass http://localhost:7070;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## 常用命令

```bash
# 查看日志
docker-compose logs -f

# 查看状态
docker-compose ps

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 重新构建
docker-compose up -d --build
```

## 项目结构

```
.
├── cmd/server/          # 应用入口（67 行简洁代码）
├── internal/
│   ├── app/             # 依赖注入与路由组装
│   ├── shared/          # 共享包（auth/database/middleware/errors/...）
│   ├── sessions/        # Sessions 模块（完整的 MVC 结构）
│   ├── tags/            # Tags 模块（完整的 MVC 结构）
│   ├── web/             # Web 模块
│   └── handler/         # 旧 SessionsHandler（待迁移）
├── templates/           # HTML 模板
├── Dockerfile
├── docker-compose.yml
├── deploy.sh            # 一键部署脚本
├── env.example          # 环境变量模板
└── README.md
```

## 开发

### 运行测试

```bash
# 全部测试
go test ./...

# 单个包测试
go test ./internal/sessions/...

# 详细输出
go test -v ./...
```

### 构建二进制

```bash
go build -o time-tracker ./cmd/server
```

## License

MIT
