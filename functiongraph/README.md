# FunctionGraph部署指南

## 概述

将Web管理界面和订阅API迁移到华为云FunctionGraph，TCP代理服务保留在ECS。

## 架构设计

```
┌─────────────────────────────────┐
│  华为云FunctionGraph             │
│  ┌───────────────────────────┐  │
│  │ HTTP函数 (Web管理+订阅API) │  │
│  └───────────────────────────┘  │
└──────────────┬────────────────────┘
               │ HTTP API调用
┌──────────────▼────────────────────┐
│  华为云ECS                         │
│  ┌───────────────────────────┐  │
│  │ TCP代理服务 (8443端口)     │  │
│  └───────────────────────────┘  │
└───────────────────────────────────┘
```

## 部署步骤

### 1. 准备FunctionGraph函数代码

```bash
cd functiongraph
go mod init functiongraph-web
go get github.com/gorilla/mux
go build -o handler handler.go
chmod +x bootstrap
```

### 2. 创建函数包

```bash
zip -r function.zip bootstrap handler static/
```

### 3. 在华为云控制台创建函数

1. 登录华为云控制台
2. 进入FunctionGraph服务
3. 创建函数：
   - **函数类型**：HTTP函数
   - **运行时**：Go 1.x
   - **入口函数**：Handler
   - **上传函数包**：function.zip

### 4. 配置环境变量

在函数配置中添加环境变量：
- `AUTH_KEY`: 认证密钥
- `SERVER_ADDR`: TCP代理服务器地址
- `WEB_USERNAME`: Web管理用户名
- `WEB_PASSWORD`: Web管理密码

### 5. 配置APIG触发器

1. 创建API Gateway触发器
2. 配置路由规则
3. 设置认证方式

### 6. 配置网络

- **VPC配置**：如果需要访问ECS上的TCP服务
- **公网访问**：配置固定公网IP（如果需要）

## 函数代码结构

```
functiongraph/
├── bootstrap          # 入口文件（必须）
├── handler.go         # HTTP处理函数
├── handler            # 编译后的可执行文件
├── go.mod             # Go模块定义
├── static/            # 静态文件（可选）
└── README.md          # 说明文档
```

## 函数接口定义

根据[华为云文档](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)：

```go
func Handler(w http.ResponseWriter, r *http.Request)
```

- `w`: HTTP响应写入器
- `r`: HTTP请求对象

## API端点

### 公开API（无需认证）

- `GET /api/subscribe?token=xxx` - 获取订阅配置

### 管理API（需要Basic Auth）

- `GET /api/subscription/link` - 生成订阅链接
- `GET /api/config` - 获取配置
- `POST /api/config` - 更新配置
- `GET /api/ips` - 获取IP列表
- `GET /api/status` - 获取状态

## 配置存储方案

### 方案1: 环境变量（简单）

适合配置较少的情况。

### 方案2: OBS对象存储（推荐）

将配置文件存储在OBS，函数运行时读取。

### 方案3: DCS Redis（高性能）

适合需要频繁读取配置的场景。

## 注意事项

1. **函数超时**：默认15秒，HTTP函数最长30秒
2. **内存限制**：根据需求配置内存大小
3. **并发限制**：注意函数并发数限制
4. **日志输出**：使用标准输出，自动收集到LTS

## 成本优化

1. **按需调用**：FunctionGraph按调用次数计费
2. **冷启动优化**：使用预热功能
3. **缓存配置**：减少OBS读取次数

## 测试

```bash
# 测试订阅API
curl "https://your-function-url/api/subscribe?token=YOUR_TOKEN"

# 测试管理API
curl -u admin:password "https://your-function-url/api/subscription/link"
```

## 参考文档

- [Go函数开发概述](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)
- [开发Go事件函数](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0441.html)
- [使用Go开发HTTP函数](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0603.html)
- [将代码设计为云函数模式](https://support.huaweicloud.com/devg-functiongraph/functiongraph_01_0406.html)



