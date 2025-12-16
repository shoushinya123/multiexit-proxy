# 使用说明

## 快速开始

### 1. 服务端部署

#### 1.1 环境准备

```bash
# 安装依赖
sudo apt-get update
sudo apt-get install -y iptables iproute2

# 确保有多个公网IP绑定到网络接口
ip addr show
```

#### 1.2 配置服务端

复制并编辑配置文件：

```bash
cp configs/server.yaml.example configs/server.yaml
vim configs/server.yaml
```

重要配置项：
- `exit_ips`: 配置你的多个公网IP地址
- `snat.gateway`: 配置网关地址
- `snat.interface`: 配置网络接口名称（如eth0）
- `auth.key`: 设置强密码

#### 1.3 生成TLS证书（可选）

如果使用自签名证书：

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

#### 1.4 运行服务端

```bash
# 编译
go build -o multiexit-proxy-server ./cmd/server

# 运行（需要root权限）
sudo ./multiexit-proxy-server -config configs/server.yaml
```

### 2. 客户端部署

#### 2.1 配置客户端

复制并编辑配置文件：

```bash
cp configs/client.json.example configs/client.json
vim configs/client.json
```

配置项：
- `server.address`: 服务端地址和端口
- `server.sni`: SNI伪装域名（可选）
- `auth.key`: 与服务端相同的密钥
- `local.socks5`: 本地SOCKS5代理地址

#### 2.2 运行客户端

```bash
# 编译
go build -o multiexit-proxy-client ./cmd/client

# 运行
./multiexit-proxy-client -config configs/client.json
```

#### 2.3 配置应用使用代理

**Linux/macOS:**
```bash
export http_proxy=http://127.0.0.1:8080
export https_proxy=http://127.0.0.1:8080
export ALL_PROXY=socks5://127.0.0.1:1080
```

**Windows:**
```powershell
$env:http_proxy="http://127.0.0.1:8080"
$env:https_proxy="http://127.0.0.1:8080"
```

**浏览器配置:**
- SOCKS5代理: `127.0.0.1:1080`
- HTTP代理: `127.0.0.1:8080`

## IP分配策略

### 轮询策略（Round Robin）

默认策略，按连接顺序轮流使用各个IP：

```yaml
strategy:
  type: "round_robin"
```

### 按端口分配

特定端口范围映射到特定IP：

```yaml
strategy:
  type: "port_based"
  port_ranges:
    - range: "0-32767"
      ip: "1.2.3.4"
    - range: "32768-65535"
      ip: "5.6.7.8"
```

### 按目标地址分配

相同目标地址使用相同出口IP：

```yaml
strategy:
  type: "destination_based"
```

## 故障排查

### 服务端无法启动

1. **检查权限**: 确保使用root权限运行
2. **检查端口**: 确保443端口未被占用
3. **检查配置**: 验证配置文件格式正确
4. **查看日志**: 检查 `/var/log/multiexit-proxy.log`

### SNAT不生效

1. **检查IP绑定**: `ip addr show`
2. **检查路由规则**: `ip rule show`
3. **检查iptables规则**: `iptables -t nat -L -n`
4. **检查网关配置**: 确保gateway地址正确

### 客户端无法连接

1. **检查服务端地址**: 确保地址和端口正确
2. **检查密钥**: 确保客户端和服务端密钥一致
3. **检查防火墙**: 确保服务端443端口开放
4. **检查网络**: 确保可以访问服务端

## 性能优化

### 调整缓冲区大小

在代码中修改 `copyData` 函数的缓冲区大小：

```go
buf := make([]byte, 64*1024) // 增加到64KB
```

### 启用连接复用

未来版本将支持连接池和连接复用。

## 安全建议

1. **使用强密码**: 设置复杂的auth.key
2. **定期更换密钥**: 定期更换认证密钥
3. **限制访问**: 使用防火墙限制服务端访问
4. **监控日志**: 定期检查日志文件
5. **更新软件**: 及时更新到最新版本

## 注意事项

⚠️ **重要提示**：

1. 本系统需要root权限才能配置iptables规则
2. 服务端需要绑定多个公网IP（EIP）
3. 确保防火墙规则允许相关端口
4. 遵守当地法律法规使用
5. 仅在Linux系统上支持SNAT功能（macOS/Windows不支持）

## 常见问题

**Q: 可以在macOS/Windows上运行服务端吗？**
A: 可以运行，但SNAT功能仅在Linux上支持。

**Q: 如何添加更多出口IP？**
A: 在配置文件的 `exit_ips` 中添加IP地址即可。

**Q: 支持UDP代理吗？**
A: 当前版本仅支持TCP，UDP支持将在未来版本中添加。

**Q: 如何查看每个IP的使用情况？**
A: 查看日志文件或使用 `iptables -t nat -L -n -v` 查看统计信息。



