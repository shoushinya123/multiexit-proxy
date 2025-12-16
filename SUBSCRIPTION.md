# 订阅功能使用指南

## 功能概述

订阅功能允许客户端通过一个订阅链接自动获取服务器配置，无需手动配置。

## 服务端：生成订阅链接

### 方法1: 通过Web管理界面

1. 访问Web管理界面: `http://YOUR_SERVER:8080`
2. 登录后访问: `http://YOUR_SERVER:8080/api/subscription/link`
3. 获取订阅链接和二维码

### 方法2: 直接使用API

```bash
# 使用Basic Auth认证
curl -u admin:password http://YOUR_SERVER:8080/api/subscription/link
```

返回示例:
```json
{
  "token": "your-auth-key",
  "link": "http://YOUR_SERVER:8080/api/subscribe?token=your-auth-key",
  "qr_code": "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=..."
}
```

## 客户端：使用订阅链接

### 方法1: 命令行参数

```bash
# 直接使用订阅链接启动客户端
./client -subscribe "http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN"
```

### 方法2: 保存配置后使用

```bash
# 使用订阅链接并保存到配置文件
./client -subscribe "http://YOUR_SERVER:8080/api/subscribe?token=YOUR_TOKEN" -config client.json

# 之后可以直接使用配置文件
./client -config client.json
```

## 订阅链接格式

```
http://YOUR_SERVER:PORT/api/subscribe?token=YOUR_TOKEN
```

- `YOUR_SERVER`: 服务器IP或域名
- `PORT`: Web管理界面端口（默认8080）
- `YOUR_TOKEN`: 订阅token（默认使用认证密钥）

## 订阅配置内容

订阅链接返回base64编码的JSON配置，包含：

- 服务器地址和端口
- 认证密钥
- SNI设置
- 出口IP列表
- 分配策略
- 有效期

## 测试订阅

### 1. 获取订阅链接

```bash
# 在服务器上
curl -u admin:admin123 http://localhost:8081/api/subscription/link
```

### 2. 测试订阅内容

```bash
# 获取订阅配置（base64编码）
curl "http://localhost:8081/api/subscribe?token=YOUR_TOKEN"

# 解码查看内容
curl "http://localhost:8081/api/subscribe?token=YOUR_TOKEN" | base64 -d | jq
```

### 3. 客户端使用订阅

```bash
# 使用订阅链接启动客户端
./client -subscribe "http://localhost:8081/api/subscribe?token=YOUR_TOKEN"
```

## 安全建议

1. **使用强token**: 修改默认token为随机生成的强密钥
2. **HTTPS**: 在生产环境使用HTTPS保护订阅链接
3. **定期更换**: 定期更换订阅token
4. **访问控制**: 限制订阅链接的访问频率

## 示例流程

### 完整示例

```bash
# 1. 服务端生成订阅链接
TOKEN=$(curl -s -u admin:admin123 http://localhost:8081/api/subscription/link | jq -r '.token')
LINK=$(curl -s -u admin:admin123 http://localhost:8081/api/subscription/link | jq -r '.link')

echo "订阅链接: $LINK"

# 2. 客户端使用订阅链接
./client -subscribe "$LINK"

# 3. 或保存配置后使用
./client -subscribe "$LINK" -config my-client.json
./client -config my-client.json
```

## 常见问题

### Q: 订阅链接无效？
A: 检查token是否正确，确保服务端配置的auth.key与token匹配

### Q: 客户端无法解析订阅？
A: 确保订阅链接可访问，检查网络连接和防火墙设置

### Q: 订阅过期？
A: 默认有效期为365天，可以重新获取订阅链接

