# MultiExit Proxy 使用教程

## 📖 目录

1. [快速开始](#快速开始)
2. [配置说明](#配置说明)
3. [功能使用](#功能使用)
4. [高级功能](#高级功能)
5. [故障排查](#故障排查)

---

## 快速开始

### 1. 服务端部署

#### 方式1: 直接部署

```bash
# 1. 编译
go build -o server ./cmd/server
go build -o client ./cmd/client

# 2. 配置
cp configs/server.yaml.example configs/server.yaml
# 编辑配置文件

# 3. 运行（需要root权限）
sudo ./server -config configs/server.yaml
```

#### 方式2: Docker部署（推荐）

```bash
# 1. 配置
cp configs/server.yaml.example configs/server.yaml
# 编辑配置文件

# 2. 启动
docker-compose up -d

# 3. 查看日志
docker-compose logs -f
```

---

## 配置说明

### 服务端配置 (server.yaml)

```yaml
server:
  listen_addr: "0.0.0.0:8443"
  tls:
    cert: "/path/to/cert.pem"
    key: "/path/to/key.pem"

auth:
  key: "your-secret-key"

# 出口IP列表
exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"

# IP选择策略
strategy:
  type: "round_robin"  # round_robin, destination_based, load_balanced
  config:
    # load_balanced策略配置
    method: "connections"  # connections 或 traffic

# SNAT配置
snat:
  enabled: true
  gateway: "1.2.3.1"
  interface: "eth0"

# Web管理界面
web:
  enabled: true
  listen_addr: "0.0.0.0:8080"
  username: "admin"
  password: "admin123"

# 健康检查
health_check:
  enabled: true
  interval: "30s"
  timeout: "5s"

# 监控统计
monitor:
  enabled: true

# DDoS防护
security:
  ddos:
    enabled: true
    max_connections_per_ip: 10
    connection_rate_limit: 5
    block_duration: "5m"
  ip_filter:
    enabled: false
    whitelist: []
    blacklist: []

# 用户管理（可选）
users:
  - username: "user1"
    password: "password123"
    rate_limit: 1048576  # 1MB/s
    allowed_ips: ["0.0.0.0/0"]
```

### 客户端配置 (client.json)

```json
{
  "server_addr": "your-server.com:8443",
  "sni": "cloudflare.com",
  "auth_key": "your-secret-key",
  "local_addr": "127.0.0.1:1080"
}
```

---

## 功能使用

### 1. 基本代理使用

#### 启动客户端

```bash
./client -config configs/client.json
```

#### 配置系统代理

**macOS/Linux**:
```bash
export ALL_PROXY=socks5://127.0.0.1:1080
```

**Windows**:
```
设置 → 网络和Internet → 代理 → 手动代理设置
SOCKS代理: 127.0.0.1:1080
```

#### 测试代理

```bash
# HTTP测试
curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/ip

# HTTPS测试
curl --socks5-hostname 127.0.0.1:1080 https://ifconfig.me
```

---

### 2. UDP代理使用

UDP代理已自动启用，支持SOCKS5 UDP ASSOCIATE。

**测试UDP**:
```bash
# 使用支持UDP的SOCKS5客户端
# 或使用支持UDP的工具
```

---

### 3. Web管理界面

访问: `http://YOUR_SERVER:8080`

**功能**:
- 查看系统状态
- 管理出口IP
- 查看统计信息
- 配置管理
- 用户管理

---

### 4. 订阅功能

#### 服务端生成订阅链接

```
http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN
```

#### 客户端使用订阅

```bash
./client -subscribe "http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN"
```

---

## 高级功能

### 1. IP健康检查

健康检查自动运行，无需手动配置。

**查看健康状态**:
```bash
curl -u admin:password http://localhost:8080/api/health
```

**配置检查间隔**:
```yaml
health_check:
  enabled: true
  interval: "30s"  # 检查间隔
  timeout: "5s"    # 超时时间
```

---

### 2. 负载均衡

#### 按连接数负载均衡

```yaml
strategy:
  type: "load_balanced"
  config:
    method: "connections"
```

系统会自动选择连接数最少的IP。

#### 按流量负载均衡

```yaml
strategy:
  type: "load_balanced"
  config:
    method: "traffic"
```

系统会自动选择流量最少的IP。

---

### 3. 多用户认证

#### 添加用户

**通过Web界面**: 访问用户管理页面

**通过API**:
```bash
curl -X POST -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user1",
    "password": "password123",
    "rate_limit": 1048576,
    "allowed_ips": ["0.0.0.0/0"]
  }' \
  http://localhost:8080/api/users
```

#### 用户配置

- `rate_limit`: 速率限制（字节/秒）
- `allowed_ips`: IP白名单（CIDR格式）

---

### 4. DDoS防护

#### 配置防护

```yaml
security:
  ddos:
    enabled: true
    max_connections_per_ip: 10      # 每IP最大连接数
    connection_rate_limit: 5         # 每秒连接数限制
    block_duration: "5m"             # 阻止持续时间
```

#### 查看被阻止的IP

```bash
curl -u admin:password http://localhost:8080/api/security/ddos
```

#### 解除阻止

```bash
curl -X POST -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"ip": "1.2.3.4"}' \
  http://localhost:8080/api/security/unblock
```

---

### 5. IP过滤

#### 配置白名单

```yaml
security:
  ip_filter:
    enabled: true
    whitelist:
      - "192.168.1.0/24"
      - "10.0.0.0/8"
    blacklist: []
```

#### 配置黑名单

```yaml
security:
  ip_filter:
    enabled: true
    whitelist: []
    blacklist:
      - "1.2.3.4/32"
      - "5.6.7.0/24"
```

---

### 6. 配置热更新

修改配置文件后，系统会自动检测并重新加载配置，无需重启服务。

**支持热更新的配置**:
- 出口IP列表
- IP选择策略
- 用户配置
- 安全配置

---

### 7. 监控和统计

#### 查看实时统计

```bash
curl -u admin:password http://localhost:8080/api/stats
```

**统计信息包括**:
- 总连接数
- 活跃连接数
- 流量统计（上行/下行）
- 按IP的详细统计
- 平均延迟

---

## 故障排查

### 问题1: 客户端无法连接服务端

**检查**:
1. 服务端是否运行: `ps aux | grep server`
2. 端口是否监听: `netstat -tlnp | grep 8443`
3. 防火墙规则: `iptables -L`
4. 认证密钥是否匹配

**解决**:
```bash
# 检查服务端日志
tail -f server.log

# 测试端口连通性
telnet YOUR_SERVER 8443
```

---

### 问题2: SNAT不工作

**检查**:
1. 是否有root权限: `sudo ./server`
2. iptables规则: `sudo iptables -t nat -L`
3. 路由规则: `ip rule list`
4. IP是否已绑定: `ip addr show`

**解决**:
```bash
# 检查路由规则
sudo ip rule list
sudo ip route show table 100

# 手动测试SNAT
sudo iptables -t nat -A OUTPUT -j SNAT --to-source 1.2.3.4
```

---

### 问题3: IP健康检查失败

**检查**:
1. IP是否可达: `ping 1.2.3.4`
2. 测试端口: `telnet 1.2.3.4 80`
3. 网关配置是否正确

**解决**:
```bash
# 测试IP连通性
ping -c 3 1.2.3.4
curl -I http://1.2.3.4
```

---

### 问题4: 性能问题

**检查**:
1. CPU使用率: `top`
2. 内存使用: `free -h`
3. 网络流量: `iftop`
4. 连接数: `netstat -an | grep ESTABLISHED | wc -l`

**优化**:
- 使用AES-GCM加密（已默认启用）
- 调整buffer大小
- 增加系统文件描述符限制
- 使用Docker部署

---

### 问题5: Docker容器无法使用SNAT

**原因**: Docker容器需要特殊权限

**解决**:
```yaml
# docker-compose.yml
services:
  multiexit-proxy:
    cap_add:
      - NET_ADMIN
      - SYS_ADMIN
    network_mode: host  # 使用host网络模式
```

---

## 最佳实践

1. **安全配置**:
   - 使用强密码
   - 启用IP白名单
   - 启用DDoS防护
   - 定期更新密钥

2. **性能优化**:
   - 使用AES-GCM加密
   - 启用健康检查
   - 使用负载均衡策略
   - 监控系统资源

3. **高可用**:
   - 配置多个出口IP
   - 启用健康检查自动切换
   - 定期备份配置
   - 监控日志

4. **运维建议**:
   - 使用Docker部署
   - 配置日志轮转
   - 设置监控告警
   - 定期性能测试

---

## 示例场景

### 场景1: 多IP轮换

```yaml
exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"
  - "9.10.11.12"

strategy:
  type: "round_robin"
```

每个新连接会轮流使用不同的IP。

### 场景2: 按目标地址分配

```yaml
strategy:
  type: "destination_based"
```

访问相同目标时使用相同的出口IP。

### 场景3: 负载均衡

```yaml
strategy:
  type: "load_balanced"
  config:
    method: "connections"
```

自动选择连接数最少的IP。

---

## 更多资源

- [API文档](API.md)
- [设计文档](../DESIGN.md)
- [技术规范](../TECHNICAL_SPEC.md)
- [性能测试结果](../PERFORMANCE_TEST_RESULTS.md)

