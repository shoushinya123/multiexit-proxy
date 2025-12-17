#!/bin/bash
# 自动化部署脚本

set -e

echo "=== 多出口IP代理系统 - 自动化部署 ==="

# 检查root权限
if [ "$EUID" -ne 0 ]; then 
    echo "错误: 请使用root权限运行此脚本"
    exit 1
fi

# 检查依赖
echo "检查依赖..."
command -v iptables >/dev/null 2>&1 || { echo "安装iptables..."; apt-get update && apt-get install -y iptables; }
command -v ip >/dev/null 2>&1 || { echo "安装iproute2..."; apt-get update && apt-get install -y iproute2; }
command -v openssl >/dev/null 2>&1 || { echo "安装openssl..."; apt-get update && apt-get install -y openssl; }

# 获取工作目录
WORK_DIR="/opt/multiexit-proxy"
mkdir -p $WORK_DIR
cd $WORK_DIR

# 检测公网IP地址
echo ""
echo "=== 检测公网IP地址 ==="
PUBLIC_IPS=$(ip addr show | grep -E "inet [0-9]+\.[0-9]+\.[0-9]+\.[0-9]+" | grep -v "127.0.0.1" | awk '{print $2}' | cut -d'/' -f1 | grep -v "^10\." | grep -v "^172\.1[6-9]\." | grep -v "^172\.2[0-9]\." | grep -v "^172\.3[0-1]\." | grep -v "^192\.168\." | sort -u)

if [ -z "$PUBLIC_IPS" ]; then
    echo "警告: 未检测到公网IP，请手动配置"
    PUBLIC_IPS="1.2.3.4"
fi

IP_COUNT=$(echo "$PUBLIC_IPS" | wc -l)
echo "检测到 $IP_COUNT 个公网IP:"
echo "$PUBLIC_IPS" | while read ip; do
    echo "  - $ip"
done

# 获取网关地址
echo ""
echo "=== 检测网关地址 ==="
GATEWAY=$(ip route | grep default | awk '{print $3}' | head -n1)
if [ -z "$GATEWAY" ]; then
    echo "请手动输入网关地址:"
    read GATEWAY
else
    echo "检测到网关: $GATEWAY"
fi

# 获取网络接口
echo ""
echo "=== 检测网络接口 ==="
INTERFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
if [ -z "$INTERFACE" ]; then
    echo "请手动输入网络接口名称:"
    read INTERFACE
else
    echo "检测到网络接口: $INTERFACE"
fi

# 生成随机密钥
AUTH_KEY=$(openssl rand -hex 32)
echo ""
echo "=== 生成认证密钥 ==="
echo "密钥: $AUTH_KEY"

# 生成TLS证书
echo ""
echo "=== 生成TLS证书 ==="
if [ ! -f "$WORK_DIR/cert.pem" ] || [ ! -f "$WORK_DIR/key.pem" ]; then
    openssl req -x509 -newkey rsa:4096 -keyout $WORK_DIR/key.pem -out $WORK_DIR/cert.pem -days 365 -nodes -subj "/CN=multiexit-proxy"
    chmod 600 $WORK_DIR/key.pem
    echo "TLS证书已生成"
else
    echo "TLS证书已存在，跳过生成"
fi

# 创建配置文件
echo ""
echo "=== 创建配置文件 ==="
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
$(echo "$PUBLIC_IPS" | while read ip; do
    echo "  - \"$ip\""
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
  password: "$(openssl rand -hex 16)"
EOF

echo "配置文件已创建: $WORK_DIR/server.yaml"

# 创建日志文件
touch /var/log/multiexit-proxy.log
chmod 644 /var/log/multiexit-proxy.log

# 安装systemd服务
echo ""
echo "=== 安装系统服务 ==="
if [ -f "$WORK_DIR/multiexit-proxy.service" ]; then
    cp $WORK_DIR/multiexit-proxy.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable multiexit-proxy
    echo "系统服务已安装"
fi

# 配置防火墙
echo ""
echo "=== 配置防火墙 ==="
if command -v ufw >/dev/null 2>&1; then
    ufw allow 443/tcp >/dev/null 2>&1 || true
    ufw allow 8080/tcp >/dev/null 2>&1 || true
    echo "UFW防火墙已配置"
elif command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --permanent --add-port=443/tcp >/dev/null 2>&1 || true
    firewall-cmd --permanent --add-port=8080/tcp >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
    echo "firewalld防火墙已配置"
else
    iptables -A INPUT -p tcp --dport 443 -j ACCEPT >/dev/null 2>&1 || true
    iptables -A INPUT -p tcp --dport 8080 -j ACCEPT >/dev/null 2>&1 || true
    echo "iptables防火墙已配置"
fi

echo ""
echo "=== 部署完成 ==="
echo ""
echo "配置信息:"
echo "  公网IP数量: $IP_COUNT"
echo "  网关地址: $GATEWAY"
echo "  网络接口: $INTERFACE"
echo "  认证密钥: $AUTH_KEY"
WEB_PASSWORD=$(grep "password:" $WORK_DIR/server.yaml | awk '{print $2}' | tr -d '"')
echo "  Web管理密码: $WEB_PASSWORD"
echo ""
echo "下一步:"
echo "  1. 启动服务: systemctl start multiexit-proxy"
echo "  2. 查看状态: systemctl status multiexit-proxy"
echo "  3. 访问Web管理: http://$(hostname -I | awk '{print $1}'):8080"
echo "  4. 用户名: admin"
echo "  5. 密码: $WEB_PASSWORD"





