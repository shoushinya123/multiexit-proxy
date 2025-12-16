#!/bin/bash
# 停止客户端脚本

if [ -f client.pid ]; then
    PID=$(cat client.pid)
    if ps -p $PID > /dev/null 2>&1; then
        kill $PID
        echo "✅ 客户端已停止 (PID: $PID)"
        rm client.pid
    else
        echo "⚠️  客户端未运行"
        rm client.pid
    fi
else
    echo "⚠️  未找到 client.pid 文件"
fi

