#!/bin/bash
# 自动化部署脚本 - 在本地运行，准备部署文件

set -e

echo "=========================================="
echo "  准备部署文件"
echo "=========================================="
echo ""

# 清理旧的部署目录
rm -rf deploy
mkdir -p deploy/server
mkdir -p deploy/client

# 编译服务端（Linux）
echo "📦 编译服务端 (Linux amd64)..."
GOOS=linux GOARCH=amd64 go build -o deploy/server/multiexit-proxy-server ./cmd/server

# 编译客户端（Linux）
echo "📦 编译客户端 (Linux amd64)..."
GOOS=linux GOARCH=amd64 go build -o deploy/client/multiexit-proxy-client ./cmd/client

# 复制配置文件
echo "📋 复制配置文件..."
cp configs/server.yaml.example deploy/server/server.yaml.example
cp configs/client.json.example deploy/client/client.json.example
cp step.sh deploy/server/

# 复制Web静态文件
mkdir -p deploy/server/internal/web/static
cp -r internal/web/static/* deploy/server/internal/web/static/ 2>/dev/null || true

# 复制systemd服务文件
cp deploy/server/multiexit-proxy.service deploy/server/ 2>/dev/null || true

echo ""
echo "=========================================="
echo "  ✅ 编译完成！"
echo "=========================================="
echo ""
echo "📁 部署文件位置:"
echo "   服务端: deploy/server/"
echo "   客户端: deploy/client/"
echo ""
echo "📝 下一步操作:"
echo ""
echo "1️⃣  上传文件到服务器:"
echo "   scp -r deploy/server/* root@YOUR_SERVER_IP:/opt/multiexit-proxy/"
echo ""
echo "2️⃣  登录服务器并运行自动化部署:"
echo "   ssh root@YOUR_SERVER_IP"
echo "   cd /opt/multiexit-proxy"
echo "   chmod +x step.sh"
echo "   sudo bash step.sh"
echo ""
echo "3️⃣  启动服务:"
echo "   systemctl start multiexit-proxy"
echo "   systemctl status multiexit-proxy"
echo ""
echo "4️⃣  访问Web管理界面:"
echo "   http://YOUR_SERVER_IP:8080"
echo ""
