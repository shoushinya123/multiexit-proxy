# 快速部署指南

## 方法1: 自动化部署（推荐）

### 在本地Mac上：
```bash
# 1. 编译并打包
./deploy-server.sh

# 2. 上传文件到服务器
scp -r deploy/server/* root@YOUR_SERVER_IP:/opt/multiexit-proxy/

# 3. 上传部署脚本
scp step.sh root@YOUR_SERVER_IP:/opt/multiexit-proxy/
```

### 在服务器上：
```bash
# 登录服务器
ssh root@YOUR_SERVER_IP

# 进入目录
cd /opt/multiexit-proxy

# 运行自动化部署脚本
chmod +x step.sh
sudo bash step.sh
```

脚本会自动：
- ✅ 检测公网IP地址
- ✅ 检测网关和网络接口
- ✅ 生成密钥和证书
- ✅ 创建配置文件
- ✅ 配置系统服务
- ✅ 配置防火墙

### 启动服务：
```bash
systemctl start multiexit-proxy
systemctl status multiexit-proxy
```

### 访问Web管理界面：
浏览器打开: `http://YOUR_SERVER_IP:8080`
- 用户名: admin
- 密码: 部署脚本输出的密码

## 方法2: 手动部署

详见 USAGE.md

