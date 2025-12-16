#!/bin/bash
# 简单的代理测试脚本

echo "=========================================="
echo "  简单代理测试"
echo "=========================================="
echo ""

# 检查端口监听
echo "1. 检查端口监听状态:"
echo "   SOCKS5 (1080):"
lsof -i :1080 2>/dev/null | grep LISTEN && echo "     ✅ 监听中" || echo "     ❌ 未监听"

echo "   HTTP代理 (8080):"
lsof -i :8080 2>/dev/null | grep LISTEN && echo "     ✅ 监听中" || echo "     ❌ 未监听"

echo "   服务端 (8443):"
lsof -i :8443 2>/dev/null | grep LISTEN && echo "     ✅ 监听中" || echo "     ❌ 未监听"

echo ""
echo "2. 测试SOCKS5代理基本连接:"
timeout 3 nc -zv 127.0.0.1 1080 2>&1 && echo "     ✅ 端口可连接" || echo "     ❌ 端口不可连接"

echo ""
echo "3. 使用curl测试本地代理:"
echo "   测试本地Web服务 (127.0.0.1:8081)..."
RESULT=$(curl -s --max-time 5 --socks5-hostname 127.0.0.1:1080 http://127.0.0.1:8081 2>&1)

if echo "$RESULT" | grep -q "多出口IP代理系统\|Multiexit"; then
    echo "     ✅ SOCKS5代理工作正常！"
elif [ -n "$RESULT" ]; then
    echo "     ⚠️  收到响应但可能不是预期内容"
    echo "     响应长度: $(echo "$RESULT" | wc -c) 字节"
else
    echo "     ❌ 代理连接失败"
fi

echo ""
echo "4. 查看客户端日志 (最近5行):"
tail -5 client.log 2>/dev/null || echo "     (无日志文件)"

echo ""
echo "5. 测试命令示例:"
echo "   curl --socks5-hostname 127.0.0.1:1080 http://example.com"
echo "   curl --proxy http://127.0.0.1:8080 http://example.com"
echo ""

