# MultiExit Proxy API 文档

## Web管理API

### 基础信息

- **Base URL**: `http://YOUR_SERVER:8080`
- **认证**: Basic Auth (用户名/密码)
- **Content-Type**: `application/json`

---

## API端点

### 1. 系统状态

#### GET /api/status

获取系统状态

**响应示例**:
```json
{
  "status": "running",
  "uptime": 3600,
  "version": "2.0.0"
}
```

---

### 2. IP管理

#### GET /api/ips

获取所有出口IP列表

**响应示例**:
```json
{
  "ips": [
    {
      "ip": "1.2.3.4",
      "status": "healthy",
      "connections": 10,
      "bytes_up": 1048576,
      "bytes_down": 2097152
    }
  ]
}
```

#### POST /api/ips

添加新的出口IP

**请求体**:
```json
{
  "ip": "5.6.7.8",
  "gateway": "1.2.3.1",
  "interface": "eth0"
}
```

---

### 3. 配置管理

#### GET /api/config

获取当前配置

**响应示例**:
```json
{
  "listen_addr": "0.0.0.0:8443",
  "strategy": "round_robin",
  "exit_ips": ["1.2.3.4", "5.6.7.8"],
  "auth_key": "***"
}
```

#### PUT /api/config

更新配置

**请求体**:
```json
{
  "strategy": "load_balanced",
  "exit_ips": ["1.2.3.4", "5.6.7.8"]
}
```

---

### 4. 统计信息

#### GET /api/stats

获取统计信息

**响应示例**:
```json
{
  "total_connections": 1000,
  "active_connections": 50,
  "bytes_transferred": 1073741824,
  "bytes_up": 536870912,
  "bytes_down": 536870912,
  "ip_stats": {
    "1.2.3.4": {
      "connections": 25,
      "active": 10,
      "bytes_up": 268435456,
      "bytes_down": 268435456,
      "avg_latency_ms": 50
    }
  }
}
```

---

### 5. 健康检查

#### GET /api/health

获取IP健康状态

**响应示例**:
```json
{
  "total": 3,
  "healthy": 2,
  "failed": 1,
  "healthy_ips": ["1.2.3.4", "5.6.7.8"],
  "failed_ips": ["9.10.11.12"]
}
```

---

### 6. 订阅管理

#### GET /api/subscribe

获取订阅链接

**查询参数**:
- `token`: 订阅令牌

**响应**: 返回客户端配置JSON或订阅URL

---

### 7. 用户管理

#### GET /api/users

列出所有用户

**响应示例**:
```json
{
  "users": ["user1", "user2"]
}
```

#### POST /api/users

创建新用户

**请求体**:
```json
{
  "username": "user1",
  "password": "password123",
  "rate_limit": 1048576,
  "allowed_ips": ["0.0.0.0/0"]
}
```

#### DELETE /api/users/{username}

删除用户

---

### 8. DDoS防护

#### GET /api/security/ddos

获取DDoS防护状态

**响应示例**:
```json
{
  "blocked_ips": 5,
  "max_connections": 10,
  "rate_limit": 5,
  "block_duration": "5m"
}
```

#### POST /api/security/unblock

解除IP阻止

**请求体**:
```json
{
  "ip": "1.2.3.4"
}
```

---

## 错误响应

所有API错误使用标准HTTP状态码：

```json
{
  "error": "错误描述",
  "code": "ERROR_CODE"
}
```

**常见错误码**:
- `INVALID_REQUEST`: 请求无效
- `UNAUTHORIZED`: 未授权
- `NOT_FOUND`: 资源不存在
- `INTERNAL_ERROR`: 服务器内部错误

---

## 认证

使用HTTP Basic认证：

```bash
curl -u admin:password http://localhost:8080/api/status
```

---

## 示例

### 获取统计信息
```bash
curl -u admin:password http://localhost:8080/api/stats
```

### 添加IP
```bash
curl -X POST -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"ip":"5.6.7.8","gateway":"1.2.3.1","interface":"eth0"}' \
  http://localhost:8080/api/ips
```

### 更新配置
```bash
curl -X PUT -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"strategy":"load_balanced"}' \
  http://localhost:8080/api/config
```

