#!/bin/bash
# 客户端代理测试脚本

echo "=========================================="
echo "  客户端代理测试"
echo "=========================================="
echo ""

# 检查客户端是否运行
CLIENT_PID=$(pgrep -f "client.*configs/client.json" | head -1)

if [ -z "$CLIENT_PID" ]; then
    echo "⚠️  客户端未运行，正在启动..."
    ./client -config configs/client.json > client.log 2>&1 &
    CLIENT_PID=$!
    sleep 2
    
    if ps -p $CLIENT_PID > /dev/null 2>&1; then
        echo "✅ 客户端已启动 (PID: $CLIENT_PID)"
        echo $CLIENT_PID > client.pid
    else
        echo "❌ 客户端启动失败"
        tail -10 client.log
        exit 1
    fi
else
    echo "✅ 客户端正在运行 (PID: $CLIENT_PID)"
fi

echo ""
echo "=========================================="
echo "  测试 1: SOCKS5 代理连接"
echo "=========================================="

# 使用curl通过SOCKS5代理测试
echo "测试通过 SOCKS5 代理访问 httpbin.org..."
RESULT=$(curl -s --max-time 10 --socks5-hostname 127.0.0.1:1080 http://httpbin.org/ip 2>&1)

if [ $? -eq 0 ] && echo "$RESULT" | grep -q "origin"; then
    echo "✅ SOCKS5 代理测试成功！"
    echo "响应: $RESULT"
else
    echo "❌ SOCKS5 代理测试失败"
    echo "错误: $RESULT"
fi

echo ""
echo "=========================================="
echo "  测试 2: HTTP 代理连接"
echo "=========================================="

# 使用curl通过HTTP代理测试
echo "测试通过 HTTP 代理访问 httpbin.org..."
RESULT=$(curl -s --max-time 10 --proxy http://127.0.0.1:8080 http://httpbin.org/ip 2>&1)

if [ $? -eq 0 ] && echo "$RESULT" | grep -q "origin"; then
    echo "✅ HTTP 代理测试成功！"
    echo "响应: $RESULT"
else
    echo "❌ HTTP 代理测试失败"
    echo "错误: $RESULT"
fi

echo ""
echo "=========================================="
echo "  测试 3: 访问测试网站"
echo "=========================================="

echo "测试访问 ifconfig.me (查看出口IP)..."
IP=$(curl -s --max-time 10 --socks5-hostname 127.0.0.1:1080 https://ifconfig.me 2>&1)

if [ $? -eq 0 ] && echo "$IP" | grep -qE "^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$"; then
    echo "✅ 访问测试网站成功！"
    echo "当前出口IP: $IP"
else
    echo "⚠️  无法获取IP (可能需要HTTPS支持)"
    echo "响应: $IP"
fi

echo ""
echo "=========================================="
echo "  测试 4: 代理性能测试"
echo "=========================================="

echo "测试代理响应速度..."
START_TIME=$(date +%s%N)
curl -s --max-time 10 --socks5-hostname 127.0.0.1:1080 http://httpbin.org/get > /dev/null 2>&1
END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))

if [ $? -eq 0 ]; then
    echo "✅ 请求完成，耗时: ${DURATION}ms"
else
    echo "❌ 请求失败"
fi

echo ""
echo "=========================================="
echo "  代理配置信息"
echo "=========================================="
echo ""
echo "SOCKS5 代理:"
echo "  export ALL_PROXY=socks5://127.0.0.1:1080"
echo ""
echo "HTTP 代理:"
echo "  export http_proxy=http://127.0.0.1:8080"
echo "  export https_proxy=http://127.0.0.1:8080"
echo ""
echo "使用示例:"
echo "  curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/ip"
echo "  curl --proxy http://127.0.0.1:8080 http://httpbin.org/ip"
echo ""

