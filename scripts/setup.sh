#!/bin/bash

# 多出口IP代理系统 - 服务端初始化脚本
# 需要root权限运行

set -e

echo "=== 多出口IP代理系统初始化 ==="

# 检查root权限
if [ "$EUID" -ne 0 ]; then 
    echo "错误: 请使用root权限运行此脚本"
    exit 1
fi

# 检查依赖
echo "检查依赖..."
command -v iptables >/dev/null 2>&1 || { echo "错误: 未安装iptables"; exit 1; }
command -v ip >/dev/null 2>&1 || { echo "错误: 未安装iproute2"; exit 1; }

# 显示当前IP配置
echo ""
echo "当前网络配置:"
ip addr show | grep -E "inet |inet6 " || true

echo ""
echo "请确保已绑定多个公网IP到网络接口"
echo "按Enter继续，或Ctrl+C退出..."
read

# 创建日志目录
mkdir -p /var/log
touch /var/log/multiexit-proxy.log
chmod 644 /var/log/multiexit-proxy.log

echo ""
echo "初始化完成！"
echo ""
echo "下一步："
echo "1. 编辑配置文件: configs/server.yaml"
echo "2. 设置正确的exit_ips、gateway和interface"
echo "3. 运行服务端: sudo ./multiexit-proxy-server -config configs/server.yaml"



