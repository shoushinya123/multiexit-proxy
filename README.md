# MultiExit Proxy

一个支持多出口IP的SNAT代理系统，允许客户端通过服务端的多个公网IP访问外网服务。系统采用加密传输和流量混淆技术，确保通信安全。

## ✨ 核心特性

- ✅ **多出口IP管理**：支持多个公网IP，实现SNAT源地址转换
- ✅ **多种分配策略**：轮询、按端口、按目标地址分配
- ✅ **加密传输**：TLS 1.3 + 自定义AEAD加密（ChaCha20-Poly1305）
- ✅ **流量混淆**：TLS SNI伪装、包大小混淆、时间混淆
- ✅ **SOCKS5/HTTP代理**：支持标准代理协议
- ✅ **Web管理界面**：可视化管理和监控
- ✅ **订阅功能**：支持订阅链接自动配置
- ✅ **自动化部署**：一键部署脚本

## 🏗️ 系统架构

```
┌──────────┐         ┌──────────────┐         ┌─────────────┐
│  客户端   │────────▶│  代理服务端   │────────▶│  目标服务    │
│ (Client) │  加密   │ (Proxy Svr)  │  SNAT   │ (Target)    │
└──────────┘  隧道   └──────────────┘         └─────────────┘
                        │
                        │ 管理多个出口IP
                        ▼
                  ┌─────────────┐
                  │  EIP Pool   │
                  │ IP1, IP2... │
                  └─────────────┘
```

## 🚀 快速开始

### 服务端部署

```bash
# 1. 编译并打包
./deploy-server.sh

# 2. 上传到服务器
scp -r deploy/server/* root@YOUR_SERVER:/opt/multiexit-proxy/

# 3. 在服务器上运行自动化部署
ssh root@YOUR_SERVER
cd /opt/multiexit-proxy
chmod +x step.sh
sudo bash step.sh

# 4. 启动服务
sudo systemctl start multiexit-proxy
```

### 客户端使用

```bash
# 使用订阅链接（推荐）
./client -subscribe "http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN"

# 或使用配置文件
./client -config configs/client.json
```

### Web管理界面

访问：`http://YOUR_SERVER:8080`

- 用户名：admin
- 密码：部署时生成的密码

## 📖 文档

- [设计文档](DESIGN.md) - 完整架构设计
- [技术规范](TECHNICAL_SPEC.md) - 技术实现细节
- [使用说明](USAGE.md) - 详细使用指南
- [测试指南](TESTING.md) - 客户端代理测试
- [订阅功能](SUBSCRIPTION.md) - 订阅功能说明
- [部署指南](DEPLOYMENT.md) - 部署文档

## 🔧 技术栈

- **语言**：Go 1.21+
- **加密**：TLS 1.3 + ChaCha20-Poly1305
- **网络**：iptables + iproute2 (SNAT)
- **协议**：自定义协议 + SOCKS5
- **Web框架**：Gorilla Mux

## 📋 IP分配策略

1. **轮询策略（Round Robin）** - 按连接顺序轮流使用各个IP
2. **按端口分配** - 特定端口范围映射到特定IP
3. **按目标地址分配** - 相同目标使用相同出口IP

## 🔐 安全特性

- 🔐 **端到端加密**：TLS 1.3 + AEAD加密
- 🎭 **流量混淆**：SNI伪装、包大小混淆
- 🛡️ **防重放攻击**：时间戳 + 随机数验证
- 🔑 **认证机制**：预共享密钥（PSK）

## 📦 项目结构

```
multiexit-proxy/
├── cmd/
│   ├── client/           # 客户端入口
│   └── server/           # 服务端入口
├── internal/
│   ├── config/           # 配置管理
│   ├── protocol/         # 协议实现
│   ├── proxy/            # 代理核心
│   ├── snat/             # SNAT管理
│   ├── transport/        # 传输层
│   └── web/              # Web管理界面
├── pkg/
│   └── subscribe/        # 订阅功能
├── configs/              # 配置文件示例
├── scripts/              # 部署脚本
└── functiongraph/        # FunctionGraph适配（可选）
```

## ⚠️ 注意事项

1. 本系统需要root权限或CAP_NET_ADMIN权限才能配置iptables规则
2. 服务端需要绑定多个公网IP（EIP）
3. SNAT功能仅在Linux系统上支持
4. 确保防火墙规则允许相关端口
5. 遵守当地法律法规使用

## 📄 许可证

本项目仅供学习和研究使用。

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📝 更新日志

- v1.0.0 - 初始版本
  - 多出口IP管理
  - SNAT功能实现
  - Web管理界面
  - 订阅功能
  - 自动化部署
