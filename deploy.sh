#!/bin/bash
# 部署脚本 - 在服务器上运行

set -e

echo "=== Time Tracker 部署脚本 ==="

# 1. 检查 .env 文件
if [ ! -f .env ]; then
    echo "错误: .env 文件不存在"
    echo "请复制 env.example 到 .env 并配置"
    exit 1
fi

# 2. 创建数据目录
echo "创建数据目录..."
mkdir -p /vol1/1000/docker/timejl/data
mkdir -p /vol1/1000/docker/timejl/templates

# 3. 停止旧容器
echo "停止旧容器..."
docker-compose down

# 4. 构建镜像
echo "构建镜像..."
docker-compose build

# 5. 启动服务
echo "启动服务..."
docker-compose up -d

# 6. 查看状态
echo ""
echo "=== 部署完成 ==="
docker-compose ps
echo ""
echo "查看日志: docker-compose logs -f"
echo "停止服务: docker-compose down"
echo "重启服务: docker-compose restart"
