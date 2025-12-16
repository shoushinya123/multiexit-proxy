# Trojan协议支持

## 概述

已添加Trojan协议支持，可以将隧道封装为Trojan协议。Trojan使用TLS加密，流量特征更接近HTTPS，抗检测性更好。

## Trojan协议特点

1. **TLS加密**：使用标准TLS/SSL加密
2. **密码认证**：使用SHA224哈希的密码（56字节）
3. **流量伪装**：数据包看起来像正常的HTTPS流量
4. **简单高效**：协议简单，性能优秀

## 协议格式

```
[56字节密码SHA224哈希] + [命令(1字节)] + [地址类型(1字节)] + [地址] + [端口(2字节)] + [CRLF(2字节)] + [数据]
```

## 配置示例

### 服务端配置 (server.yaml)

```yaml
server:
  listen: ":443"
  tls:
    cert: "/path/to/cert.pem"
    key: "/path/to/key.pem"

trojan:
  enabled: true
  password: "your-trojan-password"

exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"

strategy:
  type: "round_robin"

snat:
  enabled: true
  gateway: "192.168.1.1"
  interface: "eth0"
```

### 客户端配置 (client.json)

```json
{
  "server": {
    "address": "your-server.com:443",
    "sni": "your-server.com"
  },
  "auth": {
    "key": "your-trojan-password"
  },
  "local": {
    "socks5": "127.0.0.1:1080"
  }
}
```

## 使用方法

### 服务端

```bash
# 编译Trojan服务端
go build -o trojan-server ./cmd/trojan-server

# 运行
sudo ./trojan-server -config configs/server.yaml
```

### 客户端

```bash
# 编译Trojan客户端
go build -o trojan-client ./cmd/trojan-client

# 运行
./trojan-client -config configs/client.json
```

## 兼容性

- ✅ 兼容标准Trojan协议
- ✅ 可以与Trojan客户端/服务端互操作
- ✅ 支持TCP代理
- ⚠️ UDP代理暂未实现

## 安全建议

1. **使用强密码**：设置复杂的Trojan密码
2. **TLS证书**：使用有效的TLS证书（或自签名）
3. **SNI设置**：使用真实域名的SNI
4. **定期更换**：定期更换密码

## 与现有协议的对比

| 特性 | 自定义协议 | Trojan协议 |
|------|-----------|-----------|
| 加密 | TLS + AEAD | TLS |
| 认证 | PSK | 密码SHA224 |
| 流量特征 | 自定义 | 更像HTTPS |
| 兼容性 | 需要专用客户端 | 兼容标准Trojan |
| 性能 | 优秀 | 优秀 |

## 实现细节

### 协议头

Trojan协议使用密码的SHA224哈希（56字节）作为协议头，用于身份验证。

### 连接请求格式

```
命令(1字节) + 地址类型(1字节) + 地址 + 端口(2字节) + CRLF(2字节)
```

- 命令：1=CONNECT, 3=UDP
- 地址类型：1=IPv4, 3=Domain, 4=IPv6
- CRLF：\r\n（必需）

### 数据转发

Trojan协议在握手后直接转发原始数据，无需额外的协议封装。

## 参考

- [Trojan Protocol](https://trojan-gfw.github.io/trojan/protocol)
- Trojan是一个类似Shadowsocks的代理协议，使用TLS伪装

