# MultiExit Proxy

一个高性能、功能丰富的多出口 IP 代理系统，支持 SNAT、智能调度、流量分析、Web 管理面板等企业级特性。

## 📋 目录

- [项目简介](#项目简介)
- [核心特性](#核心特性)
- [系统架构](#系统架构)
- [快速开始](#快速开始)
- [详细配置](#详细配置)
- [使用指南](#使用指南)
- [API 文档](#api-文档)
- [Web 管理面板](#web-管理面板)
- [高级功能](#高级功能)
- [部署方案](#部署方案)
- [性能优化](#性能优化)
- [安全建议](#安全建议)
- [故障排查](#故障排查)
- [开发指南](#开发指南)
- [贡献指南](#贡献指南)

---

## 项目简介

MultiExit Proxy 是一个企业级的多出口 IP 代理系统，专为需要多公网 IP 出口的场景设计。系统支持多种智能调度策略、实时监控、流量分析、规则引擎等高级功能，并提供现代化的 Web 管理界面。

### 适用场景

- **多 IP 出口负载均衡**：将流量智能分配到多个公网 IP
- **地理位置优化**：根据目标地理位置选择最优出口 IP
- **规则路由**：基于域名、IP、端口等条件进行精确路由
- **流量分析与监控**：实时监控连接、流量、异常情况
- **高可用部署**：支持集群模式，实现高可用和负载均衡

---

## 核心特性

### 🚀 多出口 IP 管理

- **自动 IP 检测**：自动检测服务器上的所有公网 IP
- **手动配置**：支持手动配置出口 IP 列表
- **IP 健康检查**：自动检测 IP 可用性，故障自动剔除
- **IP 状态监控**：实时监控每个 IP 的连接数、流量、延迟等指标

### 🔄 智能调度策略

系统支持多种 IP 选择策略：

1. **轮询调度 (round_robin)**
   - 按顺序轮流使用各个出口 IP
   - 适用于流量均匀分配的场景

2. **基于端口 (port_based)**
   - 根据目标端口范围选择特定 IP
   - 支持端口段配置，如 `0-32767` 使用 IP1，`32768-65535` 使用 IP2

3. **基于目标地址 (destination_based)**
   - 根据目标 IP 或域名选择出口 IP
   - 支持精确匹配和通配符匹配

4. **负载均衡 (load_balanced)**
   - 根据连接数或流量自动选择负载最低的 IP
   - 支持按连接数 (`connections`) 或流量 (`traffic`) 两种模式

5. **地理位置优化 (geo_location)**
   - 根据目标地理位置选择延迟最低的出口 IP
   - 支持延迟优化模式

6. **规则引擎 (rule_engine)**
   - 基于域名、IP、端口的复杂规则匹配
   - 支持优先级、启用/禁用等高级功能

### 🔐 协议支持

- **SOCKS5 协议**：完整的 SOCKS5 支持，包括 TCP 和 UDP
- **Trojan 协议**：兼容 Trojan 协议，支持密码认证
- **TLS 加密**：所有连接均使用 TLS 加密传输
- **SNI 伪装**：支持 SNI 伪装，增强隐蔽性

### 🛡️ 安全特性

- **预共享密钥 (PSK) 认证**：使用 AES-GCM 加密的密钥认证
- **CSRF 防护**：Web 管理界面具备完整的 CSRF 保护
- **登录保护**：支持登录失败次数限制和 IP 封禁
- **IP 黑白名单**：支持基于 IP 的访问控制
- **速率限制**：支持 IP 级别、用户级别和全局级别的速率限制
- **DDoS 防护**：内置 DDoS 防护机制

### 📊 监控与统计

- **实时连接统计**：监控每个 IP 的活跃连接数、总连接数
- **流量统计**：统计上行、下行流量，支持按 IP、域名、时间维度查询
- **延迟监控**：实时监控每个出口 IP 的延迟
- **异常检测**：自动检测流量异常、连接异常等
- **Prometheus 集成**：提供 Prometheus 格式的监控指标

### 🗄️ 数据持久化

- **PostgreSQL 支持**：可选启用 PostgreSQL 数据库存储历史数据
- **统计表结构**：连接统计、域名统计、流量趋势、异常检测等
- **数据清理**：支持自动清理历史数据，避免数据库膨胀

### 🌐 Web 管理面板

- **现代化 UI**：基于 Next.js 和 shadcn/ui 构建的现代化界面
- **配置管理**：可视化配置编辑，支持配置验证和版本管理
- **IP 管理**：查看、添加、删除出口 IP，查看 IP 状态
- **规则管理**：可视化规则编辑，支持优先级调整
- **流量统计**：实时流量图表，支持多维度查询
- **版本回滚**：配置版本管理，支持一键回滚

### 🔧 高级功能

- **配置热重载**：支持配置文件变更后自动重载（无需重启）
- **配置版本管理**：每次配置变更自动创建版本，支持回滚
- **订阅功能**：支持订阅链接，客户端可通过订阅自动更新配置
- **集群模式**：支持多节点集群部署，实现高可用
- **连接池**：客户端支持连接池，提升性能
- **自动重连**：客户端支持自动重连，断线自动恢复

---

## 系统架构

### 架构图

```
┌─────────────┐
│   Client    │ (SOCKS5/Trojan)
└──────┬──────┘
       │ TLS
       ▼
┌─────────────────────────────────────┐
│         MultiExit Proxy Server      │
│  ┌────────────────────────────────┐  │
│  │   Protocol Handler (SOCKS5)   │  │
│  └──────────────┬────────────────┘  │
│                 │                    │
│  ┌──────────────▼────────────────┐  │
│  │      Rule Engine              │  │
│  └──────────────┬────────────────┘  │
│                 │                    │
│  ┌──────────────▼────────────────┐  │
│  │    IP Selector (Strategy)     │  │
│  └──────────────┬────────────────┘  │
│                 │                    │
│  ┌──────────────▼────────────────┐  │
│  │    SNAT Manager               │  │
│  └──────────────┬────────────────┘  │
│                 │                    │
│  ┌──────────────▼────────────────┐  │
│  │  Health Checker               │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
       │
       ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Exit IP 1  │  │  Exit IP 2  │  │  Exit IP N  │
└─────────────┘  └─────────────┘  └─────────────┘
```

### 核心模块

- **proxy**：代理核心模块，处理连接转发
- **snat**：SNAT 管理，IP 选择策略，健康检查
- **protocol**：协议处理，加密解密
- **transport**：传输层，TLS 处理
- **monitor**：监控统计，流量分析
- **web**：Web API 服务器
- **config**：配置管理，版本控制
- **database**：数据库操作
- **security**：安全功能（DDoS、IP 过滤等）

---

## 快速开始

### 环境要求

#### 服务端

- **Go 1.21+**：用于编译和运行服务端
- **PostgreSQL 15+**（可选）：用于存储统计数据
- **Linux 系统**：推荐使用 Linux（支持 SNAT 功能）
- **root 权限**：启用 SNAT 功能需要 root 权限

#### 客户端

- **Go 1.21+**：用于编译客户端
- **任意操作系统**：支持 Linux、macOS、Windows

#### 前端

- **Node.js 20+**：用于运行前端开发服务器
- **npm 或 pnpm**：包管理器

### 源码编译

#### 1. 克隆仓库

```bash
git clone <repository-url>
cd multiexit-proxy
```

#### 2. 编译服务端

```bash
# 编译服务端
go build -o server cmd/server/main.go

# 编译客户端
go build -o client cmd/client/main.go

# 编译 Trojan 服务端（可选）
go build -o trojan-server cmd/trojan-server/main.go

# 编译 Trojan 客户端（可选）
go build -o trojan-client cmd/trojan-client/main.go
```

#### 3. 准备配置文件

```bash
# 复制服务端配置示例
cp configs/server.yaml.example configs/server.yaml

# 复制客户端配置示例
cp configs/client.json.example configs/client.json

# 编辑配置文件
vim configs/server.yaml
vim configs/client.json
```

#### 4. 启动服务端

```bash
./server -config configs/server.yaml
```

#### 5. 启动客户端

```bash
./client -config configs/client.json
```

### Docker 部署

#### 1. 使用 Docker Compose（推荐）

```bash
# 启动所有服务（包括 PostgreSQL）
docker-compose up -d

# 查看日志
docker-compose logs -f multiexit-proxy

# 停止服务
docker-compose down
```

#### 2. 单独构建 Docker 镜像

```bash
# 构建镜像
docker build -t multiexit-proxy:latest .

# 运行容器
docker run -d \
  --name multiexit-proxy \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add SYS_ADMIN \
  -v $(pwd)/configs:/etc/multiexit-proxy:ro \
  -v $(pwd)/logs:/var/log/multiexit-proxy \
  multiexit-proxy:latest \
  ./server -config /etc/multiexit-proxy/server.yaml
```

### 前端开发环境

```bash
cd frontend-system-design

# 安装依赖
npm install
# 或
pnpm install

# 启动开发服务器
npm run dev
# 或
pnpm dev

# 访问 http://localhost:8081
```

---

## 详细配置

### 服务端配置 (server.yaml)

#### 基础配置

```yaml
# 服务器监听地址
server:
  listen: ":443"  # 监听所有接口的 443 端口
  tls:
    cert: "/path/to/cert.pem"      # TLS 证书路径
    key: "/path/to/key.pem"        # TLS 私钥路径
    sni_fake: true                 # 启用 SNI 伪装
    fake_snis:                     # 伪装的 SNI 列表
      - "cloudflare.com"
      - "google.com"
      - "github.com"

# 认证配置
auth:
  method: "psk"                    # 认证方法：psk
  key: "your-secret-key-change-this"  # 预共享密钥（必须修改）

# 出口 IP 列表
exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"
  - "9.10.11.12"
```

#### 调度策略配置

```yaml
# 调度策略
strategy:
  type: "round_robin"  # 可选值：
                       # - round_robin: 轮询
                       # - port_based: 基于端口
                       # - destination_based: 基于目标地址
                       # - load_balanced: 负载均衡
                       # - geo_location: 地理位置优化
                       # - rule_engine: 规则引擎

  # 如果 type 是 port_based，需要配置端口范围
  port_ranges:
    - range: "0-32767"
      ip: "1.2.3.4"
    - range: "32768-65535"
      ip: "5.6.7.8"
```

#### SNAT 配置

```yaml
# SNAT 配置（需要 root 权限）
snat:
  enabled: true
  gateway: "192.168.1.1"    # 网关地址
  interface: "eth0"         # 网络接口名称
```

#### 健康检查配置

```yaml
# 健康检查
health_check:
  enabled: true              # 启用健康检查
  interval: "30s"           # 检查间隔（默认 30 秒）
  timeout: "5s"             # 检查超时（默认 5 秒）
```

#### IP 自动检测配置

```yaml
# IP 自动检测
ip_detection:
  enabled: true              # 启用自动检测
  interface: ""              # 指定网络接口（如 "eth0"），为空则检测所有接口
```

#### 连接管理配置

```yaml
# 连接管理
connection:
  read_timeout: "30s"        # 读取超时
  write_timeout: "30s"       # 写入超时
  idle_timeout: "300s"       # 空闲超时（5 分钟）
  dial_timeout: "10s"        # 连接超时
  max_connections: 1000      # 最大并发连接数（0 = 无限制）
  keep_alive: true           # 启用 TCP KeepAlive
  keep_alive_time: "30s"     # KeepAlive 间隔
```

#### 地理位置配置

```yaml
# 地理位置选择
geo_location:
  enabled: true              # 启用地理位置选择
  api_url: ""                # 地理位置 API URL（可选，默认使用 ip-api.com）
  latency_optimize: true     # 启用延迟优化
```

#### 规则引擎配置

```yaml
# 规则引擎
rules:
  - name: "rule-1"           # 规则名称
    priority: 100            # 优先级（数字越大优先级越高）
    match_domain:            # 匹配域名（支持通配符）
      - "*.google.com"
      - "github.com"
    match_ip:                # 匹配 IP（CIDR 格式）
      - "8.8.8.0/24"
    match_port:              # 匹配端口
      - 80
      - 443
    target_ip: "1.2.3.4"     # 目标出口 IP
    action: "route"          # 动作：route（路由）或 block（阻止）
    enabled: true            # 是否启用
```

#### 监控配置

```yaml
# 监控统计
monitor:
  enabled: true              # 启用监控统计

# 流量分析
traffic_analysis:
  enabled: true              # 启用流量分析
  trend_window: "24h"        # 趋势窗口（默认 24 小时）
  anomaly_threshold: 2.0     # 异常阈值（默认 2.0，即 2 倍标准差）
```

#### 数据库配置

```yaml
# 数据库配置（可选）
database:
  enabled: true              # 启用数据库
  host: "localhost"          # 数据库主机
  port: 5432                 # 数据库端口
  database: "multiexit_proxy"  # 数据库名
  user: "multiexit"          # 用户名
  password: "multiexit123"   # 密码
  ssl_mode: "disable"        # SSL 模式：disable, require, verify-ca, verify-full
  max_conns: 100             # 最大连接数
  max_idle: 10               # 最大空闲连接数
```

#### Web 管理界面配置

```yaml
# Web 管理界面
web:
  enabled: true              # 启用 Web 管理界面
  listen: ":8080"           # Web API 监听地址
  username: "admin"         # 管理用户名
  password: "admin123"      # 管理密码（建议修改）
```

#### 日志配置

```yaml
# 日志配置
logging:
  level: "info"              # 日志级别：debug, info, warn, error
  file: "/var/log/multiexit-proxy.log"  # 日志文件路径（可选，为空则输出到标准输出）
```

#### Trojan 协议配置

```yaml
# Trojan 协议支持（可选）
trojan:
  enabled: false            # 启用 Trojan 协议
  password: ""              # Trojan 密码
```

#### 集群配置

```yaml
# 集群配置（可选）
cluster:
  enabled: false            # 启用集群模式
  node_id: "node-1"         # 节点 ID
  nodes:                    # 其他节点地址列表
    - "node2.example.com:8080"
    - "node3.example.com:8080"
  load_balancer: "round_robin"  # 负载均衡策略：round_robin 或 least_connections
  health_interval: "30s"    # 健康检查间隔
```

### 客户端配置 (client.json)

```json
{
  "server": {
    "address": "your-server.com:443",
    "sni": "cloudflare.com"
  },
  "auth": {
    "key": "your-secret-key-change-this"
  },
  "local": {
    "socks5": "127.0.0.1:1080",
    "http": "127.0.0.1:8080"
  },
  "logging": {
    "level": "info"
  },
  "reconnect": {
    "max_retries": 0,
    "initial_delay": "1s",
    "max_delay": "5m",
    "backoff_factor": 2.0,
    "jitter": true
  }
}
```

#### 客户端配置说明

- **server.address**：服务端地址和端口
- **server.sni**：TLS SNI 值
- **auth.key**：与服务端相同的预共享密钥
- **local.socks5**：本地 SOCKS5 代理监听地址
- **local.http**：本地 HTTP 代理监听地址（可选）
- **logging.level**：日志级别
- **reconnect**：重连配置
  - **max_retries**：最大重试次数（0 = 无限重试）
  - **initial_delay**：初始延迟
  - **max_delay**：最大延迟
  - **backoff_factor**：退避因子
  - **jitter**：是否启用抖动

---

## 使用指南

### 服务端启动

```bash
# 使用默认配置文件
./server -config configs/server.yaml

# 指定配置文件路径
./server -config /path/to/server.yaml

# 查看帮助
./server -h
```

### 客户端启动

```bash
# 使用配置文件启动
./client -config configs/client.json

# 使用订阅链接启动
./client -subscribe "https://your-server.com/api/subscribe?token=your-token"

# 订阅链接会自动下载配置并保存到配置文件
```

### 测试连接

#### 使用 curl 测试 SOCKS5 代理

```bash
# 通过 SOCKS5 代理访问
curl --socks5 127.0.0.1:1080 https://www.google.com

# 查看出口 IP
curl --socks5 127.0.0.1:1080 https://api.ipify.org
```

#### 使用浏览器测试

1. 配置浏览器 SOCKS5 代理：`127.0.0.1:1080`
2. 访问 https://www.whatismyip.com 查看出口 IP
3. 多次刷新页面，应该能看到不同的出口 IP（如果使用轮询策略）

### 配置热重载

修改配置文件后，系统会自动检测并重载配置（如果启用了热重载功能）。无需重启服务。

### 查看日志

```bash
# 查看服务端日志
tail -f /var/log/multiexit-proxy.log

# 或查看标准输出
./server -config configs/server.yaml
```

---

## API 文档

### 认证

所有需要认证的 API 都需要先登录获取 Session。

#### 登录

```http
POST /api/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

响应：

```json
{
  "success": true,
  "message": "Login successful"
}
```

#### 登出

```http
POST /api/logout
```

### 配置管理

#### 获取配置

```http
GET /api/config
```

响应：

```json
{
  "server": { ... },
  "auth": { ... },
  "exit_ips": [ ... ],
  ...
}
```

#### 更新配置

```http
POST /api/config
Content-Type: application/json

{
  "server": { ... },
  "auth": { ... },
  ...
}
```

响应：

```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "version": 2
}
```

### IP 管理

#### 获取 IP 列表

```http
GET /api/ips
```

响应：

```json
{
  "configured": ["1.2.3.4", "5.6.7.8"],
  "detected": ["1.2.3.4", "5.6.7.8", "9.10.11.12"],
  "stats": {
    "1.2.3.4": {
      "connections": 10,
      "traffic": 1024000,
      "latency": 50
    }
  }
}
```

#### 添加 IP

```http
POST /api/ips
Content-Type: application/json

{
  "ip": "9.10.11.12"
}
```

#### 删除 IP

```http
DELETE /api/ips/{ip}
```

### 规则管理

#### 获取规则列表

```http
GET /api/rules
```

#### 添加规则

```http
POST /api/rules
Content-Type: application/json

{
  "name": "rule-1",
  "priority": 100,
  "match_domain": ["*.google.com"],
  "target_ip": "1.2.3.4",
  "action": "route",
  "enabled": true
}
```

#### 更新规则

```http
PUT /api/rules/{id}
Content-Type: application/json

{
  "name": "rule-1",
  "priority": 200,
  ...
}
```

#### 删除规则

```http
DELETE /api/rules/{id}
```

### 统计信息

#### 获取状态

```http
GET /api/status
```

响应：

```json
{
  "uptime": 3600,
  "total_connections": 1000,
  "active_connections": 50,
  "total_traffic": 1073741824,
  "ips": {
    "1.2.3.4": {
      "connections": 25,
      "traffic": 536870912,
      "latency": 50,
      "healthy": true
    }
  }
}
```

#### 获取流量统计

```http
GET /api/traffic?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z
```

#### 获取连接统计

```http
GET /api/stats?ip=1.2.3.4
```

### 版本管理

#### 获取版本列表

```http
GET /api/versions
```

#### 回滚到指定版本

```http
POST /api/rollback
Content-Type: application/json

{
  "version": 2
}
```

### 订阅 API

#### 获取订阅

```http
GET /api/subscribe?token=your-token
```

响应：Base64 编码的订阅配置

#### 生成订阅链接

```http
GET /api/subscription/link?token=your-token
```

### Prometheus 指标

```http
GET /metrics
```

---

## Web 管理面板

### 功能概览

Web 管理面板提供以下功能：

1. **仪表盘**：系统概览、实时统计
2. **配置管理**：可视化配置编辑、验证、保存
3. **IP 管理**：查看、添加、删除出口 IP
4. **规则管理**：规则列表、添加、编辑、删除
5. **流量统计**：实时流量图表、历史数据查询
6. **版本管理**：配置版本列表、版本对比、回滚
7. **订阅管理**：生成订阅链接、查看订阅信息

### 访问方式

1. 启动前端开发服务器：
   ```bash
   cd frontend-system-design
   npm run dev
   ```

2. 访问 http://localhost:8081

3. 使用配置文件中设置的用户名和密码登录

### 界面说明

#### 登录页面

- 输入用户名和密码
- 支持记住登录状态（使用浏览器本地存储）

#### 仪表盘

- **系统状态**：运行时间、总连接数、活跃连接数
- **流量统计**：实时流量图表（上行/下行）
- **IP 状态**：各出口 IP 的连接数、流量、延迟、健康状态
- **异常告警**：最近的异常检测结果

#### 配置管理

- **配置编辑器**：YAML 格式的配置编辑器
- **配置验证**：保存前自动验证配置格式
- **配置保存**：保存后自动创建新版本
- **配置历史**：查看历史配置版本

#### IP 管理

- **IP 列表**：显示配置的 IP 和自动检测的 IP
- **IP 状态**：每个 IP 的连接数、流量、延迟
- **添加 IP**：手动添加新的出口 IP
- **删除 IP**：删除配置中的 IP（仅限配置的 IP）

#### 规则管理

- **规则列表**：显示所有规则，按优先级排序
- **规则编辑**：添加、编辑、删除规则
- **规则测试**：测试规则匹配效果
- **规则启用/禁用**：快速启用或禁用规则

#### 流量统计

- **实时图表**：实时流量趋势图
- **历史查询**：按时间范围查询历史流量
- **IP 维度**：按 IP 查看流量统计
- **域名维度**：按域名查看流量统计

#### 版本管理

- **版本列表**：显示所有配置版本
- **版本对比**：对比不同版本的差异
- **版本回滚**：一键回滚到指定版本

---

## 高级功能

### 配置热重载

系统支持配置文件变更后自动重载，无需重启服务。

**启用方式**：在配置文件中启用热重载功能（如果支持）

**工作原理**：
1. 系统监控配置文件变更
2. 检测到变更后，验证新配置
3. 验证通过后，应用新配置
4. 创建配置版本备份

### 配置版本管理

每次配置变更都会自动创建新版本，支持版本回滚。

**版本存储**：版本存储在配置目录下的 `.versions` 目录

**回滚方式**：
1. 通过 Web 管理面板选择版本并回滚
2. 通过 API 调用回滚接口
3. 手动复制版本文件

### 订阅功能

客户端可以通过订阅链接自动获取和更新配置。

**生成订阅链接**：
```bash
# 通过 API 生成
curl "http://localhost:8080/api/subscription/link?token=your-token"
```

**使用订阅**：
```bash
./client -subscribe "https://your-server.com/api/subscribe?token=your-token"
```

### 集群模式

支持多节点集群部署，实现高可用和负载均衡。

**配置示例**：
```yaml
cluster:
  enabled: true
  node_id: "node-1"
  nodes:
    - "node2.example.com:8080"
    - "node3.example.com:8080"
  load_balancer: "least_connections"
  health_interval: "30s"
```

**集群特性**：
- 节点间健康检查
- 负载均衡
- 故障自动切换
- 配置同步

### 流量分析

系统自动分析流量趋势，检测异常情况。

**异常类型**：
- **流量突增**：流量突然大幅增加
- **连接异常**：连接数异常波动
- **延迟异常**：延迟突然增加

**配置示例**：
```yaml
traffic_analysis:
  enabled: true
  trend_window: "24h"        # 趋势窗口
  anomaly_threshold: 2.0      # 异常阈值（2 倍标准差）
```

### 规则引擎

支持基于域名、IP、端口的复杂路由规则。

**规则示例**：
```yaml
rules:
  - name: "google-routing"
    priority: 100
    match_domain:
      - "*.google.com"
      - "*.googleapis.com"
    target_ip: "1.2.3.4"
    action: "route"
    enabled: true
  
  - name: "github-routing"
    priority: 90
    match_domain:
      - "github.com"
      - "*.github.io"
    target_ip: "5.6.7.8"
    action: "route"
    enabled: true
  
  - name: "block-malicious"
    priority: 200
    match_domain:
      - "*.malicious.com"
    action: "block"
    enabled: true
```

---

## 部署方案

### 单机部署

适用于小规模使用场景。

**步骤**：
1. 编译二进制文件
2. 准备配置文件
3. 启动服务
4. 配置系统服务（systemd）

**systemd 服务示例**：

```ini
[Unit]
Description=MultiExit Proxy Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/multiexit-proxy
ExecStart=/opt/multiexit-proxy/server -config /opt/multiexit-proxy/configs/server.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker 部署

适用于容器化部署场景。

**使用 Docker Compose**：
```bash
docker-compose up -d
```

**单独运行容器**：
```bash
docker run -d \
  --name multiexit-proxy \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add SYS_ADMIN \
  -v $(pwd)/configs:/etc/multiexit-proxy:ro \
  multiexit-proxy:latest
```

### 集群部署

适用于大规模、高可用场景。

**架构**：
- 多个服务端节点
- 负载均衡器（可选）
- 共享配置（可选）

**配置要点**：
1. 每个节点配置相同的出口 IP 列表
2. 启用集群模式
3. 配置节点列表
4. 配置负载均衡策略

### 云函数部署

支持部署到华为云 FunctionGraph 等云函数平台。

**目录**：`functiongraph/`

**部署步骤**：
1. 编译函数代码
2. 上传到云函数平台
3. 配置环境变量
4. 配置触发器

---

## 性能优化

### 服务端优化

1. **连接池**：启用连接池，复用连接
2. **缓冲区**：调整缓冲区大小
3. **并发控制**：合理设置最大连接数
4. **KeepAlive**：启用 TCP KeepAlive
5. **零拷贝**：使用 splice 等零拷贝技术

### 客户端优化

1. **连接池**：启用连接池
2. **多路复用**：使用连接复用
3. **重连策略**：优化重连参数

### 系统优化

1. **文件描述符**：增加系统文件描述符限制
   ```bash
   ulimit -n 65535
   ```

2. **网络参数**：优化 TCP 参数
   ```bash
   # /etc/sysctl.conf
   net.core.somaxconn = 65535
   net.ipv4.tcp_max_syn_backlog = 65535
   net.ipv4.ip_local_port_range = 10000 65535
   ```

3. **CPU 亲和性**：绑定 CPU 核心（可选）

---

## 安全建议

### 认证安全

1. **修改默认密钥**：必须修改配置文件中的默认密钥
2. **使用强密钥**：密钥长度至少 32 字符，使用随机字符
3. **定期轮换密钥**：定期更换认证密钥

### 网络安全

1. **TLS 配置**：使用有效的 TLS 证书
2. **SNI 伪装**：启用 SNI 伪装，增强隐蔽性
3. **防火墙**：配置防火墙规则，限制访问
4. **IP 白名单**：启用 IP 白名单功能

### Web 管理界面安全

1. **修改默认密码**：必须修改默认管理密码
2. **使用 HTTPS**：生产环境使用 HTTPS 访问
3. **限制访问 IP**：配置反向代理，限制访问来源
4. **定期更新**：保持系统更新

### 系统安全

1. **最小权限**：使用最小权限运行服务
2. **日志审计**：启用日志记录，定期审计
3. **监控告警**：配置监控告警，及时发现异常
4. **备份配置**：定期备份配置文件

---

## 故障排查

### 常见问题

#### 1. 服务端无法启动

**问题**：启动时提示配置错误

**排查步骤**：
1. 检查配置文件格式是否正确（YAML 语法）
2. 检查必需配置项是否填写
3. 查看日志文件获取详细错误信息

**解决方案**：
```bash
# 验证配置文件
./server -config configs/server.yaml -validate

# 查看详细日志
./server -config configs/server.yaml -log-level debug
```

#### 2. 客户端无法连接

**问题**：客户端连接服务端失败

**排查步骤**：
1. 检查服务端是否正常运行
2. 检查网络连接是否正常
3. 检查认证密钥是否匹配
4. 检查防火墙规则

**解决方案**：
```bash
# 测试网络连接
telnet your-server.com 443

# 检查服务端日志
tail -f /var/log/multiexit-proxy.log

# 验证配置
diff configs/server.yaml configs/client.json
```

#### 3. SNAT 不生效

**问题**：流量没有通过指定的出口 IP

**排查步骤**：
1. 检查是否有 root 权限
2. 检查 SNAT 配置是否正确
3. 检查网络接口是否存在
4. 检查路由表

**解决方案**：
```bash
# 检查网络接口
ip addr show

# 检查路由表
ip route show

# 检查 iptables 规则
iptables -t nat -L -n -v
```

#### 4. 健康检查失败

**问题**：IP 健康检查一直失败

**排查步骤**：
1. 检查 IP 是否可达
2. 检查健康检查配置
3. 检查网络连接

**解决方案**：
```yaml
# 调整健康检查参数
health_check:
  enabled: true
  interval: "60s"    # 增加检查间隔
  timeout: "10s"     # 增加超时时间
```

#### 5. 数据库连接失败

**问题**：无法连接 PostgreSQL 数据库

**排查步骤**：
1. 检查数据库服务是否运行
2. 检查数据库配置是否正确
3. 检查网络连接
4. 检查数据库用户权限

**解决方案**：
```bash
# 测试数据库连接
psql -h localhost -U multiexit -d multiexit_proxy

# 检查数据库服务
systemctl status postgresql

# 查看数据库日志
tail -f /var/log/postgresql/postgresql-*.log
```

### 日志分析

#### 查看错误日志

```bash
# 查看所有错误
grep -i error /var/log/multiexit-proxy.log

# 查看最近的错误
tail -n 100 /var/log/multiexit-proxy.log | grep -i error
```

#### 查看连接日志

```bash
# 查看连接相关日志
grep -i connection /var/log/multiexit-proxy.log
```

#### 查看性能日志

```bash
# 查看延迟日志
grep -i latency /var/log/multiexit-proxy.log
```

### 性能问题排查

#### 连接数过高

**问题**：连接数持续增长，不释放

**排查**：
1. 检查是否有连接泄漏
2. 检查空闲超时配置
3. 检查客户端重连逻辑

**解决**：
```yaml
connection:
  idle_timeout: "60s"      # 减少空闲超时
  max_connections: 1000    # 限制最大连接数
```

#### 流量异常

**问题**：流量突然增加或减少

**排查**：
1. 检查流量分析报告
2. 检查异常检测日志
3. 检查规则配置

**解决**：
- 查看 Web 管理面板的流量统计
- 检查异常检测结果
- 调整规则配置

---

## 开发指南

### 项目结构

```
multiexit-proxy/
├── cmd/                    # 可执行文件入口
│   ├── server/            # 服务端入口
│   ├── client/            # 客户端入口
│   ├── trojan-server/     # Trojan 服务端入口
│   └── trojan-client/     # Trojan 客户端入口
├── internal/              # 内部包
│   ├── proxy/            # 代理核心
│   ├── snat/             # SNAT 管理
│   ├── protocol/         # 协议处理
│   ├── transport/        # 传输层
│   ├── monitor/          # 监控统计
│   ├── web/              # Web API
│   ├── config/           # 配置管理
│   ├── database/         # 数据库
│   └── security/         # 安全功能
├── pkg/                   # 公共包
│   ├── socks5/           # SOCKS5 实现
│   └── subscribe/        # 订阅功能
├── configs/               # 配置文件
├── frontend-system-design/  # 前端项目
├── scripts/               # 脚本
├── tests/                 # 测试文件
└── deploy/                # 部署相关
```

### 开发环境设置

```bash
# 克隆仓库
git clone <repository-url>
cd multiexit-proxy

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
go build ./cmd/server
go build ./cmd/client
```

### 代码规范

1. **Go 代码**：遵循 Go 官方代码规范
2. **命名规范**：使用有意义的变量和函数名
3. **注释**：为公共函数和类型添加注释
4. **错误处理**：正确处理和返回错误

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/proxy

# 运行基准测试
go test -bench=. ./internal/protocol
```

### 提交代码

1. 确保所有测试通过
2. 确保代码格式化（`go fmt`）
3. 确保没有 linter 错误
4. 编写清晰的提交信息

---

## 贡献指南

### 如何贡献

1. **Fork 仓库**
2. **创建功能分支**：`git checkout -b feature/your-feature`
3. **提交更改**：`git commit -am 'Add some feature'`
4. **推送分支**：`git push origin feature/your-feature`
5. **创建 Pull Request**

### 贡献类型

- **Bug 修复**：修复已知问题
- **新功能**：添加新功能
- **文档改进**：改进文档
- **性能优化**：优化性能
- **测试**：添加或改进测试

### 代码审查

所有 Pull Request 都需要经过代码审查。请确保：

1. 代码符合项目规范
2. 所有测试通过
3. 文档已更新
4. 提交信息清晰

### 问题反馈

如发现问题，请在 GitHub Issues 中提交，包括：

1. 问题描述
2. 复现步骤
3. 预期行为
4. 实际行为
5. 环境信息（操作系统、Go 版本等）
6. 日志信息（如有）

---

## 许可证

[根据实际情况填写许可证信息]

## 联系方式

- **GitHub Issues**：[项目 Issues 链接]
- **Email**：[联系邮箱]

---

## 更新日志

### v1.0.0 (2024-01-01)

- 初始版本发布
- 支持多出口 IP 管理
- 支持 SOCKS5 和 Trojan 协议
- Web 管理面板
- 监控和统计功能
- 规则引擎
- 配置版本管理

---

**注意**：本文档会持续更新，请定期查看最新版本。
