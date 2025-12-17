#!/bin/bash

# 开发环境启动脚本
# 用于同时启动后端和前端服务

set -e

echo "=========================================="
echo "MultiExit Proxy 开发环境启动"
echo "=========================================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 检查Node.js环境
if ! command -v node &> /dev/null; then
    echo "❌ 错误: 未找到Node.js环境，请先安装Node.js"
    exit 1
fi

# 检查配置文件
if [ ! -f "configs/server.yaml" ]; then
    echo "⚠️  警告: 配置文件不存在，从示例文件复制..."
    cp configs/server.yaml.example configs/server.yaml
    echo "✅ 已创建配置文件，请根据需要修改 configs/server.yaml"
fi

# 检查TLS证书（如果配置了）
CERT_PATH=$(grep -A 1 "tls:" configs/server.yaml | grep "cert:" | awk '{print $2}' | tr -d '"')
if [ -n "$CERT_PATH" ] && [ "$CERT_PATH" != "/path/to/cert.pem" ]; then
    if [ ! -f "$CERT_PATH" ]; then
        echo "⚠️  警告: TLS证书文件不存在: $CERT_PATH"
        echo "   如果不需要TLS，可以修改配置文件"
    fi
fi

# 编译后端
echo ""
echo "📦 编译后端服务..."
go build -o server ./cmd/server
if [ $? -ne 0 ]; then
    echo "❌ 后端编译失败"
    exit 1
fi
echo "✅ 后端编译成功"

# 检查前端依赖
echo ""
echo "📦 检查前端依赖..."
cd frontend-system-design
if [ ! -d "node_modules" ]; then
    echo "📥 安装前端依赖..."
    if command -v pnpm &> /dev/null; then
        pnpm install
    elif command -v npm &> /dev/null; then
        npm install
    else
        echo "❌ 错误: 未找到pnpm或npm"
        exit 1
    fi
fi
cd ..

# 启动后端服务
echo ""
echo "🚀 启动后端服务..."
./server -config configs/server.yaml &
SERVER_PID=$!
echo "✅ 后端服务已启动 (PID: $SERVER_PID)"
echo "   后端API: http://localhost:8080/api"
echo "   默认用户名: admin"
echo "   默认密码: admin123"

# 等待后端启动
echo ""
echo "⏳ 等待后端服务启动..."
sleep 3

# 检查后端是否启动成功
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 后端服务启动失败"
    exit 1
fi

# 启动前端服务
echo ""
echo "🚀 启动前端服务..."
cd frontend-system-design
if command -v pnpm &> /dev/null; then
    pnpm dev &
else
    npm run dev &
fi
FRONTEND_PID=$!
cd ..
echo "✅ 前端服务已启动 (PID: $FRONTEND_PID)"
echo "   前端地址: http://localhost:8081"

# 保存PID到文件
echo $SERVER_PID > .server.pid
echo $FRONTEND_PID > .frontend.pid

echo ""
echo "=========================================="
echo "✅ 服务启动完成！"
echo "=========================================="
echo ""
echo "📝 服务信息:"
echo "   后端API: http://localhost:8080/api"
echo "   前端界面: http://localhost:8081"
echo ""
echo "🔐 登录信息:"
echo "   用户名: admin"
echo "   密码: admin123"
echo ""
echo "🛑 停止服务:"
echo "   ./stop-dev.sh"
echo "   或按 Ctrl+C"
echo ""
echo "📋 查看日志:"
echo "   后端日志: tail -f multiexit-proxy.log"
echo "   前端日志: 查看终端输出"
echo ""

# 等待用户中断
trap "echo ''; echo '正在停止服务...'; kill $SERVER_PID $FRONTEND_PID 2>/dev/null; rm -f .server.pid .frontend.pid; exit" INT TERM

wait

