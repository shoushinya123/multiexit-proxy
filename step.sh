#!/bin/bash
# 一键自动化部署脚本 - 在服务器上运行

set -e

echo "=========================================="
echo "  多出口IP代理系统 - 一键自动化部署"
echo "=========================================="
echo ""

# 检查是否为root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ 错误: 请使用root权限运行此脚本"
    echo "   使用: sudo bash step.sh"
    exit 1
fi

WORK_DIR="/opt/multiexit-proxy"
mkdir -p $WORK_DIR
cd $WORK_DIR

echo "📍 工作目录: $WORK_DIR"
echo ""

# 1. 检查并安装依赖
echo "🔍 步骤 1/7: 检查系统依赖..."
command -v iptables >/dev/null 2>&1 || { echo "   安装 iptables..."; apt-get update -qq && apt-get install -y iptables >/dev/null 2>&1; }
command -v ip >/dev/null 2>&1 || { echo "   安装 iproute2..."; apt-get update -qq && apt-get install -y iproute2 >/dev/null 2>&1; }
command -v openssl >/dev/null 2>&1 || { echo "   安装 openssl..."; apt-get update -qq && apt-get install -y openssl >/dev/null 2>&1; }
echo "   ✅ 依赖检查完成"
echo ""

# 2. 自动检测公网IP地址
echo "🔍 步骤 2/7: 自动检测公网IP地址..."
PUBLIC_IPS=$(ip addr show | grep -E "inet [0-9]+\.[0-9]+\.[0-9]+\.[0-9]+" | grep -v "127.0.0.1" | awk '{print $2}' | cut -d'/' -f1 | grep -vE "^(10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.)" | sort -u)

if [ -z "$PUBLIC_IPS" ]; then
    echo "   ⚠️  未自动检测到公网IP，请手动输入（每行一个，输入空行结束）:"
    PUBLIC_IPS=""
    while true; do
        read -p "   IP地址: " ip
        [ -z "$ip" ] && break
        PUBLIC_IPS="${PUBLIC_IPS}${ip}\n"
    done
    PUBLIC_IPS=$(echo -e "$PUBLIC_IPS" | grep -v "^$")
fi

IP_COUNT=$(echo "$PUBLIC_IPS" | grep -v "^$" | wc -l)
echo "   ✅ 检测到 $IP_COUNT 个公网IP:"
echo "$PUBLIC_IPS" | grep -v "^$" | while read ip; do
    [ -n "$ip" ] && echo "      • $ip"
done
echo ""

# 3. 自动检测网关和接口
echo "🔍 步骤 3/7: 自动检测网络配置..."
GATEWAY=$(ip route | grep default | awk '{print $3}' | head -n1)
INTERFACE=$(ip route | grep default | awk '{print $5}' | head -n1)

if [ -z "$GATEWAY" ]; then
    echo "   请手动输入网关地址:"
    read -p "   网关: " GATEWAY
fi

if [ -z "$INTERFACE" ]; then
    echo "   请手动输入网络接口名称 (如 eth0):"
    read -p "   接口: " INTERFACE
fi

echo "   ✅ 网关: $GATEWAY"
echo "   ✅ 接口: $INTERFACE"
echo ""

# 4. 生成密钥和证书
echo "🔍 步骤 4/7: 生成安全密钥和证书..."
AUTH_KEY=$(openssl rand -hex 32)
WEB_PASSWORD=$(openssl rand -hex 16)

if [ ! -f "$WORK_DIR/cert.pem" ] || [ ! -f "$WORK_DIR/key.pem" ]; then
    openssl req -x509 -newkey rsa:4096 -keyout $WORK_DIR/key.pem -out $WORK_DIR/cert.pem -days 365 -nodes -subj "/CN=multiexit-proxy" >/dev/null 2>&1
    chmod 600 $WORK_DIR/key.pem
    echo "   ✅ TLS证书已生成"
else
    echo "   ✅ TLS证书已存在，跳过生成"
fi
echo ""

# 5. 创建配置文件
echo "🔍 步骤 5/7: 生成配置文件..."
cat > $WORK_DIR/server.yaml <<EOF
server:
  listen: ":443"
  tls:
    cert: "$WORK_DIR/cert.pem"
    key: "$WORK_DIR/key.pem"
    sni_fake: true
    fake_snis:
      - "cloudflare.com"
      - "google.com"
      - "github.com"

auth:
  method: "psk"
  key: "$AUTH_KEY"

exit_ips:
$(echo "$PUBLIC_IPS" | grep -v "^$" | while read ip; do
    [ -n "$ip" ] && echo "  - \"$ip\""
done)

strategy:
  type: "round_robin"

snat:
  enabled: true
  gateway: "$GATEWAY"
  interface: "$INTERFACE"

logging:
  level: "info"
  file: "/var/log/multiexit-proxy.log"

web:
  enabled: true
  listen: ":8080"
  username: "admin"
  password: "$WEB_PASSWORD"
EOF

echo "   ✅ 配置文件已创建: $WORK_DIR/server.yaml"
echo ""

# 6. 配置系统服务
echo "🔍 步骤 6/7: 配置系统服务..."
cat > /etc/systemd/system/multiexit-proxy.service <<EOF
[Unit]
Description=MultiExit Proxy Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$WORK_DIR
ExecStart=$WORK_DIR/multiexit-proxy-server -config $WORK_DIR/server.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload >/dev/null 2>&1
systemctl enable multiexit-proxy >/dev/null 2>&1
echo "   ✅ 系统服务已配置"
echo ""

# 7. 配置防火墙
echo "🔍 步骤 7/7: 配置防火墙..."
if command -v ufw >/dev/null 2>&1; then
    ufw allow 443/tcp >/dev/null 2>&1 || true
    ufw allow 8080/tcp >/dev/null 2>&1 || true
    echo "   ✅ UFW防火墙已配置"
elif command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --permanent --add-port=443/tcp >/dev/null 2>&1 || true
    firewall-cmd --permanent --add-port=8080/tcp >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
    echo "   ✅ firewalld防火墙已配置"
else
    iptables -A INPUT -p tcp --dport 443 -j ACCEPT >/dev/null 2>&1 || true
    iptables -A INPUT -p tcp --dport 8080 -j ACCEPT >/dev/null 2>&1 || true
    echo "   ✅ iptables防火墙已配置"
fi

# 创建日志文件
touch /var/log/multiexit-proxy.log
chmod 644 /var/log/multiexit-proxy.log

echo ""
echo "=========================================="
echo "  ✅ 部署完成！"
echo "=========================================="
echo ""
echo "📋 配置信息："
echo "   公网IP数量: $IP_COUNT"
echo "   网关地址: $GATEWAY"
echo "   网络接口: $INTERFACE"
echo ""
echo "🔐 认证信息："
echo "   认证密钥: $AUTH_KEY"
echo "   Web用户名: admin"
echo "   Web密码: $WEB_PASSWORD"
echo ""
echo "🌐 访问地址："
SERVER_IP=$(hostname -I | awk '{print $1}')
echo "   Web管理界面: http://$SERVER_IP:8080"
echo ""
echo "📝 下一步操作："
echo "   1. 确保 multiexit-proxy-server 文件已上传到 $WORK_DIR/"
echo "   2. 启动服务: systemctl start multiexit-proxy"
echo "   3. 查看状态: systemctl status multiexit-proxy"
echo "   4. 查看日志: journalctl -u multiexit-proxy -f"
echo ""
echo "⚠️  请保存以上认证信息！"
echo ""



