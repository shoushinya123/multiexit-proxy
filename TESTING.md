# 客户端代理测试指南

## 快速测试

### 方法1: 使用自动化测试脚本（推荐）

```bash
# 1. 确保服务端正在运行
ps aux | grep server

# 2. 启动客户端（如果没有运行）
./client -config configs/client.json

# 3. 运行测试脚本
./test-client.sh
```

### 方法2: 手动测试

#### 1. 启动客户端

```bash
./client -config configs/client.json
```

#### 2. 测试SOCKS5代理

```bash
# 测试HTTP网站
curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/ip

# 测试HTTPS网站
curl --socks5-hostname 127.0.0.1:1080 https://ifconfig.me

# 测试访问速度
time curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/get
```

#### 3. 测试HTTP代理

```bash
# 测试HTTP网站
curl --proxy http://127.0.0.1:8080 http://httpbin.org/ip

# 测试HTTPS网站（如果支持）
curl --proxy http://127.0.0.1:8080 https://ifconfig.me
```

#### 4. 在浏览器中使用代理

**macOS系统设置:**
1. 系统偏好设置 → 网络
2. 选择网络连接 → 高级 → 代理
3. 勾选"SOCKS代理"
4. 服务器: `127.0.0.1`，端口: `1080`

**或使用浏览器扩展:**
- Chrome: SwitchyOmega
- Firefox: FoxyProxy

#### 5. 设置环境变量

```bash
# SOCKS5代理
export ALL_PROXY=socks5://127.0.0.1:1080

# HTTP代理
export http_proxy=http://127.0.0.1:8080
export https_proxy=http://127.0.0.1:8080

# 测试
curl http://httpbin.org/ip
```

## 验证代理是否工作

### 检查1: 查看客户端日志

```bash
tail -f client.log
```

应该看到类似输出：
```
Client starting, listening on 127.0.0.1:1080
```

### 检查2: 查看服务端日志

```bash
tail -f multiexit-proxy.log
```

应该看到客户端连接记录。

### 检查3: 检查端口监听

```bash
# 检查客户端监听端口
lsof -i :1080
lsof -i :8080

# 检查服务端监听端口
lsof -i :8443
```

### 检查4: 测试连通性

```bash
# 测试客户端到服务端的连接
telnet localhost 8443

# 测试SOCKS5代理
nc -zv 127.0.0.1 1080
```

## 常见问题排查

### 问题1: 客户端无法连接到服务端

**检查:**
1. 服务端是否运行: `ps aux | grep server`
2. 端口是否正确: 默认8443
3. 认证密钥是否匹配: 检查 `configs/server.yaml` 和 `configs/client.json`

### 问题2: 代理连接失败

**检查:**
1. 客户端是否运行: `ps aux | grep client`
2. 端口是否被占用: `lsof -i :1080`
3. 防火墙是否阻止: macOS防火墙设置

### 问题3: 访问网站失败

**检查:**
1. 服务端日志查看错误信息
2. 客户端日志查看连接状态
3. 测试简单的HTTP网站: `curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/get`

## 性能测试

```bash
# 测试响应时间
time curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/get

# 测试下载速度
curl --socks5-hostname 127.0.0.1:1080 -o /dev/null http://speedtest.tele2.net/100MB.zip

# 测试并发连接
for i in {1..10}; do
    curl --socks5-hostname 127.0.0.1:1080 http://httpbin.org/get &
done
wait
```

## 停止服务

```bash
# 停止客户端
./stop-client.sh

# 停止服务端
./stop-server.sh
```

