# Time Tracker

个人时间记录系统 - 轻量级时间管理工具，支持通过 iOS 快捷指令快速开始/停止计时。

## 功能特性

- **计时功能**: 开始/停止计时，自动计算时长
- **Sessions Web 界面**: 浏览器查看记录，支持分页和过滤
- **CSV 导出**: 导出数据用于周复盘分析
- **安全认证**: API Key 认证 + Basic Auth 保护

## 快速开始

### 推荐：NAS 用户直接部署

如果你的 NAS（群晖、威联通等）支持 Docker，这是最简单的部署方式。

1.  **创建文件夹**
    在 NAS 上创建一个文件夹（例如 `time-tracker`），并在其中新建一个名为 `docker-compose.yml` 的文件。

2.  **配置文件内容**
    将以下内容复制到 `docker-compose.yml` 中：

    ```yaml
    services:
      app:
        # 支持 Intel (amd64) 和 Apple (arm64) 架构
        image: xc9973/time-tracker:latest
        container_name: time-tracker
        restart: always
        ports:
          - "7070:8000"
        volumes:
          - ./data:/app/data
        environment:
          # 必须修改：设置你的 API 密钥（至少 32 字符）
          - TIMELOG_API_KEY=your_secret_key_at_least_32_chars_long_please_change_me
          # 必须修改：Web 界面登录密码
          - TIMELOG_ADMIN_PASSWORD=your_web_password
          # 可选：设置时区
          - TIMELOG_TZ=Asia/Shanghai
    ```

3.  **启动服务**
    *   **群晖/威联通**：在 Container Manager / Docker 套件中选择“项目”，指向该文件夹启动。
    *   **命令行**：进入该目录运行 `docker-compose up -d`。

4.  **访问**
    *   API 地址：`http://NAS_IP:7070`
    *   Web 界面：`http://NAS_IP:7070/web/sessions`

### 环境要求


- Go 1.21+
- SQLite3

### 本地运行

1. 克隆项目并安装依赖：

```bash
git clone <repository-url>
cd time-tracker
go mod download
```

2. 配置环境变量：

```bash
cp .env.example .env
# 编辑 .env 文件，设置 TIMELOG_API_KEY（至少 32 字符）
```

3. 启动服务：

```bash
# 加载环境变量并运行
export $(cat .env | xargs) && go run ./cmd/server
```

服务将在 `http://localhost:8000` 启动。

> 注意：默认只提供 Sessions 计时功能，所有日志/统计功能已移除。

### Docker 运行

```bash
# 构建镜像
docker build -t time-tracker .

# 运行容器
docker run -d \
  -p 8000:8000 \
  -v $(pwd)/data:/data \
  -e TIMELOG_API_KEY="your-secret-api-key-at-least-32-characters" \
  -e TIMELOG_TZ="Asia/Shanghai" \
  -e TIMELOG_BASIC_USER="admin" \
  -e TIMELOG_BASIC_PASS="your-password" \
  time-tracker
```

## 配置说明

| 环境变量 | 必填 | 默认值 | 说明 |
|---------|------|--------|------|
| `TIMELOG_API_KEY` | ✅ | - | API 认证密钥（至少 32 字符） |
| `TIMELOG_DB_PATH` | ❌ | `./timelog.db` | SQLite 数据库路径 |
| `TIMELOG_TZ` | ❌ | `UTC` | 显示时区 |
| `TIMELOG_BASIC_USER` | ❌ | - | Web Basic Auth 用户名 |
| `TIMELOG_BASIC_PASS` | ❌ | - | Web Basic Auth 密码 |
| `TIMELOG_RATE_LIMIT` | ❌ | `100` | 每分钟请求限制 |
| `TIMELOG_PORT` | ❌ | `8000` | 服务端口 |

## API 文档

### 认证

API 端点需要在请求头中携带 `X-API-Key`：

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8000/api/v1/sessions
```

### 端点列表

#### 健康检查

```
GET /healthz
```

无需认证，返回 `{"ok": true}`。

#### Sessions API

```
POST /api/v1/sessions/start    # 开始计时
POST /api/v1/sessions/stop     # 停止计时
GET  /api/v1/sessions/current  # 当前状态
GET  /api/v1/sessions          # 查询列表
GET  /api/v1/sessions.csv      # 导出 CSV
```

**开始计时示例：**

```bash
curl -X POST http://localhost:8000/api/v1/sessions/start \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "category": "学习",
    "task": "英语听力"
  }'
```

**停止计时示例：**

```bash
curl -X POST http://localhost:8000/api/v1/sessions/stop \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "note": "完成25分钟",
    "mood": "😀好"
  }'
```

### Web 界面

访问 `/web/sessions` 查看记录（需要 Basic Auth 认证，如果已配置）。

## iOS 快捷指令集成

### 计时快捷指令

创建一个快捷指令用于开始/停止计时：

1. 添加"获取 URL 内容"操作
2. URL: `https://your-domain.com/api/v1/sessions/start` 或 `/stop`
3. 方法: POST
4. JSON: `{ "category": "默认分类", "task": "默认任务" }`

也可以创建两个快捷指令，分别对应开始与停止。

## 部署方式

### 方式一：云服务器 + Caddy

```
# Caddyfile
time.example.com {
    reverse_proxy localhost:8000
}
```

### 方式二：家庭设备 + Cloudflared

```bash
cloudflared tunnel --url http://localhost:8000
```

## 开发

### 运行测试

```bash
go test ./...
```

### 项目结构

```
.
├── cmd/server/          # 应用入口
├── internal/
│   ├── auth/            # 认证模块
│   ├── database/        # 数据库模块
│   ├── errors/          # 错误处理
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件
│   ├── models/          # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   └── validation/      # 输入验证
├── templates/           # HTML 模板
├── Dockerfile
├── go.mod
└── README.md
```

## License

MIT
