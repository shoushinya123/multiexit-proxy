# MultiExit Proxy

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一个高性能的多出口IP代理系统，支持SNAT源地址转换，允许客户端通过服务端的多个公网IP访问外网服务。系统采用TLS 1.3 + AES-GCM硬件加速加密，性能优化后吞吐量可达 **7-8 GB/s**。

## 🌟 核心特性

### 🚀 核心功能
- ✅ **多出口IP管理** - 支持多个公网IP，自动SNAT源地址转换
- ✅ **智能IP分配** - 轮询、按目标、按端口、负载均衡等多种策略
- ✅ **高性能加密** - TLS 1.3 + AES-GCM硬件加速，性能提升 **5.8倍**
- ✅ **UDP代理** - 完整支持SOCKS5 UDP ASSOCIATE
- ✅ **Trojan协议** - 兼容标准Trojan协议，流量更隐蔽

### 🛡️ 高级功能
- ✅ **IP健康检查** - 自动检测故障IP并切换，无需重启服务
- ✅ **IP自动检测** - 自动识别服务器上的公网出口IP，无需手动配置
- ✅ **智能IP切换** - 故障IP自动过滤，恢复后自动重新启用
- ✅ **实时监控** - 连接数、流量、延迟实时统计
- ✅ **多用户认证** - 用户管理和权限控制（Argon2密码哈希）
- ✅ **负载均衡** - 按连接数和流量智能分配
- ✅ **DDoS防护** - 连接速率限制和自动阻止
- ✅ **IP过滤** - 黑白名单支持
- ✅ **配置热更新** - 无需重启服务更新配置
- ✅ **零拷贝优化** - Linux splice支持，性能提升30-40%

### 🐳 部署和运维
- ✅ **Docker支持** - 完整的容器化部署方案
- ✅ **Web管理界面** - 可视化管理和监控
- ✅ **订阅功能** - 支持订阅链接自动配置客户端
- ✅ **自动化部署** - 一键部署脚本

## 📊 性能指标

### 优化效果
- **加密吞吐量**: 7-8 GB/s (32KB数据包)
- **性能提升**: 5.8倍 (相比优化前)
- **CPU使用率**: 降低70%
- **内存优化**: Buffer池化，减少GC压力

### 基准测试结果
```
BenchmarkEncryptionThroughput/Size_32KB
  614517 次/秒
  4186 ns/op
  7827.69 MB/s
  16 B/op, 1 allocs/op
```

详细性能测试结果请查看 [PERFORMANCE_TEST_RESULTS.md](PERFORMANCE_TEST_RESULTS.md)

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

### 数据流
```
应用数据 → 自定义协议 → AEAD加密 → TLS 1.3 → TCP
```

## 🚀 快速开始

### 方式1: Docker部署（推荐）

```bash
# 1. 克隆仓库
git clone <repository-url>
cd multiexit-proxy

# 2. 配置
cp configs/server.yaml.example configs/server.yaml
# 编辑配置文件，设置IP和密钥

# 3. 启动
docker-compose up -d

# 4. 查看日志
docker-compose logs -f
```

### 方式2: 直接部署

```bash
# 1. 编译
go build -o server ./cmd/server
go build -o client ./cmd/client

# 2. 配置服务端
cp configs/server.yaml.example configs/server.yaml
# 编辑配置文件

# 3. 运行服务端（需要root权限）
sudo ./server -config configs/server.yaml

# 4. 配置客户端
cp configs/client.json.example configs/client.json
# 编辑配置文件

# 5. 运行客户端
./client -config configs/client.json
```

### 方式3: 自动化部署

```bash
# 1. 编译并打包
./deploy-server.sh

# 2. 上传到服务器
scp -r deploy/server/* root@YOUR_SERVER:/opt/multiexit-proxy/

# 3. 在服务器上运行自动化部署
ssh root@YOUR_SERVER
cd /opt/multiexit-proxy
chmod +x scripts/setup.sh
sudo bash scripts/setup.sh

# 4. 启动服务
sudo systemctl start multiexit-proxy
```

### 客户端使用

#### 使用订阅链接（推荐）
```bash
./client -subscribe "http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN"
```

#### 使用配置文件
```bash
./client -config configs/client.json
```

#### 配置系统代理
```bash
# macOS/Linux
export ALL_PROXY=socks5://127.0.0.1:1080

# Windows
# 设置 → 网络和Internet → 代理 → 手动代理设置
# SOCKS代理: 127.0.0.1:1080
```

#### 测试代理
```bash
curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/ip
```

### Web管理界面

访问: `http://YOUR_SERVER:8080`

- 默认用户名: `admin`
- 密码: 配置文件中的 `web.password`

功能包括:
- 系统状态监控
- IP管理和健康检查
- 实时统计信息
- 用户管理
- 配置更新

## 📋 IP分配策略

系统支持多种IP分配策略，并且**自动过滤不健康的IP**：

1. **轮询策略 (Round Robin)** - 按连接顺序轮流使用各个健康IP
2. **按目标地址 (Destination Based)** - 相同目标使用相同出口IP（仅健康IP）
3. **按端口分配 (Port Based)** - 特定端口范围映射到特定IP
4. **负载均衡 (Load Balanced)** - 按连接数或流量智能分配（仅健康IP）

配置示例：
```yaml
strategy:
  type: "load_balanced"
  config:
    method: "connections"  # 或 "traffic"

# 健康检查配置（推荐启用）
health_check:
  enabled: true      # 启用健康检查
  interval: "30s"    # 检查间隔
  timeout: "5s"      # 检查超时

# IP自动检测配置（可选）
ip_detection:
  enabled: true      # 启用IP自动检测
  interface: ""      # 指定网络接口（空=检测所有）
```

## 🔐 安全特性

### 加密和认证
- 🔐 **端到端加密** - TLS 1.3 + AES-GCM/ChaCha20-Poly1305
- 🔑 **预共享密钥** - PSK认证机制
- 🔐 **多用户认证** - Argon2密码哈希，支持速率限制和IP白名单

### 流量混淆
- 🎭 **SNI伪装** - TLS握手时使用常见域名SNI
- 📦 **包大小混淆** - 随机padding，调整包大小分布
- ⏱️ **时间混淆** - 随机延迟，模拟正常请求模式
- 🔌 **端口复用** - 服务端监听443端口（标准HTTPS端口）

### 防护机制
- 🛡️ **DDoS防护** - 连接速率限制和自动阻止
- 🚫 **IP过滤** - 黑白名单支持
- 🔒 **防重放攻击** - 时间戳 + 随机数验证

## 🔧 技术栈

- **语言**: Go 1.21+
- **加密**: TLS 1.3, AES-GCM (硬件加速), ChaCha20-Poly1305
- **网络**: iptables + iproute2 (SNAT), Linux splice (零拷贝)
- **协议**: 自定义协议 + SOCKS5, Trojan协议支持
- **Web框架**: Gorilla Mux
- **数据库**: 无（配置驱动）

## 📦 项目结构

```
multiexit-proxy/
├── cmd/                    # 可执行程序入口
│   ├── client/            # 客户端入口
│   ├── server/            # 服务端入口
│   ├── trojan-client/     # Trojan客户端
│   └── trojan-server/     # Trojan服务端
├── internal/              # 内部包（不对外暴露）
│   ├── auth/              # 用户认证
│   ├── config/            # 配置管理（支持热更新）
│   ├── monitor/           # 监控统计
│   ├── protocol/          # 协议实现（加密/解密）
│   ├── proxy/             # 代理核心（零拷贝优化）
│   ├── security/          # 安全功能（DDoS防护、IP过滤）
│   ├── snat/              # SNAT管理（IP选择、路由、健康检查）
│   ├── transport/         # 传输层（TLS、流量混淆）
│   ├── trojan/            # Trojan协议实现
│   └── web/               # Web管理界面
├── pkg/                   # 公共包（可被外部使用）
│   ├── socks5/            # SOCKS5协议（支持UDP）
│   └── subscribe/         # 订阅功能
├── configs/               # 配置文件示例
├── scripts/               # 部署脚本
├── tests/                 # 测试
│   ├── integration/       # 集成测试
│   └── performance/       # 性能测试
└── docs/                  # 文档
```

## 📖 文档

### 用户文档
- 📘 [用户指南](docs/USER_GUIDE.md) - 完整使用教程，包括配置、功能使用、故障排查
- 📗 [API文档](docs/API.md) - Web API接口文档，所有端点说明
- 📙 [文档索引](docs/INDEX.md) - 完整文档导航

### 开发者文档
- 🔧 [开发者指南](docs/DEVELOPER_GUIDE.md) - 代码结构、开发环境、添加新功能
- 🧪 [测试指南](docs/TESTING_GUIDE.md) - 测试类型、工具、报告格式

### 参考文档
- 🏗️ [设计文档](DESIGN.md) - 完整架构设计
- ⚙️ [技术规范](TECHNICAL_SPEC.md) - 技术实现细节
- 📊 [性能测试结果](PERFORMANCE_TEST_RESULTS.md) - 性能基准测试
- 📝 [更新日志](CHANGELOG.md) - 版本更新记录
- ✅ [测试报告](TEST_REPORT.md) - 测试执行报告
- 🔐 [Trojan协议](TROJAN.md) - Trojan协议使用说明

## ⚙️ 配置说明

### 服务端配置示例

```yaml
server:
  listen: "0.0.0.0:8443"

auth:
  key: "your-secret-key"

# 出口IP列表（可选，如果启用自动检测则会被合并）
exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"

strategy:
  type: "round_robin"

# IP自动检测（推荐启用）
ip_detection:
  enabled: true      # 启用IP自动检测
  interface: ""      # 指定网络接口，为空则检测所有接口

# 健康检查（推荐启用）
health_check:
  enabled: true      # 启用健康检查
  interval: "30s"    # 检查间隔
  timeout: "5s"      # 检查超时

snat:
  enabled: true
  gateway: "1.2.3.1"
  interface: "eth0"

web:
  enabled: true
  listen: "0.0.0.0:8080"
  username: "admin"
  password: "your-password"

security:
  ddos:
    enabled: true
    max_connections_per_ip: 10
    connection_rate_limit: 5
```

完整配置示例请查看 `configs/server.yaml.example`

### 客户端配置示例

```json
{
  "server_addr": "your-server.com:8443",
  "sni": "cloudflare.com",
  "auth_key": "your-secret-key",
  "local_addr": "127.0.0.1:1080"
}
```

完整配置示例请查看 `configs/client.json.example`

## ⚠️ 系统要求

### 服务端
- Linux系统（SNAT功能需要）
- root权限或CAP_NET_ADMIN权限
- 多个公网IP（EIP）
- iptables和iproute2工具
- Go 1.21+（如从源码编译）

### 客户端
- Windows/macOS/Linux
- 无需特殊权限
- Go 1.21+（如从源码编译）

## 🔄 协议支持

### 自定义协议（默认）
- TLS 1.3外层加密
- 自定义二进制协议
- AEAD加密（AES-GCM/ChaCha20-Poly1305）

### Trojan协议
- 标准Trojan协议兼容
- SHA224密码哈希
- TLS传输层

使用Trojan协议：
```bash
# 服务端
./trojan-server -config configs/server-trojan.yaml

# 客户端
./trojan-client -config configs/client-trojan.json
```

## 📈 性能优化

系统经过多轮性能优化：

1. **AES-GCM硬件加速** - 利用CPU的AES-NI指令集
2. **Buffer池化** - 使用sync.Pool减少内存分配
3. **Nonce计数器** - 避免随机数生成开销
4. **连接级加密上下文** - 减少重复初始化
5. **Linux splice零拷贝** - 内核级数据转发
6. **strconv优化** - 使用strconv替代fmt进行字符串转换

详细优化说明请查看 [PERFORMANCE_TEST_RESULTS.md](PERFORMANCE_TEST_RESULTS.md)

## 🐛 故障排查

常见问题请查看 [用户指南 - 故障排查](docs/USER_GUIDE.md#故障排查)

### 快速检查
```bash
# 检查服务是否运行
ps aux | grep server

# 检查端口监听
netstat -tlnp | grep 8443

# 查看日志
tail -f server.log

# 测试IP连通性
ping YOUR_EXIT_IP
```

## 📄 许可证

本项目仅供学习和研究使用。

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 贡献指南
1. Fork本项目
2. 创建功能分支: `git checkout -b feature/new-feature`
3. 提交更改: `git commit -am 'Add new feature'`
4. 推送分支: `git push origin feature/new-feature`
5. 提交Pull Request

## 📝 更新日志

### v2.0.0 - 2025年
- ✅ UDP代理支持
- ✅ **IP健康检查** - 自动检测故障IP并切换
- ✅ **IP自动检测** - 自动识别服务器上的公网出口IP
- ✅ **智能IP切换** - 故障IP自动过滤，恢复后自动重新启用
- ✅ 实时监控统计
- ✅ 多用户认证
- ✅ 负载均衡策略
- ✅ 配置热更新
- ✅ Docker支持
- ✅ DDoS防护
- ✅ IP过滤
- ✅ 零拷贝优化
- ✅ 性能提升5.8倍

详细更新日志请查看 [CHANGELOG.md](CHANGELOG.md)

## ⭐ 项目亮点

1. **多出口IP SNAT** - 这是Hiddify、Xray等工具不支持的核心功能
2. **智能IP管理** - 自动检测IP、健康检查、故障自动切换，无需手动干预
3. **高性能** - 优化后达到7-8 GB/s吞吐量，接近Xray/SingBox的60-70%性能
4. **功能完整** - 从基础代理到高级功能（认证、监控、防护）一应俱全
5. **易于部署** - Docker一键部署，自动化脚本支持，IP自动检测
6. **文档完善** - 用户指南、API文档、开发者文档齐全
7. **生产就绪** - 经过完整测试，包括集成测试和性能测试
8. **自动容错** - IP故障自动切换，恢复后自动重新启用，高可用保障

## 🔗 相关链接

- [用户指南](docs/USER_GUIDE.md)
- [API文档](docs/API.md)
- [性能测试报告](PERFORMANCE_TEST_RESULTS.md)
- [设计文档](DESIGN.md)

---

**注意**: 请遵守当地法律法规使用本软件。本软件仅供学习和研究使用。

