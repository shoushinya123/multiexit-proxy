#!/bin/bash

# 启动PostgreSQL数据库（仅数据库，不启动整个应用）

set -e

echo "=========================================="
echo "启动 PostgreSQL 数据库"
echo "=========================================="

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker未运行，请先启动Docker"
    exit 1
fi

# 检查是否已存在容器
if docker ps -a | grep -q multiexit-proxy-postgres; then
    if docker ps | grep -q multiexit-proxy-postgres; then
        echo "✅ PostgreSQL容器已在运行"
        docker ps | grep multiexit-proxy-postgres
    else
        echo "📦 启动已存在的PostgreSQL容器..."
        docker start multiexit-proxy-postgres
        echo "✅ PostgreSQL容器已启动"
    fi
else
    echo "📦 创建并启动PostgreSQL容器..."
    docker-compose up -d postgres
    echo "✅ PostgreSQL容器已创建并启动"
fi

# 等待数据库就绪
echo ""
echo "⏳ 等待数据库就绪..."
for i in {1..30}; do
    if docker exec multiexit-proxy-postgres pg_isready -U multiexit -d multiexit_proxy > /dev/null 2>&1; then
        echo "✅ 数据库已就绪"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ 数据库启动超时"
        exit 1
    fi
    sleep 1
done

echo ""
echo "=========================================="
echo "✅ PostgreSQL 数据库启动完成"
echo "=========================================="
echo ""
echo "📝 连接信息:"
echo "   主机: localhost"
echo "   端口: 5432"
echo "   数据库: multiexit_proxy"
echo "   用户名: multiexit"
echo "   密码: multiexit123"
echo ""
echo "🔗 连接字符串:"
echo "   postgresql://multiexit:multiexit123@localhost:5432/multiexit_proxy"
echo ""
echo "📋 常用命令:"
echo "   查看日志: docker logs -f multiexit-proxy-postgres"
echo "   停止数据库: docker stop multiexit-proxy-postgres"
echo "   进入数据库: docker exec -it multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy"
echo ""



