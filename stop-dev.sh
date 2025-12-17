#!/bin/bash

# 停止开发环境服务

echo "正在停止服务..."

if [ -f ".server.pid" ]; then
    SERVER_PID=$(cat .server.pid)
    if kill -0 $SERVER_PID 2>/dev/null; then
        kill $SERVER_PID
        echo "✅ 后端服务已停止 (PID: $SERVER_PID)"
    fi
    rm -f .server.pid
fi

if [ -f ".frontend.pid" ]; then
    FRONTEND_PID=$(cat .frontend.pid)
    if kill -0 $FRONTEND_PID 2>/dev/null; then
        kill $FRONTEND_PID
        echo "✅ 前端服务已停止 (PID: $FRONTEND_PID)"
    fi
    rm -f .frontend.pid
fi

# 也尝试通过进程名停止
pkill -f "server -config" 2>/dev/null && echo "✅ 已停止后端进程"
pkill -f "next dev" 2>/dev/null && echo "✅ 已停止前端进程"

echo "✅ 所有服务已停止"



