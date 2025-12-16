#!/bin/bash
# 停止服务器脚本

if [ -f server.pid ]; then
    PID=$(cat server.pid)
    if ps -p $PID > /dev/null 2>&1; then
        kill $PID
        echo "✅ 服务器已停止 (PID: $PID)"
        rm server.pid
    else
        echo "⚠️  服务器未运行"
        rm server.pid
    fi
else
    echo "⚠️  未找到 server.pid 文件"
fi

