# 技术实现规范

## 一、协议详细设计

### 1.1 协议栈

```
┌─────────────────────────────────┐
│     应用层数据 (TCP/UDP)        │
├─────────────────────────────────┤
│   SOCKS5/HTTP代理协议           │
├─────────────────────────────────┤
│   自定义协议封装                │
│   - 目标地址                    │
│   - 数据包                      │
├─────────────────────────────────┤
│   AEAD加密 (ChaCha20-Poly1305)  │
├─────────────────────────────────┤
│   TLS 1.3 加密层                │
│   - SNI伪装                     │
│   - 标准TLS握手                 │
├─────────────────────────────────┤
│   TCP传输                       │
└─────────────────────────────────┘
```

### 1.2 自定义协议格式

#### 握手消息（32字节）
```
┌──────┬──────┬──────┬─────────────┬─────────────┬──────┐
│ 版本 │ 方法 │ 保留 │   随机数    │   时间戳    │ HMAC │
│ 1B   │ 1B   │ 2B   │    16B      │     8B      │  4B  │
└──────┴──────┴──────┴─────────────┴─────────────┴──────┘
```

#### 连接请求（变长）
```
┌──────┬──────┬──────┬──────────────┬──────────────┐
│ 类型 │ 地址类型 │ 地址长度 │   目标地址    │   目标端口   │
│ 1B   │   1B   │   1B    │     变长      │     2B      │
├──────┴──────┴──────┴──────────────┴──────────────┤
│             加密数据 (使用AEAD)                    │
├──────────────────────────────────────────────────┤
│                    HMAC (4B)                      │
└──────────────────────────────────────────────────┘
```

#### 数据包格式
```
┌──────┬──────┬────────────┬──────────────┐
│ 类型 │ 流ID │   长度     │   加密数据   │
│ 1B   │  4B  │    2B      │     变长     │
├──────┴──────┴────────────┴──────────────┤
│              HMAC (4B)                   │
└──────────────────────────────────────────┘
```

### 1.3 加密算法细节

#### 密钥派生
```
主密钥 (32字节) 
    ↓
HKDF-SHA256
    ↓
┌──────────┬──────────┬──────────┐
│  加密密钥 │  认证密钥 │  IV种子  │
│   32B    │   32B    │   16B    │
└──────────┴──────────┴──────────┘
```

#### AEAD加密流程
```
明文数据
    ↓
ChaCha20-Poly1305加密
    ↓
密文 + Tag
    ↓
附加HMAC-SHA256
```

## 二、SNAT实现详细方案

### 2.1 iptables规则管理

#### 标记策略（使用CONNMARK）
```bash
# 为每个出口IP创建标记
# IP1 -> mark 1 -> table 100
# IP2 -> mark 2 -> table 101
# IP3 -> mark 3 -> table 102
```

#### 规则创建流程
```go
// 伪代码
func SetupSNATRules(ips []string) {
    for i, ip := range ips {
        mark := i + 1
        table := 100 + i
        
        // 创建路由表
        exec("ip route add default via <gateway> table {table} src {ip}")
        
        // 创建路由规则
        exec("ip rule add fwmark {mark} table {table}")
        
        // 创建SNAT规则
        exec("iptables -t nat -A OUTPUT -m mark --mark {mark} -j SNAT --to-source {ip}")
    }
}
```

### 2.2 策略路由实现

#### 按轮询分配
```go
type RoundRobinSelector struct {
    ips []net.IP
    current int
    mu sync.Mutex
}

func (r *RoundRobinSelector) SelectIP() net.IP {
    r.mu.Lock()
    defer r.mu.Unlock()
    ip := r.ips[r.current]
    r.current = (r.current + 1) % len(r.ips)
    return ip
}
```

#### 按端口分配
```go
type PortBasedSelector struct {
    portRanges map[string]net.IP
}

func (p *PortBasedSelector) SelectIP(dstPort int) net.IP {
    // 根据端口范围选择IP
    if dstPort < 32768 {
        return p.portRanges["low"]
    }
    return p.portRanges["high"]
}
```

#### 按目标地址分配
```go
type DestinationBasedSelector struct {
    ips []net.IP
}

func (d *DestinationBasedSelector) SelectIP(dstAddr string) net.IP {
    hash := sha256.Sum256([]byte(dstAddr))
    index := int(binary.BigEndian.Uint64(hash[:8])) % len(d.ips)
    return d.ips[index]
}
```

### 2.3 连接标记

在建立到目标服务的连接时，需要标记连接：
```go
func MarkConnection(conn net.Conn, mark int) error {
    // 获取文件描述符
    tcpConn := conn.(*net.TCPConn)
    file, _ := tcpConn.File()
    fd := file.Fd()
    
    // 设置SO_MARK
    err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
    return err
}
```

## 三、流量混淆技术

### 3.1 TLS SNI伪装
```go
// 使用常见域名的SNI
fakeSNIs := []string{
    "cloudflare.com",
    "google.com",
    "github.com",
    "microsoft.com",
}

func GetRandomSNI() string {
    return fakeSNIs[rand.Intn(len(fakeSNIs))]
}
```

### 3.2 包大小混淆
```go
func AddPadding(data []byte) []byte {
    // 随机padding，使包大小接近常见HTTP包大小
    paddingSize := rand.Intn(128) // 0-127字节
    padding := make([]byte, paddingSize)
    rand.Read(padding)
    return append(data, padding...)
}
```

### 3.3 时间混淆
```go
func AddRandomDelay() {
    // 随机延迟，模拟正常网络延迟
    delay := time.Duration(rand.Intn(100)) * time.Millisecond
    time.Sleep(delay)
}
```

## 四、项目结构

```
multiexit-proxy/
├── cmd/
│   ├── client/           # 客户端入口
│   │   └── main.go
│   └── server/           # 服务端入口
│       └── main.go
├── internal/
│   ├── protocol/         # 协议实现
│   │   ├── encoder.go    # 编码/解码
│   │   ├── encrypt.go    # 加密/解密
│   │   └── message.go    # 消息格式
│   ├── proxy/            # 代理功能
│   │   ├── client.go     # 客户端代理
│   │   └── server.go     # 服务端代理
│   ├── snat/             # SNAT管理
│   │   ├── iptables.go   # iptables规则
│   │   ├── selector.go   # IP选择策略
│   │   └── routing.go    # 路由管理
│   ├── transport/        # 传输层
│   │   ├── tls.go        # TLS封装
│   │   └── obfuscate.go  # 流量混淆
│   └── config/           # 配置管理
│       └── config.go
├── pkg/
│   └── socks5/           # SOCKS5协议
│       └── server.go
├── configs/
│   ├── server.yaml.example
│   └── client.json.example
├── scripts/
│   └── setup.sh          # 服务端初始化脚本
├── go.mod
├── go.sum
├── README.md
├── DESIGN.md
└── TECHNICAL_SPEC.md
```

## 五、关键依赖包

```go
// go.mod
module multiexit-proxy

go 1.21

require (
    // TLS
    github.com/refraction-networking/utls v1.5.4  // 支持自定义TLS配置
    
    // 加密
    golang.org/x/crypto v0.17.0  // ChaCha20-Poly1305
    
    // 网络工具
    github.com/google/gopacket v1.1.19  // 数据包处理
    
    // 配置
    gopkg.in/yaml.v3 v3.0.1  // YAML解析
    
    // 日志
    github.com/sirupsen/logrus v1.9.3
)
```

## 六、配置示例

### 服务端配置 (server.yaml)
```yaml
server:
  listen: ":443"
  tls:
    cert: "/path/to/cert.pem"
    key: "/path/to/key.pem"
    sni_fake: true
    fake_snis:
      - "cloudflare.com"
      - "google.com"

auth:
  method: "psk"
  key: "your-secret-key-here"

exit_ips:
  - "1.2.3.4"
  - "5.6.7.8"
  - "9.10.11.12"

strategy:
  type: "round_robin"  # round_robin, port_based, destination_based
  # 如果type是port_based:
  # port_ranges:
  #   - range: "0-32767"
  #     ip: "1.2.3.4"
  #   - range: "32768-65535"
  #     ip: "5.6.7.8"

snat:
  enabled: true
  gateway: "192.168.1.1"  # 网关地址
  interface: "eth0"

logging:
  level: "info"
  file: "/var/log/multiexit-proxy.log"
```

### 客户端配置 (client.json)
```json
{
  "server": {
    "address": "your-server.com:443",
    "sni": "cloudflare.com"
  },
  "auth": {
    "key": "your-secret-key-here"
  },
  "local": {
    "socks5": "127.0.0.1:1080",
    "http": "127.0.0.1:8080"
  },
  "logging": {
    "level": "info"
  }
}
```

## 七、安全性增强

### 7.1 防重放攻击
```go
type ReplayProtection struct {
    seen map[string]time.Time
    mu   sync.RWMutex
}

func (r *ReplayProtection) Check(nonce []byte, timestamp int64) bool {
    // 检查时间戳是否在有效窗口内
    now := time.Now().Unix()
    if abs(now - timestamp) > 300 { // 5分钟窗口
        return false
    }
    
    // 检查nonce是否已使用
    key := string(nonce)
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.seen[key]; exists {
        return false
    }
    
    r.seen[key] = time.Now()
    return true
}
```

### 7.2 速率限制
```go
type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(rps int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(rps), rps),
    }
}
```

## 八、性能优化

### 8.1 连接池
```go
type ConnectionPool struct {
    connections chan net.Conn
    factory     func() (net.Conn, error)
}

func (p *ConnectionPool) Get() (net.Conn, error) {
    select {
    case conn := <-p.connections:
        return conn, nil
    default:
        return p.factory()
    }
}
```

### 8.2 零拷贝转发
```go
func CopyZeroCopy(dst, src net.Conn) error {
    dstTCP := dst.(*net.TCPConn)
    srcTCP := src.(*net.TCPConn)
    
    dstFd, _ := dstTCP.File()
    srcFd, _ := srcTCP.File()
    
    for {
        written, err := syscall.Splice(
            int(srcFd.Fd()), nil,
            int(dstFd.Fd()), nil,
            32*1024, 0,
        )
        if written == 0 || err != nil {
            break
        }
    }
    return nil
}
```



